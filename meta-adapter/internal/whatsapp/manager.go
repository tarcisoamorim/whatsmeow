package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type Manager struct {
	container *sqlstore.Container
	clients   map[string]*ClientInstance
	mu        sync.RWMutex

	instanceRepo *repository.InstanceRepository
	messageRepo  *repository.MessageRepository
}

type ClientInstance struct {
	TenantID   string
	InstanceID string
	Client     *whatsmeow.Client
	Connected  bool
	mu         sync.RWMutex
}

func NewManager(dbURL string, instanceRepo *repository.InstanceRepository, messageRepo *repository.MessageRepository) (*Manager, error) {
	// Create whatsmeow store container
	container, err := sqlstore.New("postgres", dbURL, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to create whatsmeow store: %w", err)
	}

	return &Manager{
		container:    container,
		clients:      make(map[string]*ClientInstance),
		instanceRepo: instanceRepo,
		messageRepo:  messageRepo,
	}, nil
}

// GetOrCreateClient gets an existing client or creates a new one
func (m *Manager) GetOrCreateClient(ctx context.Context, tenantID, instanceID string) (*ClientInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, instanceID)

	// Check if client already exists
	if client, exists := m.clients[key]; exists {
		return client, nil
	}

	// Get or create device from store
	deviceStore, err := m.container.GetFirstDevice()
	if err != nil {
		deviceStore = m.container.NewDevice()
	}

	// Create whatsmeow client
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	// Set up event handler
	clientInstance := &ClientInstance{
		TenantID:   tenantID,
		InstanceID: instanceID,
		Client:     client,
		Connected:  false,
	}

	// Add event handlers
	client.AddEventHandler(m.eventHandler(clientInstance))

	m.clients[key] = clientInstance

	return clientInstance, nil
}

// GenerateQRCode generates a QR code for WhatsApp connection
func (m *Manager) GenerateQRCode(ctx context.Context, tenantID, instanceID string) (string, time.Time, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get client: %w", err)
	}

	// Check if already connected
	if clientInstance.Client.IsConnected() {
		return "", time.Time{}, fmt.Errorf("instance is already connected")
	}

	// Generate QR channel
	qrChan, err := clientInstance.Client.GetQRChannel(ctx)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Connect
	err = clientInstance.Client.Connect()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to connect: %w", err)
	}

	// Wait for QR code
	select {
	case evt := <-qrChan:
		switch evt.Event {
		case "code":
			// Generate QR code image
			png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				return "", time.Time{}, fmt.Errorf("failed to encode QR code: %w", err)
			}

			// Convert to base64
			qrBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
			expiresAt := time.Now().Add(90 * time.Second) // QR codes typically expire in 60-90 seconds

			// Update instance in database
			err = m.instanceRepo.UpdateQRCode(ctx, tenantID, instanceID, qrBase64, expiresAt)
			if err != nil {
				logger.Error("Failed to update QR code in database", zap.Error(err))
			}

			return qrBase64, expiresAt, nil

		case "success":
			// Connected successfully
			clientInstance.mu.Lock()
			clientInstance.Connected = true
			clientInstance.mu.Unlock()

			err = m.instanceRepo.UpdateConnectionStatus(ctx, tenantID, instanceID, true)
			if err != nil {
				logger.Error("Failed to update connection status", zap.Error(err))
			}

			return "", time.Time{}, fmt.Errorf("connected successfully, no QR code needed")
		}

	case <-time.After(2 * time.Minute):
		return "", time.Time{}, fmt.Errorf("timeout waiting for QR code")
	}

	return "", time.Time{}, fmt.Errorf("unexpected QR code generation failure")
}

// SendTextMessage sends a text message via WhatsApp
func (m *Manager) SendTextMessage(ctx context.Context, tenantID, instanceID, to, text string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	// Check if connected
	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	// Parse JID (WhatsApp ID)
	jid, err := types.ParseJID(to)
	if err != nil {
		// Try adding @s.whatsapp.net if not present
		jid, err = types.ParseJID(to + "@s.whatsapp.net")
		if err != nil {
			return "", fmt.Errorf("invalid recipient: %w", err)
		}
	}

	// Create message
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	// Send message
	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send WhatsApp message",
			zap.Error(err),
			zap.String("tenant_id", tenantID),
			zap.String("instance_id", instanceID),
			zap.String("to", to),
		)
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	logger.Info("WhatsApp message sent successfully",
		zap.String("message_id", resp.ID),
		zap.String("tenant_id", tenantID),
		zap.String("instance_id", instanceID),
		zap.String("to", to),
	)

	return resp.ID, nil
}

// SendImageMessage sends an image message via WhatsApp
func (m *Manager) SendImageMessage(ctx context.Context, tenantID, instanceID, to string, imageData []byte, caption string, mimeType string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	// Upload image
	uploaded, err := clientInstance.Client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Parse recipient JID
	jid, err := m.parseJID(to)
	if err != nil {
		return "", err
	}

	// Create image message
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Caption:       proto.String(caption),
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send image message", zap.Error(err))
		return "", fmt.Errorf("failed to send image: %w", err)
	}

	logger.Info("Image message sent", zap.String("message_id", resp.ID))
	return resp.ID, nil
}

