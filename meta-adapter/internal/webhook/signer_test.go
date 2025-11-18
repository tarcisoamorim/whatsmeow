package webhook

import (
	"testing"
)

func TestSigner_Sign(t *testing.T) {
	signer := NewSigner("test-secret")

	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: WebhookPayload{
				Event:     EventMessageReceived,
				Timestamp: "2025-11-18T21:00:00Z",
				Data: map[string]interface{}{
					"message_id": "test-123",
					"from":       "5511999999999",
				},
			},
			wantErr: false,
		},
		{
			name: "simple map",
			payload: map[string]string{
				"test": "value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature, err := signer.Sign(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if signature == "" {
					t.Error("Sign() returned empty signature")
				}
				if len(signature) < 10 {
					t.Error("Sign() returned suspiciously short signature")
				}
				// Check signature format (should start with "sha256=")
				if signature[:7] != "sha256=" {
					t.Errorf("Sign() signature format incorrect, got %s", signature[:7])
				}
			}
		})
	}
}

func TestSigner_Verify(t *testing.T) {
	signer := NewSigner("test-secret")

	payload := WebhookPayload{
		Event:     EventMessageReceived,
		Timestamp: "2025-11-18T21:00:00Z",
		Data: map[string]interface{}{
			"message_id": "test-123",
		},
	}

	// Generate signature
	signature, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Failed to generate signature: %v", err)
	}

	tests := []struct {
		name      string
		payload   interface{}
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			signature: signature,
			want:      true,
		},
		{
			name:      "invalid signature",
			payload:   payload,
			signature: "sha256=invalid",
			want:      false,
		},
		{
			name: "modified payload",
			payload: WebhookPayload{
				Event:     EventMessageReceived,
				Timestamp: "2025-11-18T21:00:00Z",
				Data: map[string]interface{}{
					"message_id": "modified-456",
				},
			},
			signature: signature,
			want:      false,
		},
		{
			name:      "empty signature",
			payload:   payload,
			signature: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signer.Verify(tt.payload, tt.signature); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSigner_DifferentSecrets(t *testing.T) {
	signer1 := NewSigner("secret1")
	signer2 := NewSigner("secret2")

	payload := map[string]string{"test": "value"}

	// Sign with signer1
	signature, err := signer1.Sign(payload)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Verify with signer2 (different secret)
	if signer2.Verify(payload, signature) {
		t.Error("Verify() should fail with different secret")
	}
}

func TestSigner_SamePayloadSameSignature(t *testing.T) {
	signer := NewSigner("test-secret")

	payload := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}

	signature1, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	signature2, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if signature1 != signature2 {
		t.Error("Same payload should produce same signature")
	}
}
