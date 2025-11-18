package models

// Meta WhatsApp Business API compatible models
// Based on https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages

// SendMessageRequest represents a message send request compatible with Meta API
type SendMessageRequest struct {
	MessagingProduct string      `json:"messaging_product"` // Always "whatsapp"
	RecipientType    string      `json:"recipient_type,omitempty"`
	To               string      `json:"to"`
	Type             MessageType `json:"type"`

	// Message content (only one should be set based on Type)
	Text        *TextContent        `json:"text,omitempty"`
	Image       *MediaContent       `json:"image,omitempty"`
	Video       *MediaContent       `json:"video,omitempty"`
	Audio       *MediaContent       `json:"audio,omitempty"`
	Document    *DocumentContent    `json:"document,omitempty"`
	Sticker     *MediaContent       `json:"sticker,omitempty"`
	Location    *LocationContent    `json:"location,omitempty"`
	Contacts    []ContactContent    `json:"contacts,omitempty"`
	Interactive *InteractiveContent `json:"interactive,omitempty"`
	Template    *TemplateContent    `json:"template,omitempty"`
	Reaction    *ReactionContent    `json:"reaction,omitempty"`

	Context *MessageContext `json:"context,omitempty"` // For replies
}

type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeVideo       MessageType = "video"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeDocument    MessageType = "document"
	MessageTypeSticker     MessageType = "sticker"
	MessageTypeLocation    MessageType = "location"
	MessageTypeContacts    MessageType = "contacts"
	MessageTypeInteractive MessageType = "interactive"
	MessageTypeTemplate    MessageType = "template"
	MessageTypeReaction    MessageType = "reaction"
)

// TextContent represents text message content
type TextContent struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

// MediaContent represents media (image, video, audio, sticker)
type MediaContent struct {
	ID       string `json:"id,omitempty"`        // Media ID (if uploaded via Meta API)
	Link     string `json:"link,omitempty"`      // HTTP(S) URL
	Caption  string `json:"caption,omitempty"`   // Only for image/video
	Filename string `json:"filename,omitempty"`  // Only for audio
}

// DocumentContent represents document message
type DocumentContent struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// LocationContent represents location message
type LocationContent struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ContactContent represents contact message
type ContactContent struct {
	Name      ContactName    `json:"name"`
	Phones    []ContactPhone `json:"phones,omitempty"`
	Emails    []ContactEmail `json:"emails,omitempty"`
	Addresses []ContactAddr  `json:"addresses,omitempty"`
	Org       *ContactOrg    `json:"org,omitempty"`
	Birthday  string         `json:"birthday,omitempty"`
}

type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
}

type ContactPhone struct {
	Phone string `json:"phone"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

type ContactEmail struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"`
}

type ContactAddr struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

// InteractiveContent represents interactive messages (buttons, lists)
type InteractiveContent struct {
	Type   string                  `json:"type"` // "button", "list"
	Header *InteractiveHeader      `json:"header,omitempty"`
	Body   InteractiveBody         `json:"body"`
	Footer *InteractiveFooter      `json:"footer,omitempty"`
	Action InteractiveAction       `json:"action"`
}

type InteractiveHeader struct {
	Type     string          `json:"type"` // "text", "image", "video", "document"
	Text     string          `json:"text,omitempty"`
	Image    *MediaContent   `json:"image,omitempty"`
	Video    *MediaContent   `json:"video,omitempty"`
	Document *MediaContent   `json:"document,omitempty"`
}

type InteractiveBody struct {
	Text string `json:"text"`
}

type InteractiveFooter struct {
	Text string `json:"text"`
}

type InteractiveAction struct {
	Buttons  []ActionButton  `json:"buttons,omitempty"`
	Button   string          `json:"button,omitempty"` // For list type
	Sections []ActionSection `json:"sections,omitempty"`
}

type ActionButton struct {
	Type  string      `json:"type"` // "reply"
	Reply ButtonReply `json:"reply"`
}

type ButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ActionSection struct {
	Title string          `json:"title,omitempty"`
	Rows  []SectionRow    `json:"rows"`
}

type SectionRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// TemplateContent represents template message
type TemplateContent struct {
	Name       string               `json:"name"`
	Language   TemplateLanguage     `json:"language"`
	Components []TemplateComponent  `json:"components,omitempty"`
}

type TemplateLanguage struct {
	Code string `json:"code"` // e.g., "pt_BR", "en_US"
}

type TemplateComponent struct {
	Type       string                `json:"type"` // "header", "body", "button"
	SubType    string                `json:"sub_type,omitempty"`
	Parameters []TemplateParameter   `json:"parameters,omitempty"`
	Index      int                   `json:"index,omitempty"`
}

type TemplateParameter struct {
	Type     string        `json:"type"` // "text", "currency", "date_time", "image", "video", "document"
	Text     string        `json:"text,omitempty"`
	Currency *Currency     `json:"currency,omitempty"`
	DateTime *DateTime     `json:"date_time,omitempty"`
	Image    *MediaContent `json:"image,omitempty"`
	Video    *MediaContent `json:"video,omitempty"`
	Document *MediaContent `json:"document,omitempty"`
}

type Currency struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"`
}

type DateTime struct {
	FallbackValue string `json:"fallback_value"`
}

// ReactionContent represents reaction message
type ReactionContent struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// MessageContext for replies
type MessageContext struct {
	MessageID string `json:"message_id"`
}

// SendMessageResponse represents the response from send message
type SendMessageResponse struct {
	MessagingProduct string          `json:"messaging_product"`
	Contacts         []ContactResult `json:"contacts"`
	Messages         []MessageResult `json:"messages"`
}

type ContactResult struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type MessageResult struct {
	ID               string `json:"id"`
	MessageStatus    string `json:"message_status,omitempty"`
}

// ErrorResponse represents Meta API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorData    any    `json:"error_data,omitempty"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
	FBTraceID    string `json:"fbtrace_id,omitempty"`
}

// Webhook models for incoming messages
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

type WebhookChange struct {
	Value WebhookValue `json:"value"`
	Field string       `json:"field"`
}

type WebhookValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         WebhookMetadata   `json:"metadata"`
	Contacts         []WebhookContact  `json:"contacts,omitempty"`
	Messages         []WebhookMessage  `json:"messages,omitempty"`
	Statuses         []WebhookStatus   `json:"statuses,omitempty"`
}

type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WebhookContact struct {
	Profile WebhookProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

type WebhookProfile struct {
	Name string `json:"name"`
}

type WebhookMessage struct {
	From      string               `json:"from"`
	ID        string               `json:"id"`
	Timestamp string               `json:"timestamp"`
	Type      MessageType          `json:"type"`

	Text        *TextContent         `json:"text,omitempty"`
	Image       *WebhookMedia        `json:"image,omitempty"`
	Video       *WebhookMedia        `json:"video,omitempty"`
	Audio       *WebhookMedia        `json:"audio,omitempty"`
	Document    *WebhookMedia        `json:"document,omitempty"`
	Sticker     *WebhookMedia        `json:"sticker,omitempty"`
	Location    *LocationContent     `json:"location,omitempty"`
	Contacts    []ContactContent     `json:"contacts,omitempty"`
	Reaction    *ReactionContent     `json:"reaction,omitempty"`
	Context     *MessageContext      `json:"context,omitempty"`
}

type WebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type WebhookStatus struct {
	ID           string               `json:"id"`
	RecipientID  string               `json:"recipient_id"`
	Status       string               `json:"status"` // "sent", "delivered", "read", "failed"
	Timestamp    string               `json:"timestamp"`
	Conversation *ConversationInfo    `json:"conversation,omitempty"`
	Pricing      *PricingInfo         `json:"pricing,omitempty"`
	Errors       []ErrorDetail        `json:"errors,omitempty"`
}

type ConversationInfo struct {
	ID                 string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Origin             Origin `json:"origin"`
}

type Origin struct {
	Type string `json:"type"` // "business_initiated", "user_initiated", "referral_conversion"
}

type PricingInfo struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}
