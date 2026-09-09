package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// makeTestJWT creates a JWT with the given claims (header.payload.signature).
// For testing only — the signature is a dummy value (not cryptographically valid).
func makeTestJWT(claims auth.Claims) string {
	header := map[string]string{"alg": "RS256", "kid": "test", "typ": "JWT"}
	headerBytes, _ := json.Marshal(header)
	payloadBytes, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(headerBytes) + "." +
		base64.RawURLEncoding.EncodeToString(payloadBytes) + "." +
		"dummy-signature"
}

func TestDevVerifier_ValidToken(t *testing.T) {
	claims := auth.Claims{
		Subject:   "user-123",
		Issuer:    "test-issuer",
		Email:     "user@example.com",
		ExpiresAt: time.Now().Add(time.Hour),
		Scope:     "bill:read ai:ask",
	}
	token := makeTestJWT(claims)

	v := DevVerifier{}
	p, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if p.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", p.UserID)
	}
	if p.Email != "user@example.com" {
		t.Errorf("expected Email 'user@example.com', got '%s'", p.Email)
	}
	if !p.HasScope(auth.ScopeBillRead) {
		t.Errorf("expected scope bill:read")
	}
	if !p.HasScope(auth.ScopeAIAsk) {
		t.Errorf("expected scope ai:ask")
	}
	if p.HasScope(auth.ScopeUserAdmin) {
		t.Errorf("should not have scope user:admin")
	}
}

func TestDevVerifier_ExpiredToken(t *testing.T) {
	claims := auth.Claims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(-time.Hour), // expired 1 hour ago
	}
	token := makeTestJWT(claims)

	v := DevVerifier{}
	_, err := v.Verify(context.Background(), token)
	if err != auth.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestDevVerifier_InvalidToken(t *testing.T) {
	v := DevVerifier{}
	_, err := v.Verify(context.Background(), "not.a.valid.jwt.with.extra.parts")
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestDevVerifier_AnonymousWhenNoToken(t *testing.T) {
	// A truly empty token should also fail.
	v := DevVerifier{}
	_, err := v.Verify(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		input    string
		expected auth.Scopes
	}{
		{"", auth.Scopes{}},
		{"bill:read", auth.Scopes{"bill:read"}},
		{"bill:read ai:ask", auth.Scopes{"bill:read", "ai:ask"}},
		{"  bill:read   ai:ask  ", auth.Scopes{"bill:read", "ai:ask"}},
	}
	for _, tt := range tests {
		got := auth.ParseScopes(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("ParseScopes(%q): expected %d scopes, got %d", tt.input, len(tt.expected), len(got))
			continue
		}
		for i, s := range got {
			if s != tt.expected[i] {
				t.Errorf("ParseScopes(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], s)
			}
		}
	}
}

func TestPrincipal_HasScope(t *testing.T) {
	p := auth.Principal{
		UserID: "user-1",
		Scopes: auth.Scopes{"bill:read", "search:read"},
	}
	if !p.HasScope("bill:read") {
		t.Error("expected HasScope(bill:read) = true")
	}
	if p.HasScope("user:admin") {
		t.Error("expected HasScope(user:admin) = false")
	}
}

func TestPrincipal_IsAuthenticated(t *testing.T) {
	authd := auth.Principal{UserID: "user-1"}
	anon := auth.Anonymous()
	if !authd.IsAuthenticated() {
		t.Error("expected authenticated principal")
	}
	if !anon.IsAnonymous() {
		t.Error("expected anonymous principal")
	}
}

func TestRequire_SufficientScope(t *testing.T) {
	p := auth.Principal{
		UserID: "user-1",
		Scopes: auth.Scopes{"bill:read"},
	}
	err := auth.Require(p, auth.ScopeBillRead)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRequire_InsufficientScope(t *testing.T) {
	p := auth.Principal{
		UserID: "user-1",
		Scopes: auth.Scopes{"bill:read"},
	}
	err := auth.Require(p, auth.ScopeUserAdmin)
	if err == nil {
		t.Error("expected ForbiddenError")
	}
}
