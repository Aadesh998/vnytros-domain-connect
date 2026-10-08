package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mailforge/config"
	apiqueue "domain-connect-backend/internal/queue"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Queue names are defined once in internal/queue/topics.go and aliased here so
// the publisher and the api's consumers can never drift apart.
const (
	QueueCampaignSend  = apiqueue.QueueCampaignSend
	QueueCampaignTrack = apiqueue.QueueCampaignTrack
)

var (
	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
)

func Connect() error {
	if config.AppConfig.RabbitMQURL == "" {
		return fmt.Errorf("queue: RABITMQ is not configured")
	}

	mu.Lock()
	defer mu.Unlock()
	return dialLocked()
}

// dialLocked establishes conn/ch. Caller must hold mu.
func dialLocked() error {
	closeLocked()

	c, err := amqp.Dial(config.AppConfig.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("queue: dial: %w", err)
	}

	channel, err := c.Channel()
	if err != nil {
		_ = c.Close()
		return fmt.Errorf("queue: channel: %w", err)
	}

	for _, name := range []string{QueueCampaignSend, QueueCampaignTrack} {
		if _, err := channel.QueueDeclare(name, true, false, false, false, nil); err != nil {
			_ = channel.Close()
			_ = c.Close()
			return fmt.Errorf("queue: declare %q: %w", name, err)
		}
	}

	conn = c
	ch = channel
	log.Printf("queue: publisher ready")
	return nil
}

// closeLocked tears down conn/ch. Caller must hold mu.
func closeLocked() {
	if ch != nil {
		_ = ch.Close()
		ch = nil
	}
	if conn != nil {
		_ = conn.Close()
		conn = nil
	}
}

// Close shuts the publisher down.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	closeLocked()
}

// publish marshals payload onto queueName, redialling once if the connection
// went away while the instance was idle.
func publish(ctx context.Context, queueName string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue: marshal: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	for attempt := range 2 {
		if ch == nil || conn == nil || conn.IsClosed() {
			if dialErr := dialLocked(); dialErr != nil {
				return dialErr
			}
		}

		err = ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		})
		if err == nil {
			return nil
		}

		log.Printf("queue: publish to %q failed (attempt %d): %v", queueName, attempt+1, err)
		closeLocked()
	}

	return fmt.Errorf("queue: publish to %q: %w", queueName, err)
}

// PublishCampaignSend enqueues one batch of a campaign for the worker to send.
func PublishCampaignSend(ctx context.Context, job jobs.CampaignSendJob) error {
	return publish(ctx, QueueCampaignSend, job)
}

// PublishCampaignTrack enqueues one engagement event.
func PublishCampaignTrack(ctx context.Context, job jobs.CampaignTrackJob) error {
	return publish(ctx, QueueCampaignTrack, job)
}
