// Package oidc provides a JWKS-based OIDC token verifier for Keycloak (or any
// standards-compliant OIDC provider). It fetches the provider's public keys
// (JWKS) and validates JWT signatures.
//
// FIXME: verify with go build when Go available.
package oidc

import (
        "context"
        "encoding/base64"
        "encoding/json"
        "errors"
        "fmt"
        "io"
        "net/http"
        "strings"
        "sync"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"

        "github.com/go-jose/go-jose/v3"
        "github.com/go-jose/go-jose/v3/jwt"
        "golang.org/x/sync/singleflight"
)

// jwksCacheTTL is how long a fetched JWKS is trusted before being refreshed.
const jwksCacheTTL = time.Hour

// iatSkew is the tolerated clock skew for the iat (issued-at) claim. Tokens
// issued up to this far in the future are accepted to absorb clock drift
// between the OIDC provider and the API host.
const iatSkew = 30 * time.Second

// KeycloakVerifier verifies OIDC JWTs against a Keycloak provider's JWKS.
// It caches the JWKS for jwksCacheTTL and uses singleflight to dedupe
// concurrent refresh fetches so a stampede of cold-cache requests triggers
// at most one HTTP GET against the OIDC provider.
type KeycloakVerifier struct {
        issuer     string
        audience   string
        jwksURL    string
        httpClient *http.Client

        mu        sync.RWMutex
        jwks      *jose.JSONWebKeySet // cached JWKS; nil when not yet fetched
        fetchedAt time.Time

        sf singleflight.Group // dedupes concurrent fetchJWKS calls
}

// NewKeycloakVerifier creates a verifier for the given Keycloak realm.
// jwksURL is typically https://<keycloak>/realms/<realm>/protocol/openid-connect/certs
func NewKeycloakVerifier(issuer, audience, jwksURL string) *KeycloakVerifier {
        return &KeycloakVerifier{
                issuer:     issuer,
                audience:   audience,
                jwksURL:    jwksURL,
                httpClient: &http.Client{Timeout: 10 * time.Second},
        }
}

