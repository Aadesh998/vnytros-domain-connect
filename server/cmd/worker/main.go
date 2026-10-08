package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"
	"domain-connect-backend/internal/queue"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/telemetry"
	"domain-connect-backend/internal/worker"
)

func main() {
	config.LoadConfig()
	config.InitLogger()

	shutdownTracing := telemetry.Init(context.Background(), "vnytros-worker")
	defer shutdownTracing()
	// The worker has no HTTP server of its own, so this listener is the only
	// way to see job throughput, latency and retry churn from outside.
	shutdownMetrics := telemetry.Serve("9103")
	defer shutdownMetrics()

	db.InitDB()
	sqldb, err := db.DB.DB()
	if err != nil {
		log.Fatalf("worker: failed to access db handle: %v", err)
	}
	defer sqldb.Close()

	url := config.AppConfig.RabitMQ
	if url == "" {
		url = "amqp://admin:secret@localhost:5672/"
	}

	qc, err := queue.NewClient(url)
	if err != nil {
		log.Fatalf("worker: failed to connect to RabbitMQ: %v", err)
	}
	defer qc.Close()

	queue.SetPublisher(qc)

	for _, q := range []string{
		queue.QueueWebhookDispatch,
		queue.QueueEmailSend,
		queue.QueueVerifyDomain,
		queue.QueueCampaignSend,
		queue.QueueCampaignTrack,
	} {
		if err := qc.DeclareQueue(q); err != nil {
			log.Fatalf("worker: declare %q: %v", q, err)
		}
	}

	domainRepo := repository.NewDomainRepository(db.DB)
	webhookLogRepo := repository.NewWebhookEventLogRepository(db.DB)

	reg := worker.NewRegistry(qc, db.DB, domainRepo, webhookLogRepo)
	if err := reg.Register(queue.QueueWebhookDispatch, reg.HandleWebhook); err != nil {
		log.Fatalf("worker: %v", err)
	}
	if err := reg.Register(queue.QueueEmailSend, reg.HandleEmail); err != nil {
		log.Fatalf("worker: %v", err)
	}
	if err := reg.Register(queue.QueueVerifyDomain, reg.HandleVerifyDomain); err != nil {
		log.Fatalf("worker: %v", err)
	}
	if err := reg.Register(queue.QueueCampaignSend, reg.HandleCampaignSend); err != nil {
		log.Fatalf("worker: %v", err)
	}
	if err := reg.Register(queue.QueueCampaignTrack, reg.HandleCampaignTrack); err != nil {
		log.Fatalf("worker: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("worker: starting")
	if err := reg.Start(ctx); err != nil {
		log.Fatalf("worker: %v", err)
	}
	log.Println("worker: stopped")
}
