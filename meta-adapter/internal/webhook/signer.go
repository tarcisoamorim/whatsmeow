package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Signer handles HMAC signing of webhook payloads
type Signer struct {
	secret []byte
}

// NewSigner creates a new webhook signer
func NewSigner(secret string) *Signer {
	return &Signer{
		secret: []byte(secret),
	}
}

// Sign creates an HMAC signature for a webhook payload
func (s *Signer) Sign(payload interface{}) (string, error) {
	// Serialize payload to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Create HMAC
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(data)
	signature := hex.EncodeToString(mac.Sum(nil))

	return "sha256=" + signature, nil
}

// Verify verifies an HMAC signature
func (s *Signer) Verify(payload interface{}, signature string) bool {
	expectedSignature, err := s.Sign(payload)
	if err != nil {
		return false
	}

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