// SendVideoMessage sends a video message via WhatsApp
func (m *Manager) SendVideoMessage(ctx context.Context, tenantID, instanceID, to string, videoData []byte, caption string, mimeType string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	// Upload video
	uploaded, err := clientInstance.Client.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return "", fmt.Errorf("failed to upload video: %w", err)
	}

	jid, err := m.parseJID(to)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Caption:       proto.String(caption),
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send video message", zap.Error(err))
		return "", fmt.Errorf("failed to send video: %w", err)
	}

	logger.Info("Video message sent", zap.String("message_id", resp.ID))
	return resp.ID, nil
}

// SendAudioMessage sends an audio/voice message via WhatsApp
func (m *Manager) SendAudioMessage(ctx context.Context, tenantID, instanceID, to string, audioData []byte, mimeType string, isVoice bool) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	// Upload audio
	uploaded, err := clientInstance.Client.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio: %w", err)
	}

	jid, err := m.parseJID(to)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Ptt:           proto.Bool(isVoice), // Push-to-talk (voice message)
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send audio message", zap.Error(err))
		return "", fmt.Errorf("failed to send audio: %w", err)
	}

	logger.Info("Audio message sent", zap.String("message_id", resp.ID))
	return resp.ID, nil
}

// SendDocumentMessage sends a document message via WhatsApp
func (m *Manager) SendDocumentMessage(ctx context.Context, tenantID, instanceID, to string, documentData []byte, fileName string, mimeType string, caption string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	// Upload document
	uploaded, err := clientInstance.Client.Upload(ctx, documentData, whatsmeow.MediaDocument)
	if err != nil {
		return "", fmt.Errorf("failed to upload document: %w", err)
	}

	jid, err := m.parseJID(to)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			FileName:      proto.String(fileName),
			Caption:       proto.String(caption),
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send document message", zap.Error(err))
		return "", fmt.Errorf("failed to send document: %w", err)
	}

	logger.Info("Document message sent", zap.String("message_id", resp.ID))
	return resp.ID, nil
}

// SendLocationMessage sends a location message via WhatsApp
func (m *Manager) SendLocationMessage(ctx context.Context, tenantID, instanceID, to string, latitude, longitude float64, name, address string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(to)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		LocationMessage: &waProto.LocationMessage{
			DegreesLatitude:  proto.Float64(latitude),
			DegreesLongitude: proto.Float64(longitude),
			Name:             proto.String(name),
			Address:          proto.String(address),
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send location message", zap.Error(err))
		return "", fmt.Errorf("failed to send location: %w", err)
	}

	logger.Info("Location message sent", zap.String("message_id", resp.ID))
	return resp.ID, nil
}

// DownloadMedia downloads media from a WhatsApp message
func (m *Manager) DownloadMedia(ctx context.Context, tenantID, instanceID string, msg *waProto.Message) ([]byte, error) {
	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	data, err := clientInstance.Client.Download(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	return data, nil
}

// SetPresence sets the presence status (online/offline)
func (m *Manager) SetPresence(ctx context.Context, tenantID, instanceID string, available bool) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	var presence types.Presence
	if available {
		presence = types.PresenceAvailable
	} else {
		presence = types.PresenceUnavailable
	}

	err = clientInstance.Client.SendPresence(presence)
	if err != nil {
		logger.Error("Failed to send presence", zap.Error(err), zap.Bool("available", available))
		return fmt.Errorf("failed to send presence: %w", err)
	}

	logger.Info("Presence updated", zap.Bool("available", available))
	return nil
}

// SendChatPresence sends typing/recording indicators
func (m *Manager) SendChatPresence(ctx context.Context, tenantID, instanceID, to string, state string, media string) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(to)
	if err != nil {
		return err
	}

	var chatPresence types.ChatPresence
	switch state {
	case "composing":
		chatPresence = types.ChatPresenceComposing
	case "paused":
		chatPresence = types.ChatPresencePaused
	default:
		return fmt.Errorf("invalid state: %s", state)
	}

	var mediaType types.ChatPresenceMedia
	switch media {
	case "audio":
		mediaType = types.ChatPresenceMediaAudio
	case "text":
		mediaType = types.ChatPresenceMediaText
	default:
		mediaType = types.ChatPresenceMediaText
	}

	err = clientInstance.Client.SendChatPresence(jid, chatPresence, mediaType)
	if err != nil {
		logger.Error("Failed to send chat presence", zap.Error(err))
		return fmt.Errorf("failed to send chat presence: %w", err)
	}

	logger.Debug("Chat presence sent", zap.String("state", state), zap.String("to", to))
	return nil
}

// MarkMessageRead marks a message as read
func (m *Manager) MarkMessageRead(ctx context.Context, tenantID, instanceID, chatJID string, messageIDs []string, timestamp time.Time) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(chatJID)
	if err != nil {
		return err
	}

	err = clientInstance.Client.MarkRead(messageIDs, timestamp, jid, jid)
	if err != nil {
		logger.Error("Failed to mark messages as read", zap.Error(err))
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	logger.Info("Messages marked as read", zap.Int("count", len(messageIDs)))
	return nil
}

