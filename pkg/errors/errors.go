/*
Обертка над стандартным пакетом errors, для кастомизации ошибок и гибкости.
*/
package errors

import (
	"errors"
	"fmt"
)

func New(msg string) error {
	return errors.New(msg)
}

type typedError struct {
	errType string
	text    string
}

func (err *typedError) Error() string {
	return fmt.Sprintf("[%s] %s", err.errType, err.text)
}

func NewTypedError(typ string, text string) error {
	return &typedError{errType: typ, text: text}
}

type ValidationError struct {
	field string
	text  string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s is not valid: %s", err.field, err.text)
}

func NewValidationError(field string, text string) error {
	return &ValidationError{field: field, text: text}
}

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
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
