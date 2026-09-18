package oidc

import (
        "context"
        "crypto/rand"
        "crypto/rsa"
        "encoding/base64"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"

        "github.com/go-jose/go-jose/v3"
        "github.com/go-jose/go-jose/v3/jwt"
)

// makeTestJWT creates a JWT with the given claims (header.payload.signature).
// For testing only — the signature is a dummy value (not cryptographically valid).
// Used by the DevVerifier tests; the KeycloakVerifier tests use signTestJWT
// instead, which produces a real RS256 signature against a private key.
func makeTestJWT(claims auth.Claims) string {
        header := map[string]string{"alg": "RS256", "kid": "test", "typ": "JWT"}
        headerBytes, _ := json.Marshal(header)
        payloadBytes, _ := json.Marshal(claims)
        return base64.RawURLEncoding.EncodeToString(headerBytes) + "." +
                base64.RawURLEncoding.EncodeToString(payloadBytes) + "." +
                "dummy-signature"
}

// testJWKS is a per-test JWKS server + private key. Tokens signed by
// privKey are verifiable against jwksURL; tokens signed by any other key
// are not.
type testJWKS struct {
        privKey *rsa.PrivateKey
        kid     string
        jwksURL string
        server  *httptest.Server
}

// newTestJWKS spins up a httptest.Server that serves a JWKS containing the
// public half of a freshly-generated RSA key. The returned privKey can be
// used to sign tokens that the verifier will accept.
func newTestJWKS(t *testing.T, kid string) *testJWKS {
        t.Helper()
        priv, err := rsa.GenerateKey(rand.Reader, 2048)
        if err != nil {
                t.Fatalf("rsa.GenerateKey: %v", err)
        }
        pubJWK := jose.JSONWebKey{
                Key:       &priv.PublicKey,
                KeyID:     kid,
                Algorithm: string(jose.RS256),
        }
        jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pubJWK}}
        jwksBytes, _ := json.Marshal(jwks)

        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                _, _ = w.Write(jwksBytes)
        }))
        t.Cleanup(srv.Close)

        return &testJWKS{
                privKey: priv,
                kid:     kid,
                jwksURL: srv.URL,
                server:  srv,
        }
}

// sign signs claims with privKey under the configured kid and returns the
// compact-serialized JWT.
func (j *testJWKS) sign(t *testing.T, claims any) string {
        t.Helper()
        signingKey := jose.SigningKey{
                Algorithm: jose.RS256,
                Key:       jose.JSONWebKey{Key: j.privKey, KeyID: j.kid, Algorithm: string(jose.RS256)},
        }
        signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{EmbedJWK: false})
        if err != nil {
                t.Fatalf("jose.NewSigner: %v", err)
        }
        payload, err := json.Marshal(claims)
        if err != nil {
                t.Fatalf("json.Marshal: %v", err)
        }
        tok, err := signer.Sign(payload)
        if err != nil {
                t.Fatalf("signer.Sign: %v", err)
        }
        s, err := tok.CompactSerialize()
        if err != nil {
                t.Fatalf("CompactSerialize: %v", err)
        }
        return s
}

// jwtClaims is the JSON shape we sign in tests. NumericDate fields are used
// so exp/iat encode as Unix timestamps per RFC 7519. Audience uses
// jwt.Audience which serialises as either a string (single value) or array,
// matching the JWT spec.
type jwtClaims struct {
        Subject     string            `json:"sub"`
        Issuer      string            `json:"iss"`
        Audience    jwt.Audience      `json:"aud,omitempty"`
        ExpiresAt   *jwt.NumericDate   `json:"exp,omitempty"`
        IssuedAt    *jwt.NumericDate   `json:"iat,omitempty"`
        Email       string            `json:"email,omitempty"`
        Scope       string            `json:"scope,omitempty"`
        RealmAccess *auth.RealmAccess `json:"realm_access,omitempty"`
}

const (
        testIssuer   = "https://idp.example.com/realms/civic"
        testAudience = "civic-intelligence"
        testKID      = "test-key-1"
)

// newTestVerifier wires a KeycloakVerifier at the test JWKS server with
// the platform's standard issuer/audience.
func newTestVerifier(j *testJWKS) *KeycloakVerifier {
        return NewKeycloakVerifier(testIssuer, testAudience, j.jwksURL)
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

// === KeycloakVerifier tests (P0-3 / issue #59) ===
//
// These tests stand up a httptest.Server that serves a JWKS for a freshly
// generated RSA key, then sign tokens with that key. Each test exercises a
// distinct failure mode: valid token, expired, wrong issuer, wrong audience,
// invalid signature, malformed token, alg=none header-injection attack,
// unknown kid, and iat-in-the-future.

func TestKeycloakVerifier_ValidToken(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)
        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
                Email:     "user@example.com",
                Scope:     "bill:read",
        })

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
        if !p.HasScope("bill:read") {
                t.Errorf("expected scope bill:read")
        }
        if !p.ExpiresAt.IsZero() && !p.ExpiresAt.After(now) {
                t.Errorf("expected ExpiresAt in the future, got %v", p.ExpiresAt)
        }
}

func TestKeycloakVerifier_ExpiredToken(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)
        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)), // expired
                IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
        })

        _, err := v.Verify(context.Background(), token)
        if err != auth.ErrTokenExpired {
                t.Errorf("expected ErrTokenExpired, got %v", err)
        }
}

