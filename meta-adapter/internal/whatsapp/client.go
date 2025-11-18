package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Client wraps whatsmeow client with additional functionality
type Client struct {
	cli            *whatsmeow.Client
	container      *sqlstore.Container
	device         *store.Device
	eventHandlers  []EventHandler
	connected      bool
	mu             sync.RWMutex
	storePath      string
	autoReconnect  bool
}

// EventHandler is a function that handles WhatsApp events
type EventHandler func(evt interface{})

// NewClient creates a new WhatsApp client
func NewClient(storePath string, autoReconnect bool) (*Client, error) {
	// Ensure store directory exists
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	// Initialize store container
	dbPath := filepath.Join(storePath, "whatsmeow.db")
	container, err := sqlstore.New("sqlite", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), waLog.Zerolog(log.Logger))
	if err != nil {
		return nil, fmt.Errorf("failed to create store container: %w", err)
	}

	// Get first device or create new one
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create whatsmeow client
	cli := whatsmeow.NewClient(deviceStore, waLog.Zerolog(log.Logger))
	cli.EnableAutoReconnect = autoReconnect
	cli.AutoTrustIdentity = true

	client := &Client{
		cli:           cli,
		container:     container,
		device:        deviceStore,
		storePath:     storePath,
		autoReconnect: autoReconnect,
		eventHandlers: make([]EventHandler, 0),
	}

	// Register event handlers
	cli.AddEventHandler(client.handleEvent)

	return client, nil
}

// AddEventHandler adds an event handler
func (c *Client) AddEventHandler(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandlers = append(c.eventHandlers, handler)
}

// handleEvent dispatches events to registered handlers
func (c *Client) handleEvent(evt interface{}) {
	// Update connection status
	switch evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()
		log.Info().Msg("WhatsApp connected")
	case *events.Disconnected:
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		log.Warn().Msg("WhatsApp disconnected")
	case *events.LoggedOut:
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		log.Warn().Msg("WhatsApp logged out")
	}

	// Dispatch to registered handlers
	c.mu.RLock()
	handlers := make([]EventHandler, len(c.eventHandlers))
	copy(handlers, c.eventHandlers)
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(evt)
	}
}

// Connect connects to WhatsApp
func (c *Client) Connect(ctx context.Context) error {
	if c.cli.Store.ID == nil {
		return fmt.Errorf("not logged in, please scan QR code first")
	}

	return c.cli.Connect()
}

// Disconnect disconnects from WhatsApp
func (c *Client) Disconnect() {
	c.cli.Disconnect()
}

// IsConnected returns true if connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.cli.IsConnected()
}

// IsLoggedIn returns true if logged in
func (c *Client) IsLoggedIn() bool {
	return c.cli.Store.ID != nil
}

// GetQRCode generates a QR code for pairing
func (c *Client) GetQRCode(ctx context.Context) (string, error) {
	if c.cli.Store.ID != nil {
		return "", fmt.Errorf("already logged in")
	}

	qrChan, err := c.cli.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel: %w", err)
	}

	if err := c.cli.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	// Wait for QR code
	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			return evt.Code, nil
		}
		return "", fmt.Errorf("unexpected QR event: %s", evt.Event)
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("timeout waiting for QR code")
	}
}

// Logout logs out from WhatsApp
func (c *Client) Logout() error {
	return c.cli.Logout()
}

// SendMessage sends a message
func (c *Client) SendMessage(ctx context.Context, to string, message *waE2E.Message) (types.MessageID, time.Time, error) {
	if !c.IsConnected() {
		return "", time.Time{}, fmt.Errorf("not connected")
	}

	jid, err := parseJID(to)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid recipient: %w", err)
	}

	resp, err := c.cli.SendMessage(ctx, jid, message)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send message: %w", err)
	}

	return resp.ID, resp.Timestamp, nil
}

