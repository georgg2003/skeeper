// Package errors wraps the standard library errors package with validation helpers,
// typed errors, and fmt-based wrapping—same ideas, a bit nicer to use here.
package errors

import (
	"errors"
	"fmt"
	"runtime"
)

// New wraps errors.New for a single import path across the repo.
func New(msg string) error {
	return errors.New(msg)
}

// Is delegates to errors.Is.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As delegates to errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// AsType is a generic wrapper around errors.As for typed unwrap helpers.
func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

// Join delegates to errors.Join.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// TypedError annotates an underlying error with a short type label for logs and RPC mapping.
type TypedError struct {
	Type string
	Err  error
}

func (e *TypedError) Error() string {
	return fmt.Sprintf("[%s] %v", e.Type, e.Err)
}

func (e *TypedError) Unwrap() error { return e.Err }

// NewTypedError builds a TypedError.
func NewTypedError(typ string, err error) error {
	return &TypedError{Type: typ, Err: err}
}

// ValidationError describes invalid user or wire input for a single field.
type ValidationError struct {
	Field   string
	Message string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s is not valid: %s", err.Field, err.Message)
}

// NewValidationError constructs a ValidationError.
func NewValidationError(field string, text string) error {
	return &ValidationError{Field: field, Message: text}
}

// Wrap adds context to an error with fmt.Errorf("%s: %w", ...).
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf is like Wrap with a formatted message.
func Wrapf(err error, msg string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err)
}

// WithLocation appends file:line of the caller to the error message (debug aid).
func WithLocation(err error) error {
	if err == nil {
		return nil
	}
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%w (at %s:%d)", err, file, line)
}