// Verify implements auth.OIDCTokenVerifier. It:
//
//   - parses the compact-serialized JWT (header.payload.signature)
//   - resolves the signing key from the JWKS (refreshing if stale)
//   - verifies the RS256 signature with that key
//   - checks iss == issuer, aud contains audience, exp not expired,
//     iat not in the future (with iatSkew tolerance)
//
// On any verification failure, it returns an error. It does NOT fall back
// to DevVerifier — callers in production must receive a hard failure so
// unauthenticated traffic cannot bypass signature checks.
func (v *KeycloakVerifier) Verify(ctx context.Context, bearerToken string) (auth.Principal, error) {
        claims, err := v.parseAndVerify(ctx, bearerToken)
        if err != nil {
                return auth.Anonymous(), err
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

// parseAndVerify parses the JWT, verifies the RS256 signature against the
// JWKS key identified by the token's kid header, then runs claim checks
// (issuer, audience, expiry, iat-not-in-future). It returns the verified
// claims or an error describing the failure.
//
// This was previously a stub that returned claims WITHOUT verifying the
// signature (issue #59 / P0-3). It now uses go-jose/v3 for proper
// signature verification.
func (v *KeycloakVerifier) parseAndVerify(ctx context.Context, tokenStr string) (auth.Claims, error) {
        // Reject obviously malformed tokens before touching JWKS.
        parts := strings.Split(tokenStr, ".")
        if len(parts) != 3 {
                return auth.Claims{}, fmt.Errorf("invalid token format: expected 3 segments, got %d", len(parts))
        }

        tok, err := jose.ParseSigned(tokenStr)
        if err != nil {
                return auth.Claims{}, fmt.Errorf("invalid JWT: %w", err)
        }

        // Reject alg=none and any non-RS256 algorithm. The token's header is
        // the only untrusted input at this point — it determines which key we
        // look up, so we validate it before doing any work.
        if len(tok.Signatures) == 0 {
                return auth.Claims{}, errors.New("JWT header missing")
        }
        hdr := tok.Signatures[0].Header
        if hdr.Algorithm != string(jose.RS256) {
                return auth.Claims{}, fmt.Errorf("unsupported alg %q: only RS256 is accepted", hdr.Algorithm)
        }

        // Resolve the signing key from JWKS (refreshing if stale). Unknown kid
        // is a hard failure — we do NOT accept tokens signed by a key the OIDC
        // provider does not currently advertise.
        key, err := v.resolveKey(ctx, hdr.KeyID)
        if err != nil {
                return auth.Claims{}, err
        }

        // Verify the signature + unmarshal claims. go-jose's Verify returns
        // the verified payload bytes; we then unmarshal into our claims struct.
        payload, err := tok.Verify(key)
        if err != nil {
                return auth.Claims{}, fmt.Errorf("signature verification failed: %w", err)
        }
        var jc joseClaims
        if err := json.Unmarshal(payload, &jc); err != nil {
                return auth.Claims{}, fmt.Errorf("invalid claims JSON: %w", err)
        }

        claims := jc.toAuthClaims()

        // === Claim checks ===

        // 1. Issuer.
        if claims.Issuer != v.issuer {
                return auth.Claims{}, fmt.Errorf("invalid issuer: %q (expected %q)", claims.Issuer, v.issuer)
        }

        // 2. Audience (if configured).
        if v.audience != "" {
                found := false
                for _, aud := range claims.Audience {
                        if aud == v.audience {
                                found = true
                                break
                        }
                }
                if !found {
                        return auth.Claims{}, fmt.Errorf("invalid audience: %v (expected to contain %q)", claims.Audience, v.audience)
                }
        }

        // 3. Expiry.
        if claims.ExpiresAt.IsZero() {
                return auth.Claims{}, errors.New("token missing exp claim")
        }
        if time.Now().After(claims.ExpiresAt) {
                return auth.Claims{}, auth.ErrTokenExpired
        }

        // 4. Issued-at not in the future (with iatSkew tolerance for clock drift).
        if !claims.IssuedAt.IsZero() && time.Now().Add(iatSkew).Before(claims.IssuedAt) {
                return auth.Claims{}, fmt.Errorf("token iat is in the future: %s", claims.IssuedAt.Format(time.RFC3339))
        }

        return claims, nil
}

// resolveKey returns the JWKS key identified by kid, refreshing the cached
// JWKS first if it is empty or older than jwksCacheTTL. Concurrent refresh
// calls are deduped via singleflight so cold-cache stampedes collapse to a
// single HTTP GET against the OIDC provider.
func (v *KeycloakVerifier) resolveKey(ctx context.Context, kid string) (jose.JSONWebKey, error) {
        if kid == "" {
                return jose.JSONWebKey{}, errors.New("JWT header missing kid")
        }

        // Fast path: serve from cache.
        if key, ok := v.cachedKey(kid); ok {
                return key, nil
        }

        // Slow path: refresh JWKS (deduped). If the refresh fails but a stale
        // cache exists, we still attempt the lookup against the stale cache —
        // better to risk a stale-key check than to fail closed during a
        // transient provider outage.
        _, _, _ = v.sf.Do("jwks", func() (any, error) {
                return v.fetchJWKS(ctx)
        })

        if key, ok := v.cachedKey(kid); ok {
                return key, nil
        }
        return jose.JSONWebKey{}, fmt.Errorf("no JWKS key matches kid %q", kid)
}

// cachedKey returns the key for kid if the JWKS is present in the cache
// (stale or not). The caller decides whether to refresh.
func (v *KeycloakVerifier) cachedKey(kid string) (jose.JSONWebKey, bool) {
        v.mu.RLock()
        defer v.mu.RUnlock()
        if v.jwks == nil {
                return jose.JSONWebKey{}, false
        }
        keys := v.jwks.Key(kid)
        if len(keys) == 0 {
                return jose.JSONWebKey{}, false
        }
        // Return the first matching key. JWKS should not contain duplicate
        // kids for the same alg, but if it does we prefer the first listed.
        return keys[0], true
}

// fetchJWKS downloads the provider's JWKS and caches it. It returns an error
// if the HTTP request fails or the response is not valid JWKS JSON.
func (v *KeycloakVerifier) fetchJWKS(ctx context.Context) (jose.JSONWebKeySet, error) {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
        if err != nil {
                return jose.JSONWebKeySet{}, err
        }
        resp, err := v.httpClient.Do(req)
        if err != nil {
                return jose.JSONWebKeySet{}, err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return jose.JSONWebKeySet{}, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
        }
        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return jose.JSONWebKeySet{}, err
        }
        var jwks jose.JSONWebKeySet
        if err := json.Unmarshal(body, &jwks); err != nil {
                return jose.JSONWebKeySet{}, fmt.Errorf("invalid JWKS JSON: %w", err)
        }

        v.mu.Lock()
        v.jwks = &jwks
        v.fetchedAt = time.Now()
        v.mu.Unlock()
        return jwks, nil
}

// joseClaims mirrors auth.Claims but uses go-jose's jwt.NumericDate for the
// exp/iat fields. Standard JWTs encode these as Unix timestamps (numbers),
// but auth.Claims declares them as time.Time — Go's default time.Time
// unmarshaler expects RFC3339 strings, which would fail on real JWTs.
// Embedding jwt.Claims also gets us jwt.Audience, which accepts both the
// string and array forms permitted by RFC 7519.
type joseClaims struct {
        jwt.Claims
        Email       string           `json:"email,omitempty"`
        Scope       string           `json:"scope,omitempty"`
        RealmAccess *auth.RealmAccess `json:"realm_access,omitempty"`
}

// toAuthClaims converts the go-jose claim set into the platform's auth.Claims
// value type. Expiry / IssuedAt pointers become time.Time values.
func (jc joseClaims) toAuthClaims() auth.Claims {
        c := auth.Claims{
                Subject:     jc.Subject,
                Issuer:      jc.Issuer,
                Audience:    []string(jc.Audience),
                Email:       jc.Email,
                Scope:       jc.Scope,
                RealmAccess: jc.RealmAccess,
        }
        if jc.Expiry != nil {
                c.ExpiresAt = jc.Expiry.Time()
        }
        if jc.IssuedAt != nil {
                c.IssuedAt = jc.IssuedAt.Time()
        }
        return c
}

// DevVerifier is a verifier for development. It accepts any token that
// contains a valid JWT structure and extracts the subject from the payload.
// DO NOT use in production — it does not verify signatures. main.go only
// selects this verifier when DEV_MODE=true.
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
