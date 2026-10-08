package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"domain-connect-backend/internal/queue"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/telemetry"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type HandlerFunc func(ctx context.Context, body []byte) error

type Register struct {
	qc             *queue.Client
	db             *gorm.DB
	domainRepo     repository.DomainRepository
	webhookLogRepo repository.WebhookEventLogRepository
	handlers       map[string]HandlerFunc
	mu             sync.RWMutex
}

func NewRegistry(
	qc *queue.Client,
	db *gorm.DB,
	domainRepo repository.DomainRepository,
	webhookLogRepo repository.WebhookEventLogRepository,
) *Register {
	return &Register{
		qc:             qc,
		db:             db,
		domainRepo:     domainRepo,
		webhookLogRepo: webhookLogRepo,
		handlers:       make(map[string]HandlerFunc),
	}
}

func (r *Register) Register(queueName string, fn HandlerFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if queueName == "" {
		return fmt.Errorf("worker: queue name is empty")
	}
	if fn == nil {
		return fmt.Errorf("worker: handler is nil for %q", queueName)
	}
	if _, ok := r.handlers[queueName]; ok {
		return fmt.Errorf("worker: handler already registered for %q", queueName)
	}

	r.handlers[queueName] = fn
	log.Printf("worker: registered handler for %q", queueName)
	return nil
}

func (r *Register) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	r.mu.RLock()
	for queueName, fn := range r.handlers {
		wg.Add(1)
		go func(qn string, h HandlerFunc) {
			defer wg.Done()
			r.consume(ctx, qn, h)
		}(queueName, fn)
	}
	r.mu.RUnlock()

	wg.Wait()
	return nil
}

func (r *Register) consume(ctx context.Context, qn string, fn HandlerFunc) {
	ch := r.qc.Channel()
	if ch == nil {
		log.Printf("worker[%s]: channel not open, aborting consumer", qn)
		return
	}

	msgs, err := ch.Consume(
		qn,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("worker[%s]: failed to consume: %v", qn, err)
		return
	}

	log.Printf("worker[%s]: consuming", qn)
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker[%s]: stopping consumer", qn)
			return

		case msg, ok := <-msgs:
			if !ok {
				log.Printf("worker[%s]: delivery channel closed", qn)
				return
			}
			r.dispatch(ctx, qn, fn, msg)
		}
	}
}

// dispatch runs one job. It is the single place every queue message passes
// through, so it is also where the worker's traces and metrics are recorded —
// the four outcomes below are exactly the `outcome` label values.
func (r *Register) dispatch(ctx context.Context, qn string, fn HandlerFunc, msg amqp.Delivery) {
	start := time.Now()

	ctx, span := telemetry.Tracer("worker").Start(ctx, "job "+qn,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", qn),
			attribute.Int("messaging.message.body.size", len(msg.Body)),
		),
	)
	defer span.End()

	// outcome is set on every path below, including the panic path, so the
	// counter can never silently miss a job.
	outcome := "panic"
	defer func() {
		telemetry.JobsTotal.WithLabelValues(qn, outcome).Inc()
		telemetry.JobDuration.WithLabelValues(qn).Observe(time.Since(start).Seconds())
	}()

	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("worker[%s]: PANIC: %v", qn, rec)
			span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", rec))
			span.SetAttributes(attribute.String("job.outcome", "panic"))
			_ = msg.Nack(false, false) // poison message → DLQ
		}
	}()

	err := fn(ctx, msg.Body)
	switch {
	case err == nil:
		outcome = "ack"
		span.SetStatus(codes.Ok, "")
		_ = msg.Ack(false)

	case errors.As(err, new(*ErrPermanent)):
		outcome = "dlq"
		log.Printf("worker[%s]: permanent failure (DLQ): %v", qn, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "permanent failure")
		_ = msg.Nack(false, false)

	default:
		outcome = "requeue"
		log.Printf("worker[%s]: retryable failure (requeue): %v", qn, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "retryable failure")
		_ = msg.Nack(false, true)
	}
	span.SetAttributes(attribute.String("job.outcome", outcome))
}
