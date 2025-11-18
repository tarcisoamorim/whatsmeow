package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

// Deliverer handles consuming webhook events from queue and delivering them
type Deliverer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
	signer     *Signer
	repository *repository.Database
}

// NewDeliverer creates a new webhook deliverer
func NewDeliverer(rabbitmqURL, webhookSecret string, db *repository.Database) (*Deliverer, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS to process one message at a time
	err = channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &Deliverer{
		conn:       conn,
		channel:    channel,
		httpClient: httpClient,
		signer:     NewSigner(webhookSecret),
		repository: db,
	}, nil
}

// Start starts consuming and delivering webhooks
func (d *Deliverer) Start(ctx context.Context) error {
	logger := pkglogger.Get()

	msgs, err := d.channel.Consume(
		WebhookQueue, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	logger.Info("Webhook deliverer started, waiting for events...")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Webhook deliverer shutting down...")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			// Process message
			if err := d.processMessage(ctx, msg); err != nil {
				logger.Error("Failed to process webhook message",
					zap.Error(err),
					zap.String("message_id", msg.MessageId),
				)
			}
		}
	}
}

// processMessage processes a single webhook message
func (d *Deliverer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	logger := pkglogger.Get()

	// Deserialize event
	var event Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		logger.Error("Failed to unmarshal event", zap.Error(err))
		msg.Nack(false, false) // Don't requeue malformed messages
		return err
	}

	logger.Info("Processing webhook event",
		zap.String("event_id", event.ID),
		zap.String("type", string(event.Type)),
		zap.Int("retry_count", event.RetryCount),
	)

	// Deliver webhook
	result := d.deliver(ctx, &event)

	// Log delivery attempt
	d.logDelivery(ctx, &event, &result)

	if result.Success {
		// ACK message on success
		msg.Ack(false)
		logger.Info("Webhook delivered successfully",
			zap.String("event_id", event.ID),
			zap.Int("status_code", result.StatusCode),
			zap.Duration("duration", result.Duration),
		)
		return nil
	}

	// Handle failure
	event.RetryCount++

	if event.RetryCount >= event.MaxRetries {
		// Max retries exceeded, send to DLQ
		logger.Error("Webhook max retries exceeded, sending to DLQ",
			zap.String("event_id", event.ID),
			zap.Int("retry_count", event.RetryCount),
		)
		msg.Nack(false, false) // Don't requeue, goes to DLQ
		return fmt.Errorf("max retries exceeded")
	}

	// Calculate backoff delay (exponential: 1s, 2s, 4s, 8s, 16s)
	delay := time.Duration(1<<uint(event.RetryCount-1)) * time.Second
	if delay > 16*time.Second {
		delay = 16 * time.Second
	}

	logger.Warn("Webhook delivery failed, will retry",
		zap.String("event_id", event.ID),
		zap.Int("retry_count", event.RetryCount),
		zap.Duration("delay", delay),
		zap.Error(result.Error),
	)

	// Requeue with delay
	time.Sleep(delay)

	// Republish event with updated retry count
	body, _ := json.Marshal(event)
	err := d.channel.PublishWithContext(
		ctx,
		WebhookExchange,
		WebhookQueue,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			MessageId:    event.ID,
		},
	)

	if err != nil {
		logger.Error("Failed to republish event", zap.Error(err))
		msg.Nack(false, true) // Requeue via RabbitMQ
		return err
	}

	msg.Ack(false) // ACK original message
	return nil
}

// deliver delivers a webhook to the target URL
func (d *Deliverer) deliver(ctx context.Context, event *Event) DeliveryResult {
	start := time.Now()

	// Build payload
	payload := WebhookPayload{
		Event:     event.Type,
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Data:      event.Data,
	}

	// Sign payload
	signature, err := d.signer.Sign(payload)
	if err != nil {
		return DeliveryResult{
			Success: false,
			Error:   fmt.Errorf("failed to sign payload: %w", err),
		}
	}
	payload.Signature = signature

	// Serialize payload
	body, err := json.Marshal(payload)
	if err != nil {
		return DeliveryResult{
			Success: false,
			Error:   fmt.Errorf("failed to marshal payload: %w", err),
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", event.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{
			Success: false,
			Error:   fmt.Errorf("failed to create request: %w", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsApp-Meta-API-Adapter/1.0")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", string(event.Type))
	req.Header.Set("X-Webhook-ID", event.ID)

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return DeliveryResult{
			Success:  false,
			Error:    err,
			Duration: time.Since(start),
		}
	}
	defer resp.Body.Close()

	// Read response body (limited to 1KB)
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Check if successful (2xx status code)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return DeliveryResult{
		Success:      success,
		StatusCode:   resp.StatusCode,
		ResponseBody: string(responseBody),
		Duration:     time.Since(start),
		Error:        nil,
	}
}

// logDelivery logs a webhook delivery attempt to the database
func (d *Deliverer) logDelivery(ctx context.Context, event *Event, result *DeliveryResult) {
	logger := pkglogger.Get()

	status := "failed"
	if result.Success {
		status = "delivered"
	}

	errorMsg := ""
	if result.Error != nil {
		errorMsg = result.Error.Error()
	}

	query := `
		INSERT INTO webhook_logs (
			tenant_id, instance_id, event_id, event_type, webhook_url,
			status, status_code, response_body, error, retry_count, duration_ms,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`

	_, err := d.repository.ExecContext(
		ctx, query,
		event.TenantID,
		event.InstanceID,
		event.ID,
		event.Type,
		event.WebhookURL,
		status,
		result.StatusCode,
		result.ResponseBody,
		errorMsg,
		event.RetryCount,
		result.Duration.Milliseconds(),
	)

	if err != nil {
		logger.Error("Failed to log webhook delivery", zap.Error(err))
	}
}

// Close closes the deliverer connection
func (d *Deliverer) Close() error {
	if err := d.channel.Close(); err != nil {
		return err
	}
	return d.conn.Close()
}
