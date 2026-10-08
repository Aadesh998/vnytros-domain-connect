package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"domain-connect-backend/internal/jobs"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	URL  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

var publisher *Client

func SetPublisher(c *Client) { publisher = c }

func NewClient(url string) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("queue: empty RabbitMQ URL")
	}
	c := &Client{URL: url}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable broker url)"
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.Redacted()
}

func (c *Client) connect() error {
	conn, err := amqp.Dial(c.URL)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("amqp channel: %w", err)
	}

	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("amqp qos: %w", err)
	}

	c.conn = conn
	c.ch = ch
	log.Printf("queue: connected to %s", redactURL(c.URL))

	go c.watchClose()
	return nil
}

func (c *Client) watchClose() {
	err := <-c.conn.NotifyClose(make(chan *amqp.Error))
	if err != nil {
		log.Printf("queue: connection closed: %v — reconnecting in 5s", err)
		time.Sleep(5 * time.Second)
		if reconnectErr := c.connect(); reconnectErr != nil {
			log.Printf("queue: reconnect failed: %v", reconnectErr)
		}
	}
}

// DeclareQueue is idempotent — safe to call every startup.
func (c *Client) DeclareQueue(name string) error {
	if c.ch == nil {
		return fmt.Errorf("queue: channel not open")
	}
	_, err := c.ch.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", name, err)
	}
	log.Printf("queue: declared %q", name)
	return nil
}

// Publish marshals payload as JSON and pushes it onto queueName.
func (c *Client) Publish(ctx context.Context, queueName string, payload any) error {
	if c.ch == nil {
		return fmt.Errorf("queue: channel not open")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	err = c.ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
	})
	if err != nil {
		return fmt.Errorf("publish to %q: %w", queueName, err)
	}
	return nil
}

// Channel exposes the AMQP channel directly to consumers.
func (c *Client) Channel() *amqp.Channel { return c.ch }

// Close shuts down channel and connection.
func (c *Client) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func Enqueue(ctx context.Context, payload any) error {
	if publisher == nil {
		return fmt.Errorf("queue: publisher not configured")
	}

	var queueName string
	switch payload.(type) {
	case jobs.EmailJob:
		queueName = QueueEmailSend
	case jobs.WebhookJob:
		queueName = QueueWebhookDispatch
	case jobs.VerifyDomainJob:
		queueName = QueueVerifyDomain

	default:
		return fmt.Errorf("queue: unknown job type %T", payload)
	}

	return publisher.Publish(ctx, queueName, payload)
}

func EnqueueEmail(ctx context.Context, to, subject, template string, vars map[string]string) error {
	return Enqueue(ctx, jobs.EmailJob{
		To:       to,
		Subject:  subject,
		Template: template,
		Vars:     vars,
	})
}

func EnqueueWebhook(ctx context.Context, eventLogID uint, url, event, payload string) error {
	return Enqueue(ctx, jobs.WebhookJob{
		EventLogID: eventLogID,
		URL:        url,
		Event:      event,
		Payload:    payload,
	})
}

func EnqueueVerifyDomain(ctx context.Context, domainID uint, attempt int) error {
	return Enqueue(ctx, jobs.VerifyDomainJob{
		DomainID: domainID,
		Attempt:  attempt,
	})
}
