// Package auth provides shared authentication and authorization primitives.
// The identity service owns user records; this package defines the
// cross-service Principal type that other services use to make authorization
// decisions without re-implementing role/scope logic.
package auth

import (
	"context"
	"fmt"
	"time"
)

// Principal is the identity of the caller at the point a request is
// authorised. It carries an opaque UserID, the issuer (OIDC provider URL),
// the granted scopes, and an expiry. It is intentionally a value type so it
// is safe to pass around.
type Principal struct {
	UserID    string
	Subject   string // OIDC sub claim
	Issuer    string
	Email     string
	Scopes    Scopes
	Roles     []string
	ExpiresAt time.Time
}

// IsAuthenticated reports whether the principal has a non-empty UserID.
func (p Principal) IsAuthenticated() bool {
	return p.UserID != ""
}

// IsAnonymous reports the inverse of IsAuthenticated.
func (p Principal) IsAnonymous() bool { return !p.IsAuthenticated() }

// HasScope reports whether the principal has been granted the given scope.
func (p Principal) HasScope(scope string) bool {
	for _, s := range p.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasRole reports whether the principal has been granted the given role.
func (p Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Can reports whether the principal satisfies any of the given scope
// requirements. An empty requirements slice is always allowed (the call site
// is responsible for default-deny for sensitive operations).
func (p Principal) Can(requirements ...string) bool {
	if len(requirements) == 0 {
		return true
	}
	for _, want := range requirements {
		if p.HasScope(want) {
			return true
		}
	}
	return false
}

// Scopes is the list of permission scopes granted to a Principal.
type Scopes []string

// Standard scope constants used across the platform. These are NOT role
// names; they are fine-grained capabilities. Roles are mapped to scopes by
// the identity service's RBAC table.
const (
	// Read scopes.
	ScopeBillRead      = "bill:read"
	ScopeBillWrite     = "bill:write"
	ScopeBillModerate  = "bill:moderate"
	ScopeSearchRead    = "search:read"
	ScopeDocumentsRead = "documents:read"
	ScopeEvidenceRead  = "evidence:read"
	ScopeEvidenceWrite = "evidence:write"
	ScopeAIAsk         = "ai:ask"
	ScopeAIModerate    = "ai:moderate"
	ScopeNotificationRead  = "notification:read"
	ScopeNotificationWrite = "notification:write"
	ScopeUserAdmin     = "user:admin"

	// Internal scopes (only services hold these).
	ScopeIngestionService = "service:ingestion"
	ScopeIntelligenceService = "service:intelligence"
	ScopeLegislationService  = "service:legislation"
)

// Require checks that the principal holds the given scope. It returns a
// typed *ForbiddenError if not.
func Require(p Principal, scope string) error {
	if !p.HasScope(scope) {
		return &ForbiddenError{Required: scope, Principal: p.UserID}
	}
	return nil
}

// RequireAny checks that the principal holds at least one of the given
// scopes.
func RequireAny(p Principal, scopes ...string) error {
	if p.Can(scopes...) {
		return nil
	}
	return &ForbiddenError{Required: fmt.Sprintf("%v", scopes), Principal: p.UserID}
}

// ForbiddenError is returned by Require when the principal is missing a scope.
type ForbiddenError struct {
	Required string
	Principal string
}

// Error implements error.
func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("principal %s missing required scope: %s", e.Principal, e.Required)
}

// Is allows errors.Is to match any ForbiddenError.
func (e *ForbiddenError) Is(target error) bool {
	_, ok := target.(*ForbiddenError)
	return ok
}

// ctxKey is the context key type for principal storage.
type ctxKey int

const principalKey ctxKey = 0

// WithPrincipal returns a context carrying the given principal.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFromContext returns the principal stored in ctx, or an anonymous
// principal if none is present.
func PrincipalFromContext(ctx context.Context) Principal {
	if v, ok := ctx.Value(principalKey).(Principal); ok {
		return v
	}
	return Principal{}
}

// Anonymous is the zero-value principal (no user).
func Anonymous() Principal { return Principal{} }