func TestKeycloakVerifier_WrongIssuer(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)
        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    "https://evil.example.com/realms/impostor",
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected error for wrong issuer, got nil")
        }
        if !strings.Contains(err.Error(), "invalid issuer") {
                t.Errorf("expected 'invalid issuer' in error, got %v", err)
        }
}

func TestKeycloakVerifier_WrongAudience(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)
        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{"some-other-audience"},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected error for wrong audience, got nil")
        }
        if !strings.Contains(err.Error(), "invalid audience") {
                t.Errorf("expected 'invalid audience' in error, got %v", err)
        }
}

func TestKeycloakVerifier_InvalidSignature(t *testing.T) {
        // Two JWKS servers, each with its own key. We sign with one key and
        // verify against the other — the signature must not validate.
        signerJWKS := newTestJWKS(t, "signer-key")
        verifierJWKS := newTestJWKS(t, "signer-key") // same kid, different key

        v := NewKeycloakVerifier(testIssuer, testAudience, verifierJWKS.jwksURL)
        now := time.Now().UTC()
        token := signerJWKS.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected signature verification to fail, got nil")
        }
        if !strings.Contains(err.Error(), "signature verification failed") {
                t.Errorf("expected 'signature verification failed' in error, got %v", err)
        }
}

func TestKeycloakVerifier_MalformedToken(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)

        cases := []struct {
                name  string
                token string
        }{
                {"empty", ""},
                {"one-segment", "just-header"},
                {"two-segments", "header.payload"},
                {"four-segments", "a.b.c.d"},
                {"garbage", "not-a-jwt"},
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        _, err := v.Verify(context.Background(), tc.token)
                        if err == nil {
                                t.Errorf("expected error for %q, got nil", tc.token)
                        }
                })
        }
}

func TestKeycloakVerifier_UnknownKid(t *testing.T) {
        // The token header references a kid that does not exist in the JWKS.
        // Even if the signature were valid for some other key, we must reject
        // it because the OIDC provider does not advertise the signing key.
        j := newTestJWKS(t, "known-kid")
        v := newTestVerifier(j)

        // Manually build a token whose header carries a different kid. We use
        // the existing signer (which signs with "known-kid") and then rewrite
        // the header — but that breaks the signature, which is exactly what we
        // want to verify against an unknown-kid lookup path.
        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })

        // Rewrite the kid in the header to an unknown value. This invalidates
        // the signature (different header bytes), but the verifier should fail
        // at the JWKS-lookup step before even attempting signature validation.
        parts := strings.SplitN(token, ".", 3)
        headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
        var hdr map[string]any
        _ = json.Unmarshal(headerBytes, &hdr)
        hdr["kid"] = "unknown-kid"
        newHeader, _ := json.Marshal(hdr)
        parts[0] = base64.RawURLEncoding.EncodeToString(newHeader)
        tampered := strings.Join(parts, ".")

        _, err := v.Verify(context.Background(), tampered)
        if err == nil {
                t.Fatal("expected error for unknown kid, got nil")
        }
        if !strings.Contains(err.Error(), "no JWKS key matches kid") {
                t.Errorf("expected 'no JWKS key matches kid' in error, got %v", err)
        }
}

func TestKeycloakVerifier_AlgNoneRejected(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)

        // Build a token whose header advertises alg=none — a classic JWT
        // header-injection attack. The verifier MUST reject any algorithm
        // other than RS256 before even looking at the signature.
        now := time.Now().UTC()
        claimsJSON, _ := json.Marshal(jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })
        header := map[string]string{"alg": "none", "kid": testKID, "typ": "JWT"}
        headerJSON, _ := json.Marshal(header)
        token := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
                base64.RawURLEncoding.EncodeToString(claimsJSON) + "." +
                "" // empty signature segment

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected error for alg=none, got nil")
        }
        // alg=none is rejected either at parse time (go-jose may refuse empty
        // signature segments) or at the alg check. Either path is acceptable
        // — the requirement is that the token MUST be rejected.
        msg := err.Error()
        if !strings.Contains(msg, "unsupported alg") &&
                !strings.Contains(msg, "invalid JWT") &&
                !strings.Contains(msg, "invalid token") {
                t.Errorf("expected alg/parse error, got %v", err)
        }
}

func TestKeycloakVerifier_IatInFuture(t *testing.T) {
        j := newTestJWKS(t, testKID)
        v := newTestVerifier(j)
        now := time.Now().UTC()
        // iat is 5 minutes in the future — beyond the iatSkew (30s) tolerance.
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(5 * time.Minute)),
        })

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected error for iat in the future, got nil")
        }
        if !strings.Contains(err.Error(), "iat is in the future") {
                t.Errorf("expected 'iat is in the future' in error, got %v", err)
        }
}

// TestKeycloakVerifier_DoesNotFallbackToDev verifies that, when the JWKS
// endpoint is unreachable, the verifier fails closed — it does NOT silently
// fall back to trust-the-claims (DevVerifier) behavior. This is the
// regression guard for P0-3.
func TestKeycloakVerifier_DoesNotFallbackToDev(t *testing.T) {
        j := newTestJWKS(t, testKID)
        // Point the verifier at an unreachable URL (httptest server closed).
        j.server.Close()
        v := newTestVerifier(j)

        now := time.Now().UTC()
        token := j.sign(t, jwtClaims{
                Subject:   "user-123",
                Issuer:    testIssuer,
                Audience:  jwt.Audience{testAudience},
                ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
                IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
        })

        _, err := v.Verify(context.Background(), token)
        if err == nil {
                t.Fatal("expected error when JWKS is unreachable, got nil — verifier is silently trusting claims (P0-3 regression)")
        }
}
