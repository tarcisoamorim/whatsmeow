package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/errors"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/models"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// MessageTranslator translates between Meta API format and whatsmeow protobuf
type MessageTranslator struct {
	client *whatsapp.Client
}

// NewMessageTranslator creates a new message translator
func NewMessageTranslator(client *whatsapp.Client) *MessageTranslator {
	return &MessageTranslator{
		client: client,
	}
}

// TranslateToWhatsApp converts Meta API request to whatsmeow message
func (t *MessageTranslator) TranslateToWhatsApp(ctx context.Context, req *models.SendMessageRequest) (*waE2E.Message, error) {
	var msg *waE2E.Message

	switch req.Type {
	case models.MessageTypeText:
		msg = t.translateText(req)
	case models.MessageTypeImage:
		return t.translateImage(ctx, req)
	case models.MessageTypeVideo:
		return t.translateVideo(ctx, req)
	case models.MessageTypeAudio:
		return t.translateAudio(ctx, req)
	case models.MessageTypeDocument:
		return t.translateDocument(ctx, req)
	case models.MessageTypeLocation:
		msg = t.translateLocation(req)
	case models.MessageTypeContacts:
		msg = t.translateContacts(req)
	case models.MessageTypeReaction:
		msg = t.translateReaction(req)
	default:
		return nil, errors.ValidationError("type", fmt.Sprintf("unsupported message type: %s", req.Type))
	}

	// Add context info for replies
	if req.Context != nil {
		addContextInfo(msg, req.Context)
	}

	return msg, nil
}

// Text message translation
func (t *MessageTranslator) translateText(req *models.SendMessageRequest) *waE2E.Message {
	if req.Text == nil {
		return nil
	}

	return &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(req.Text.Body),
		},
	}
}

// Image message translation
func (t *MessageTranslator) translateImage(ctx context.Context, req *models.SendMessageRequest) (*waE2E.Message, error) {
	if req.Image == nil {
		return nil, errors.ValidationError("image", "image content is required")
	}

	// Download image if URL provided
	var data []byte
	var err error

	if req.Image.Link != "" {
		data, err = downloadMedia(req.Image.Link)
		if err != nil {
			return nil, errors.WrapError(err, "failed to download image")
		}
	} else {
		return nil, errors.ValidationError("image", "either 'id' or 'link' must be provided")
	}

	// Upload to WhatsApp
	upload, err := t.client.UploadMedia(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return nil, errors.WrapError(err, "failed to upload image")
	}

	// Build message
	caption := ""
	if req.Image.Caption != "" {
		caption = req.Image.Caption
	}

	return whatsapp.BuildImageMessage(upload, caption, nil), nil
}

// Video message translation
func (t *MessageTranslator) translateVideo(ctx context.Context, req *models.SendMessageRequest) (*waE2E.Message, error) {
	if req.Video == nil {
		return nil, errors.ValidationError("video", "video content is required")
	}

	var data []byte
	var err error

	if req.Video.Link != "" {
		data, err = downloadMedia(req.Video.Link)
		if err != nil {
			return nil, errors.WrapError(err, "failed to download video")
		}
	} else {
		return nil, errors.ValidationError("video", "either 'id' or 'link' must be provided")
	}

	upload, err := t.client.UploadMedia(ctx, data, whatsmeow.MediaVideo)
	if err != nil {
		return nil, errors.WrapError(err, "failed to upload video")
	}

	caption := ""
	if req.Video.Caption != "" {
		caption = req.Video.Caption
	}

	return whatsapp.BuildVideoMessage(upload, caption, nil), nil
}

// Audio message translation
func (t *MessageTranslator) translateAudio(ctx context.Context, req *models.SendMessageRequest) (*waE2E.Message, error) {
	if req.Audio == nil {
		return nil, errors.ValidationError("audio", "audio content is required")
	}

	var data []byte
	var err error

	if req.Audio.Link != "" {
		data, err = downloadMedia(req.Audio.Link)
		if err != nil {
			return nil, errors.WrapError(err, "failed to download audio")
		}
	} else {
		return nil, errors.ValidationError("audio", "either 'id' or 'link' must be provided")
	}

	upload, err := t.client.UploadMedia(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, errors.WrapError(err, "failed to upload audio")
	}

	return whatsapp.BuildAudioMessage(upload), nil
}

