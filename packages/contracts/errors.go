// Package contracts — typed domain errors. Use errors.Is / errors.As to match.
package contracts

import "errors"

var (
	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned for unique-constraint violations or version conflicts.
	ErrConflict = errors.New("conflict")

	// ErrValidation is returned when input fails domain validation.
	ErrValidation = errors.New("validation error")

	// ErrEvidenceMissing is returned when a claim lacks supporting evidence.
	ErrEvidenceMissing = errors.New("evidence missing")

	// ErrUnauthorized is returned when the caller is not authenticated.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the caller is authenticated but lacks permissions.
	ErrForbidden = errors.New("forbidden")

	// ErrRateLimited is returned when the caller has exceeded rate limits.
	ErrRateLimited = errors.New("rate limited")

	// ErrBudgetExceeded is returned when the AI gateway's daily budget is exhausted.
	ErrBudgetExceeded = errors.New("budget exceeded")
)
