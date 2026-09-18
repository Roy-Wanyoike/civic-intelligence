package auth

import (
	"context"
	"errors"
	"time"
)

// OIDCTokenVerifier is the interface implemented by anything that can verify
// an OIDC access token and turn it into a Principal. The concrete
// implementation (KeycloakVerifier) lives in identity service
// infrastructure; other services receive a verifier injected at startup.
type OIDCTokenVerifier interface {
	// Verify parses and verifies the bearer token, returning the
	// Principal it represents. Implementations MUST verify the signature,
	// expiry, audience and issuer.
	Verify(ctx context.Context, bearerToken string) (Principal, error)
}

// StaticVerifier is a verifier that always returns the same principal. It is
// intended only for tests and local development.
type StaticVerifier struct {
	Principal Principal
}

// Verify implements OIDCTokenVerifier.
func (s StaticVerifier) Verify(_ context.Context, _ string) (Principal, error) {
	return s.Principal, nil
}

// ErrInvalidToken is returned by verifiers when a token cannot be accepted.
var ErrInvalidToken = errors.New("invalid bearer token")

// ErrTokenExpired is returned by verifiers when a token has expired.
var ErrTokenExpired = errors.New("token expired")

// Claims is the standard OIDC claim set used by the platform. Concrete
// verifiers may carry additional claims but these are guaranteed.
type Claims struct {
	Subject   string    `json:"sub"`
	Issuer    string    `json:"iss"`
	Audience  []string  `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
	Email     string    `json:"email,omitempty"`
	Scope     string    `json:"scope,omitempty"`
	RealmAccess *RealmAccess `json:"realm_access,omitempty"`
}

// RealmAccess is the Keycloak-specific role mapping embedded in tokens.
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// TokenVerifierFunc adapts a function into an OIDCTokenVerifier.
type TokenVerifierFunc func(ctx context.Context, bearer string) (Principal, error)

// Verify implements OIDCTokenVerifier.
func (f TokenVerifierFunc) Verify(ctx context.Context, bearer string) (Principal, error) {
	return f(ctx, bearer)
}

// ParseScopes splits a space-delimited OIDC scope string into a slice.
func ParseScopes(s string) Scopes {
	out := Scopes{}
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
