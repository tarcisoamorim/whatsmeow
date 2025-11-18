package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
	"go.uber.org/zap"

	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

const (
	WebhookQueue      = "webhooks"
	WebhookDLQ        = "webhooks.dlq"
	WebhookExchange   = "webhooks.exchange"
	MaxRetries        = 5
)

// Publisher handles publishing webhook events to RabbitMQ
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher creates a new webhook publisher
func NewPublisher(rabbitmqURL string) (*Publisher, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		WebhookExchange, // name
		"direct",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare main queue
	_, err = channel.QueueDeclare(
		WebhookQueue, // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    WebhookExchange,
			"x-dead-letter-routing-key": WebhookDLQ,
		},
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		WebhookQueue,    // queue name
		WebhookQueue,    // routing key
		WebhookExchange, // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Declare DLQ
	_, err = channel.QueueDeclare(
		WebhookDLQ, // name
		true,       // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to exchange
	err = channel.QueueBind(
		WebhookDLQ,      // queue name
		WebhookDLQ,      // routing key
		WebhookExchange, // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind DLQ: %w", err)
	}

	return &Publisher{
		conn:    conn,
		channel: channel,
	}, nil
}

// Publish publishes a webhook event to the queue
func (p *Publisher) Publish(ctx context.Context, event *Event) error {
	logger := pkglogger.Get()

	// Set defaults
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.MaxRetries == 0 {
		event.MaxRetries = MaxRetries
	}

	// Serialize event
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to queue
	err = p.channel.PublishWithContext(
		ctx,
		WebhookExchange, // exchange
		WebhookQueue,    // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			MessageId:    event.ID,
			Timestamp:    event.Timestamp,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info("Webhook event published",
		zap.String("event_id", event.ID),
		zap.String("type", string(event.Type)),
		zap.String("tenant_id", event.TenantID),
	)

	return nil
}

// Close closes the publisher connection
func (p *Publisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
