// Package errors wraps the standard library errors package with validation helpers,
// typed errors, and fmt-based wrapping—same ideas, a bit nicer to use here.
package errors

import (
	"errors"
	"fmt"
	"runtime"
)

func New(msg string) error {
	return errors.New(msg)
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

type TypedError struct {
	Type string
	Err  error
}

func (e *TypedError) Error() string {
	return fmt.Sprintf("[%s] %v", e.Type, e.Err)
}

func (e *TypedError) Unwrap() error { return e.Err }

func NewTypedError(typ string, err error) error {
	return &TypedError{Type: typ, Err: err}
}

type ValidationError struct {
	Field   string
	Message string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s is not valid: %s", err.Field, err.Message)
}

func NewValidationError(field string, text string) error {
	return &ValidationError{Field: field, Message: text}
}

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func Wrapf(err error, msg string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err)
}

func WithLocation(err error) error {
	if err == nil {
		return nil
	}
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%w (at %s:%d)", err, file, line)
}
