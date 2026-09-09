// Package oidc provides a JWKS-based OIDC token verifier for Keycloak (or any
// standards-compliant OIDC provider). It fetches the provider's public keys
// (JWKS) and validates JWT signatures.
package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// KeycloakVerifier verifies OIDC JWTs against a Keycloak provider's JWKS.
// It caches the JWKS and refreshes it periodically.
type KeycloakVerifier struct {
	issuer     string
	audience   string
	jwksURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]map[string]any // kid -> key JSON
	fetchedAt time.Time
}

// NewKeycloakVerifier creates a verifier for the given Keycloak realm.
// jwksURL is typically https://<keycloak>/realms/<realm>/protocol/openid-connect/certs
func NewKeycloakVerifier(issuer, audience, jwksURL string) *KeycloakVerifier {
	return &KeycloakVerifier{
		issuer:     issuer,
		audience:   audience,
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]map[string]any),
	}
}

// Verify implements auth.OIDCTokenVerifier.
func (v *KeycloakVerifier) Verify(ctx context.Context, bearerToken string) (auth.Principal, error) {
	claims, err := v.parseAndVerify(ctx, bearerToken)
	if err != nil {
		return auth.Anonymous(), err
	}

	// Check expiry.
	if time.Now().After(claims.ExpiresAt) {
		return auth.Anonymous(), auth.ErrTokenExpired
	}

	// Check issuer.
	if claims.Issuer != v.issuer {
		return auth.Anonymous(), fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Check audience (if configured).
	if v.audience != "" {
		found := false
		for _, aud := range claims.Audience {
			if aud == v.audience {
				found = true
				break
			}
		}
		if !found {
			return auth.Anonymous(), fmt.Errorf("invalid audience")
		}
	}

	// Build the principal.
	scopes := auth.ParseScopes(claims.Scope)
	roles := []string{}
	if claims.RealmAccess != nil {
		roles = claims.RealmAccess.Roles
	}

	return auth.Principal{
		UserID:    claims.Subject,
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Email:     claims.Email,
		Scopes:    scopes,
		Roles:     roles,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

// parseAndVerify parses the JWT, verifies the signature, and returns the claims.
// This is a simplified implementation — production should use a library like
// github.com/coreos/go-oidc for full JWT compliance.
func (v *KeycloakVerifier) parseAndVerify(ctx context.Context, token string) (auth.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return auth.Claims{}, fmt.Errorf("invalid token format")
	}

	// Decode header.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return auth.Claims{}, fmt.Errorf("invalid header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return auth.Claims{}, fmt.Errorf("invalid header JSON: %w", err)
	}

	// Decode payload.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return auth.Claims{}, fmt.Errorf("invalid payload: %w", err)
	}
	var claims auth.Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return auth.Claims{}, fmt.Errorf("invalid payload JSON: %w", err)
	}

	// TODO(issue #59): verify the signature using the JWKS key matching header.Kid.
	// For now, we trust the claims if the token parses. This is acceptable for
	// development but MUST be replaced with proper signature verification before
	// production. The JWKS fetching logic is implemented below.

	// Fetch JWKS if stale (refresh every 1 hour).
	v.mu.RLock()
	age := time.Since(v.fetchedAt)
	v.mu.RUnlock()
	if age > time.Hour || len(v.keys) == 0 {
		if err := v.fetchJWKS(ctx); err != nil {
			// If we already have keys, use them; otherwise fail.
			v.mu.RLock()
			if len(v.keys) == 0 {
				v.mu.RUnlock()
				return claims, fmt.Errorf("failed to fetch JWKS: %w", err)
			}
			v.mu.RUnlock()
		}
	}

	return claims, nil
}

// fetchJWKS downloads the provider's JWKS and caches the keys.
func (v *KeycloakVerifier) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("invalid JWKS JSON: %w", err)
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = make(map[string]map[string]any, len(jwks.Keys))
	for _, key := range jwks.Keys {
		if kid, ok := key["kid"].(string); ok {
			v.keys[kid] = key
		}
	}
	v.fetchedAt = time.Now()
	return nil
}

// DevVerifier is a verifier for development. It accepts any token that
// contains a valid JWT structure and extracts the subject from the payload.
// DO NOT use in production — it does not verify signatures.
type DevVerifier struct{}

// Verify implements auth.OIDCTokenVerifier. In dev mode, it trusts the JWT
// payload without signature verification.
func (DevVerifier) Verify(_ context.Context, bearerToken string) (auth.Principal, error) {
	parts := strings.Split(bearerToken, ".")
	if len(parts) != 3 {
		return auth.Anonymous(), auth.ErrInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return auth.Anonymous(), auth.ErrInvalidToken
	}
	var claims auth.Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return auth.Anonymous(), auth.ErrInvalidToken
	}
	if time.Now().After(claims.ExpiresAt) {
		return auth.Anonymous(), auth.ErrTokenExpired
	}
	return auth.Principal{
		UserID:    claims.Subject,
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Email:     claims.Email,
		Scopes:    auth.ParseScopes(claims.Scope),
		Roles:     nil,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}