// DeleteMessage deletes a message for everyone
func (m *Manager) DeleteMessage(ctx context.Context, tenantID, instanceID, chatJID, messageID string) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(chatJID)
	if err != nil {
		return err
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, clientInstance.Client.BuildRevoke(jid, types.EmptyJID, messageID))
	if err != nil {
		logger.Error("Failed to delete message", zap.Error(err))
		return fmt.Errorf("failed to delete message: %w", err)
	}

	logger.Info("Message deleted", zap.String("deleted_message_id", resp.ID))
	return nil
}

// ReactToMessage sends a reaction to a message
func (m *Manager) ReactToMessage(ctx context.Context, tenantID, instanceID, chatJID, messageID, emoji string) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(chatJID)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				RemoteJid: proto.String(chatJID),
				FromMe:    proto.Bool(false),
				Id:        proto.String(messageID),
			},
			Text:              proto.String(emoji),
			SenderTimestampMs: proto.Int64(time.Now().UnixMilli()),
		},
	}

	resp, err := clientInstance.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		logger.Error("Failed to send reaction", zap.Error(err))
		return "", fmt.Errorf("failed to send reaction: %w", err)
	}

	logger.Info("Reaction sent", zap.String("emoji", emoji), zap.String("to_message", messageID))
	return resp.ID, nil
}

// parseJID parses a phone number or JID string
func (m *Manager) parseJID(recipient string) (types.JID, error) {
	jid, err := types.ParseJID(recipient)
	if err != nil {
		// Try adding @s.whatsapp.net if not present
		jid, err = types.ParseJID(recipient + "@s.whatsapp.net")
		if err != nil {
			return types.EmptyJID, fmt.Errorf("invalid recipient: %w", err)
		}
	}
	return jid, nil
}

// Disconnect disconnects a WhatsApp instance
func (m *Manager) Disconnect(ctx context.Context, tenantID, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, instanceID)

	clientInstance, exists := m.clients[key]
	if !exists {
		return fmt.Errorf("client not found")
	}

	clientInstance.Client.Disconnect()

	clientInstance.mu.Lock()
	clientInstance.Connected = false
	clientInstance.mu.Unlock()

	// Update database
	err := m.instanceRepo.UpdateConnectionStatus(ctx, tenantID, instanceID, false)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	delete(m.clients, key)

	return nil
}

// eventHandler creates an event handler for WhatsApp events
func (m *Manager) eventHandler(clientInstance *ClientInstance) func(interface{}) {
	logger := pkglogger.Get()

	return func(evt interface{}) {
		ctx := context.Background()

		switch v := evt.(type) {
		case *events.Message:
			// Handle incoming message
			logger.Info("Received WhatsApp message",
				zap.String("from", v.Info.Sender.String()),
				zap.String("message_id", v.Info.ID),
			)

			// Save to database
			// TODO: Implement full message parsing and saving

		case *events.Receipt:
			// Handle message delivery/read receipts
			logger.Debug("Received receipt",
				zap.String("message_id", v.MessageIDs[0]),
				zap.String("type", string(v.Type)),
			)

			// Update message status in database
			switch v.Type {
			case types.ReceiptTypeDelivered:
				for _, msgID := range v.MessageIDs {
					err := m.messageRepo.MarkAsDelivered(ctx, clientInstance.TenantID, clientInstance.InstanceID, msgID)
					if err != nil {
						logger.Error("Failed to mark message as delivered", zap.Error(err), zap.String("message_id", msgID))
					}
				}
			case types.ReceiptTypeRead:
				for _, msgID := range v.MessageIDs {
					err := m.messageRepo.MarkAsRead(ctx, clientInstance.TenantID, clientInstance.InstanceID, msgID)
					if err != nil {
						logger.Error("Failed to mark message as read", zap.Error(err), zap.String("message_id", msgID))
					}
				}
			}

		case *events.Connected:
			logger.Info("WhatsApp connected", zap.String("instance_id", clientInstance.InstanceID))

			clientInstance.mu.Lock()
			clientInstance.Connected = true
			clientInstance.mu.Unlock()

			err := m.instanceRepo.UpdateConnectionStatus(ctx, clientInstance.TenantID, clientInstance.InstanceID, true)
			if err != nil {
				logger.Error("Failed to update connection status", zap.Error(err))
			}

		case *events.Disconnected:
			logger.Info("WhatsApp disconnected", zap.String("instance_id", clientInstance.InstanceID))

			clientInstance.mu.Lock()
			clientInstance.Connected = false
			clientInstance.mu.Unlock()

			err := m.instanceRepo.UpdateConnectionStatus(ctx, clientInstance.TenantID, clientInstance.InstanceID, false)
			if err != nil {
				logger.Error("Failed to update connection status", zap.Error(err))
			}
		}
	}
}

// Shutdown gracefully shuts down all WhatsApp connections
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.Client.Disconnect()
	}

	m.clients = make(map[string]*ClientInstance)
}
