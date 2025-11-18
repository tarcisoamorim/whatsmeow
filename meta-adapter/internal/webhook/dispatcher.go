package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/models"
	"go.mau.fi/whatsmeow/types/events"
)

// Dispatcher sends webhooks to configured URL
type Dispatcher struct {
	webhookURL    string
	timeout       time.Duration
	retryAttempts int
	phoneNumberID string
	client        *http.Client
}

// NewDispatcher creates a new webhook dispatcher
func NewDispatcher(webhookURL string, timeout time.Duration, retryAttempts int, phoneNumberID string) *Dispatcher {
	return &Dispatcher{
		webhookURL:    webhookURL,
		timeout:       timeout,
		retryAttempts: retryAttempts,
		phoneNumberID: phoneNumberID,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// DispatchMessage dispatches a message event as webhook
func (d *Dispatcher) DispatchMessage(evt *events.Message) error {
	if d.webhookURL == "" {
		return nil // No webhook configured
	}

	payload := d.buildMessageWebhook(evt)
	return d.send(payload)
}

// DispatchReceipt dispatches a receipt (delivery/read status) event
func (d *Dispatcher) DispatchReceipt(evt *events.Receipt) error {
	if d.webhookURL == "" {
		return nil
	}

	payload := d.buildReceiptWebhook(evt)
	return d.send(payload)
}

// buildMessageWebhook converts whatsmeow message event to Meta webhook format
func (d *Dispatcher) buildMessageWebhook(evt *events.Message) *models.WebhookPayload {
	webhookMsg := models.WebhookMessage{
		From:      evt.Info.Sender.User,
		ID:        evt.Info.ID,
		Timestamp: fmt.Sprintf("%d", evt.Info.Timestamp.Unix()),
	}

	// Determine message type and content
	msg := evt.Message

	if msg.Conversation != nil && *msg.Conversation != "" {
		webhookMsg.Type = models.MessageTypeText
		webhookMsg.Text = &models.TextContent{
			Body: *msg.Conversation,
		}
	} else if msg.ExtendedTextMessage != nil {
		webhookMsg.Type = models.MessageTypeText
		webhookMsg.Text = &models.TextContent{
			Body: msg.ExtendedTextMessage.GetText(),
		}
	} else if msg.ImageMessage != nil {
		webhookMsg.Type = models.MessageTypeImage
		webhookMsg.Image = &models.WebhookMedia{
			ID:       evt.Info.ID,
			MimeType: msg.ImageMessage.GetMimetype(),
			SHA256:   fmt.Sprintf("%x", msg.ImageMessage.GetFileSHA256()),
			Caption:  msg.ImageMessage.GetCaption(),
		}
	} else if msg.VideoMessage != nil {
		webhookMsg.Type = models.MessageTypeVideo
		webhookMsg.Video = &models.WebhookMedia{
			ID:       evt.Info.ID,
			MimeType: msg.VideoMessage.GetMimetype(),
			SHA256:   fmt.Sprintf("%x", msg.VideoMessage.GetFileSHA256()),
			Caption:  msg.VideoMessage.GetCaption(),
		}
	} else if msg.AudioMessage != nil {
		webhookMsg.Type = models.MessageTypeAudio
		webhookMsg.Audio = &models.WebhookMedia{
			ID:       evt.Info.ID,
			MimeType: msg.AudioMessage.GetMimetype(),
			SHA256:   fmt.Sprintf("%x", msg.AudioMessage.GetFileSHA256()),
		}
	} else if msg.DocumentMessage != nil {
		webhookMsg.Type = models.MessageTypeDocument
		webhookMsg.Document = &models.WebhookMedia{
			ID:       evt.Info.ID,
			MimeType: msg.DocumentMessage.GetMimetype(),
			SHA256:   fmt.Sprintf("%x", msg.DocumentMessage.GetFileSHA256()),
			Caption:  msg.DocumentMessage.GetCaption(),
			Filename: msg.DocumentMessage.GetFileName(),
		}
	} else if msg.LocationMessage != nil {
		webhookMsg.Type = models.MessageTypeLocation
		webhookMsg.Location = &models.LocationContent{
			Latitude:  msg.LocationMessage.GetDegreesLatitude(),
			Longitude: msg.LocationMessage.GetDegreesLongitude(),
			Name:      msg.LocationMessage.GetName(),
			Address:   msg.LocationMessage.GetAddress(),
		}
	} else if msg.ReactionMessage != nil {
		webhookMsg.Type = models.MessageTypeReaction
		webhookMsg.Reaction = &models.ReactionContent{
			MessageID: msg.ReactionMessage.GetKey().GetID(),
			Emoji:     msg.ReactionMessage.GetText(),
		}
	} else {
		// Unsupported message type
		log.Warn().Str("message_id", evt.Info.ID).Msg("Unsupported message type for webhook")
		return nil
	}

	// Add context if it's a reply
	if msg.GetExtendedTextMessage().GetContextInfo() != nil {
		webhookMsg.Context = &models.MessageContext{
			MessageID: msg.GetExtendedTextMessage().GetContextInfo().GetStanzaID(),
		}
	}

	return &models.WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []models.WebhookEntry{
			{
				ID: d.phoneNumberID,
				Changes: []models.WebhookChange{
					{
						Value: models.WebhookValue{
							MessagingProduct: "whatsapp",
							Metadata: models.WebhookMetadata{
								DisplayPhoneNumber: evt.Info.Chat.User,
								PhoneNumberID:      d.phoneNumberID,
							},
							Contacts: []models.WebhookContact{
								{
									Profile: models.WebhookProfile{
										Name: evt.Info.PushName,
									},
									WaID: evt.Info.Sender.User,
								},
							},
							Messages: []models.WebhookMessage{webhookMsg},
						},
						Field: "messages",
					},
				},
			},
		},
	}
}

// buildReceiptWebhook converts receipt event to Meta webhook format
func (d *Dispatcher) buildReceiptWebhook(evt *events.Receipt) *models.WebhookPayload {
	status := "sent"
	switch evt.Type {
	case events.ReceiptTypeDelivered:
		status = "delivered"
	case events.ReceiptTypeRead:
		status = "read"
	case events.ReceiptTypePlayed:
		status = "played"
	}

	statuses := make([]models.WebhookStatus, len(evt.MessageIDs))
	for i, msgID := range evt.MessageIDs {
		statuses[i] = models.WebhookStatus{
			ID:          msgID,
			RecipientID: evt.Sender.User,
			Status:      status,
			Timestamp:   fmt.Sprintf("%d", evt.Timestamp.Unix()),
		}
	}

	return &models.WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []models.WebhookEntry{
			{
				ID: d.phoneNumberID,
				Changes: []models.WebhookChange{
					{
						Value: models.WebhookValue{
							MessagingProduct: "whatsapp",
							Metadata: models.WebhookMetadata{
								PhoneNumberID: d.phoneNumberID,
							},
							Statuses: statuses,
						},
						Field: "messages",
					},
				},
			},
		},
	}
}

// send sends the webhook payload with retry logic
func (d *Dispatcher) send(payload *models.WebhookPayload) error {
	if payload == nil {
		return nil
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= d.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
			log.Debug().Int("attempt", attempt).Msg("Retrying webhook dispatch")
		}

		ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
		req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(jsonData))
		cancel()

		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "WhatsApp-Meta-Adapter/1.0")

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Msg("Failed to send webhook")
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Debug().
				Str("url", d.webhookURL).
				Int("status", resp.StatusCode).
				Msg("Webhook dispatched successfully")
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		log.Warn().
			Int("status", resp.StatusCode).
			Int("attempt", attempt).
			Msg("Webhook returned non-2xx status")
	}

	log.Error().Err(lastErr).Msg("Failed to dispatch webhook after all retries")
	return lastErr
}