// DownloadMedia downloads media from a message
func (c *Client) DownloadMedia(msg *waE2E.Message) ([]byte, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	// Determine which media type and download
	if msg.ImageMessage != nil {
		return c.cli.Download(msg.ImageMessage)
	} else if msg.VideoMessage != nil {
		return c.cli.Download(msg.VideoMessage)
	} else if msg.AudioMessage != nil {
		return c.cli.Download(msg.AudioMessage)
	} else if msg.DocumentMessage != nil {
		return c.cli.Download(msg.DocumentMessage)
	} else if msg.StickerMessage != nil {
		return c.cli.Download(msg.StickerMessage)
	}

	return nil, fmt.Errorf("no downloadable media in message")
}

// UploadMedia uploads media and returns upload response
func (c *Client) UploadMedia(ctx context.Context, data []byte, mediaType whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	if !c.IsConnected() {
		return whatsmeow.UploadResponse{}, fmt.Errorf("not connected")
	}

	return c.cli.Upload(ctx, data, mediaType)
}

// GetUserInfo gets user information
func (c *Client) GetUserInfo(jids []types.JID) (map[types.JID]types.UserInfo, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	resp, err := c.cli.GetUserInfo(jids)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return resp, nil
}

// GetOwnJID returns the client's own JID
func (c *Client) GetOwnJID() types.JID {
	if c.cli.Store.ID == nil {
		return types.EmptyJID
	}
	return *c.cli.Store.ID
}

// GetPushName returns the client's push name
func (c *Client) GetPushName() string {
	return c.cli.Store.PushName
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.Disconnect()

	if c.container != nil {
		return c.container.Close()
	}

	return nil
}

// BuildTextMessage builds a simple text message
func BuildTextMessage(text string) *waE2E.Message {
	return &waE2E.Message{
		Conversation: proto.String(text),
	}
}

// BuildImageMessage builds an image message
func BuildImageMessage(upload whatsmeow.UploadResponse, caption string, jpegThumbnail []byte) *waE2E.Message {
	return &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			Mimetype:      proto.String("image/jpeg"),
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(uint64(upload.FileLength)),
			JPEGThumbnail: jpegThumbnail,
		},
	}
}

// BuildVideoMessage builds a video message
func BuildVideoMessage(upload whatsmeow.UploadResponse, caption string, jpegThumbnail []byte) *waE2E.Message {
	return &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			Mimetype:      proto.String("video/mp4"),
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(uint64(upload.FileLength)),
			JPEGThumbnail: jpegThumbnail,
		},
	}
}

// BuildAudioMessage builds an audio message
func BuildAudioMessage(upload whatsmeow.UploadResponse) *waE2E.Message {
	return &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			Mimetype:      proto.String("audio/ogg; codecs=opus"),
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(uint64(upload.FileLength)),
		},
	}
}

// BuildDocumentMessage builds a document message
func BuildDocumentMessage(upload whatsmeow.UploadResponse, filename, mimetype string, caption string) *waE2E.Message {
	return &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			Mimetype:      proto.String(mimetype),
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(uint64(upload.FileLength)),
			FileName:      proto.String(filename),
		},
	}
}

// BuildLocationMessage builds a location message
func BuildLocationMessage(latitude, longitude float64, name, address string) *waE2E.Message {
	return &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  proto.Float64(latitude),
			DegreesLongitude: proto.Float64(longitude),
			Name:             proto.String(name),
			Address:          proto.String(address),
		},
	}
}

// BuildReactionMessage builds a reaction message
func BuildReactionMessage(targetMessageID string, emoji string) *waE2E.Message {
	return &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waE2E.MessageKey{
				ID: proto.String(targetMessageID),
			},
			Text: proto.String(emoji),
		},
	}
}

// Helper functions

func parseJID(to string) (types.JID, error) {
	// If already contains @, parse directly
	if jid, err := types.ParseJID(to); err == nil {
		return jid, nil
	}

	// Otherwise, assume it's a phone number
	// Remove any non-digit characters
	phone := ""
	for _, r := range to {
		if r >= '0' && r <= '9' {
			phone += string(r)
		}
	}

	if phone == "" {
		return types.EmptyJID, fmt.Errorf("invalid phone number")
	}

	// Construct JID with default server
	return types.JID{
		User:   phone,
		Server: types.DefaultUserServer,
	}, nil
}
