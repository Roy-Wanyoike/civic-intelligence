// Package contracts — typed domain errors. Use errors.Is / errors.As to match.
package contracts

import (
        "errors"
        "fmt"
)

var (
        // ErrConflict is returned for unique-constraint violations or version conflicts.
        ErrConflict = errors.New("conflict")

        // ErrUnauthorized is returned when the caller is not authenticated.
        ErrUnauthorized = errors.New("unauthorized")

        // ErrForbidden is returned when the caller is authenticated but lacks permissions.
        ErrForbidden = errors.New("forbidden")

        // ErrRateLimited is returned when the caller has exceeded rate limits.
        ErrRateLimited = errors.New("rate limited")

        // ErrBudgetExceeded is returned when the AI gateway's daily budget is exhausted.
        ErrBudgetExceeded = errors.New("budget exceeded")
)

// ErrNotFound is a typed error returned when an entity does not exist.
// Includes the entity type + ID so callers can produce a helpful message.
type ErrNotFound struct {
        EntityType string
        Kind       string // optional sub-type, e.g. "bill", "document", "by_id"
        ID         string
}

func (e ErrNotFound) Error() string {
        if e.EntityType != "" {
                return fmt.Sprintf("%s %q not found", e.EntityType, e.ID)
        }
        return fmt.Sprintf("entity %q not found", e.ID)
}

// IsNotFound reports whether err is an ErrNotFound (using errors.As).
func IsNotFound(err error) bool {
        var e ErrNotFound
        return errors.As(err, &e)
}

// Note: ErrValidation, ErrStageTransition, and ErrEvidenceMissing are typed
// error structs defined in country.go (alongside the types that produce them).
// Use errors.As to extract their structured fields.
