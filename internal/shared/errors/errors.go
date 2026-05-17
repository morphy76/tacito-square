// Package errors defines domain error types for Tacito Square components.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common domain conditions.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal error")
	ErrUnavailable   = errors.New("service unavailable")
)

// DomainError wraps a sentinel error with additional context.
type DomainError struct {
	Err     error
	Message string
	Code    string
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Err.Error())
}

// Unwrap returns the underlying sentinel error.
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewNotFound creates a not-found domain error.
func NewNotFound(entity, id string) *DomainError {
	return &DomainError{
		Err:     ErrNotFound,
		Message: fmt.Sprintf("%s with id %q not found", entity, id),
		Code:    "NOT_FOUND",
	}
}

// NewAlreadyExists creates an already-exists domain error.
func NewAlreadyExists(entity, name string) *DomainError {
	return &DomainError{
		Err:     ErrAlreadyExists,
		Message: fmt.Sprintf("%s with name %q already exists", entity, name),
		Code:    "ALREADY_EXISTS",
	}
}

// NewInvalidInput creates an invalid-input domain error.
func NewInvalidInput(reason string) *DomainError {
	return &DomainError{
		Err:     ErrInvalidInput,
		Message: reason,
		Code:    "INVALID_INPUT",
	}
}