// Document message translation
func (t *MessageTranslator) translateDocument(ctx context.Context, req *models.SendMessageRequest) (*waE2E.Message, error) {
	if req.Document == nil {
		return nil, errors.ValidationError("document", "document content is required")
	}

	var data []byte
	var err error

	if req.Document.Link != "" {
		data, err = downloadMedia(req.Document.Link)
		if err != nil {
			return nil, errors.WrapError(err, "failed to download document")
		}
	} else {
		return nil, errors.ValidationError("document", "either 'id' or 'link' must be provided")
	}

	upload, err := t.client.UploadMedia(ctx, data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, errors.WrapError(err, "failed to upload document")
	}

	filename := req.Document.Filename
	if filename == "" {
		filename = "document"
	}

	mimetype := "application/octet-stream"
	caption := ""
	if req.Document.Caption != "" {
		caption = req.Document.Caption
	}

	return whatsapp.BuildDocumentMessage(upload, filename, mimetype, caption), nil
}

// Location message translation
func (t *MessageTranslator) translateLocation(req *models.SendMessageRequest) *waE2E.Message {
	if req.Location == nil {
		return nil
	}

	return whatsapp.BuildLocationMessage(
		req.Location.Latitude,
		req.Location.Longitude,
		req.Location.Name,
		req.Location.Address,
	)
}

// Contacts message translation
func (t *MessageTranslator) translateContacts(req *models.SendMessageRequest) *waE2E.Message {
	if req.Contacts == nil || len(req.Contacts) == 0 {
		return nil
	}

	contacts := make([]*waE2E.ContactMessage, len(req.Contacts))

	for i, contact := range req.Contacts {
		vcard := buildVCard(contact)
		contacts[i] = &waE2E.ContactMessage{
			DisplayName: proto.String(contact.Name.FormattedName),
			Vcard:       proto.String(vcard),
		}
	}

	if len(contacts) == 1 {
		return &waE2E.Message{
			ContactMessage: contacts[0],
		}
	}

	return &waE2E.Message{
		ContactsArrayMessage: &waE2E.ContactsArrayMessage{
			DisplayName: proto.String(req.Contacts[0].Name.FormattedName),
			Contacts:    contacts,
		},
	}
}

// Reaction message translation
func (t *MessageTranslator) translateReaction(req *models.SendMessageRequest) *waE2E.Message {
	if req.Reaction == nil {
		return nil
	}

	return whatsapp.BuildReactionMessage(req.Reaction.MessageID, req.Reaction.Emoji)
}

// Helper functions

func addContextInfo(msg *waE2E.Message, ctx *models.MessageContext) {
	contextInfo := &waE2E.ContextInfo{
		StanzaID: proto.String(ctx.MessageID),
	}

	// Add context info to the appropriate message type
	if msg.ExtendedTextMessage != nil {
		msg.ExtendedTextMessage.ContextInfo = contextInfo
	} else if msg.ImageMessage != nil {
		msg.ImageMessage.ContextInfo = contextInfo
	} else if msg.VideoMessage != nil {
		msg.VideoMessage.ContextInfo = contextInfo
	} else if msg.DocumentMessage != nil {
		msg.DocumentMessage.ContextInfo = contextInfo
	}
}

func downloadMedia(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download media: status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func buildVCard(contact models.ContactContent) string {
	var vcard strings.Builder

	vcard.WriteString("BEGIN:VCARD\n")
	vcard.WriteString("VERSION:3.0\n")

	// Name
	vcard.WriteString(fmt.Sprintf("FN:%s\n", contact.Name.FormattedName))
	if contact.Name.FirstName != "" || contact.Name.LastName != "" {
		vcard.WriteString(fmt.Sprintf("N:%s;%s;%s;;\n",
			contact.Name.LastName,
			contact.Name.FirstName,
			contact.Name.MiddleName,
		))
	}

	// Phone numbers
	for _, phone := range contact.Phones {
		phoneType := "CELL"
		if phone.Type != "" {
			phoneType = strings.ToUpper(phone.Type)
		}
		vcard.WriteString(fmt.Sprintf("TEL;TYPE=%s:%s\n", phoneType, phone.Phone))
	}

	// Emails
	for _, email := range contact.Emails {
		emailType := "INTERNET"
		if email.Type != "" {
			emailType = strings.ToUpper(email.Type)
		}
		vcard.WriteString(fmt.Sprintf("EMAIL;TYPE=%s:%s\n", emailType, email.Email))
	}

	// Organization
	if contact.Org != nil {
		if contact.Org.Company != "" {
			vcard.WriteString(fmt.Sprintf("ORG:%s\n", contact.Org.Company))
		}
		if contact.Org.Title != "" {
			vcard.WriteString(fmt.Sprintf("TITLE:%s\n", contact.Org.Title))
		}
	}

	// Birthday
	if contact.Birthday != "" {
		vcard.WriteString(fmt.Sprintf("BDAY:%s\n", contact.Birthday))
	}

	vcard.WriteString("END:VCARD")

	return vcard.String()
}
