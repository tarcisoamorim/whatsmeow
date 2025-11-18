package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_Generate(t *testing.T) {
	manager := NewJWTManager("test-secret")

	tests := []struct {
		name      string
		tenantID  string
		clientID  string
		scopes    []string
		expiresIn time.Duration
		wantErr   bool
	}{
		{
			name:      "valid token",
			tenantID:  "tenant-123",
			clientID:  "client-456",
			scopes:    []string{"messages.send", "messages.read"},
			expiresIn: time.Hour,
			wantErr:   false,
		},
		{
			name:      "empty scopes",
			tenantID:  "tenant-123",
			clientID:  "client-456",
			scopes:    []string{},
			expiresIn: time.Hour,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.Generate(tt.tenantID, tt.clientID, tt.scopes, tt.expiresIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("Generate() returned empty token")
			}
		})
	}
}

func TestJWTManager_Verify(t *testing.T) {
	manager := NewJWTManager("test-secret")

	// Generate a valid token
	tenantID := "tenant-123"
	clientID := "client-456"
	scopes := []string{"messages.send"}
	token, err := manager.Generate(tenantID, clientID, scopes, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		checkClaims func(*Claims) bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
			checkClaims: func(c *Claims) bool {
				return c.TenantID == tenantID && c.ClientID == clientID && len(c.Scopes) == 1
			},
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.Verify(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Error("Verify() returned nil claims")
					return
				}
				if tt.checkClaims != nil && !tt.checkClaims(claims) {
					t.Error("Claims validation failed")
				}
			}
		})
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret")

	// Generate a token that expires immediately
	token, err := manager.Generate("tenant-123", "client-456", []string{"test"}, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to verify expired token
	_, err = manager.Verify(token)
	if err == nil {
		t.Error("Verify() should fail for expired token")
	}
	if err != ErrTokenExpired && err != ErrInvalidToken {
		t.Errorf("Expected ErrTokenExpired or ErrInvalidToken, got %v", err)
	}
}

func TestClaims_HasScope(t *testing.T) {
	claims := &Claims{
		TenantID: "tenant-123",
		ClientID: "client-456",
		Scopes:   []string{"messages.send", "messages.read", "instances.manage"},
	}

	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"existing scope", "messages.send", true},
		{"another existing scope", "instances.manage", true},
		{"non-existing scope", "messages.delete", false},
		{"empty scope", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasScope(tt.scope); got != tt.want {
				t.Errorf("HasScope(%s) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestJWTManager_DifferentSecrets(t *testing.T) {
	manager1 := NewJWTManager("secret1")
	manager2 := NewJWTManager("secret2")

	// Generate token with manager1
	token, err := manager1.Generate("tenant-123", "client-456", []string{"test"}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to verify with manager2 (different secret)
	_, err = manager2.Verify(token)
	if err == nil {
		t.Error("Verify() should fail when using different secret")
	}
}

func TestJWTManager_TokenStructure(t *testing.T) {
	manager := NewJWTManager("test-secret")

	token, err := manager.Generate("tenant-123", "client-456", []string{"test"}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Parse token without verification to check structure
	parser := jwt.NewParser()
	claims := &Claims{}
	_, _, err = parser.ParseUnverified(token, claims)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if claims.TenantID != "tenant-123" {
		t.Errorf("TenantID = %s, want tenant-123", claims.TenantID)
	}
	if claims.ClientID != "client-456" {
		t.Errorf("ClientID = %s, want client-456", claims.ClientID)
	}
	if claims.Issuer != "whatsapp-meta-api-adapter" {
		t.Errorf("Issuer = %s, want whatsapp-meta-api-adapter", claims.Issuer)
	}
}
