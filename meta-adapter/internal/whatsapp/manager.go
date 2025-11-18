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
