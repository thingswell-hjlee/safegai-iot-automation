// Package errors defines standard domain error types for the SafeGAI gateway.
package errors

import "fmt"

// ErrorCode identifies the category of a domain error.
type ErrorCode string

const (
	CodeValidation ErrorCode = "VALIDATION_ERROR"
	CodeNotFound   ErrorCode = "NOT_FOUND"
	CodeConflict   ErrorCode = "CONFLICT"
	CodeInternal   ErrorCode = "INTERNAL_ERROR"
	CodeTimeout    ErrorCode = "TIMEOUT"
)

// DomainError is the base interface for all domain errors.
type DomainError interface {
	error
	Code() ErrorCode
	Retryable() bool
}

// ValidationError indicates invalid input or state.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Code() ErrorCode { return CodeValidation }
func (e *ValidationError) Retryable() bool { return false }

// NotFoundError indicates a requested resource was not found.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

func (e *NotFoundError) Code() ErrorCode { return CodeNotFound }
func (e *NotFoundError) Retryable() bool { return false }

// ConflictError indicates a state conflict (e.g. duplicate or concurrent modification).
type ConflictError struct {
	Resource string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Resource, e.Message)
}

func (e *ConflictError) Code() ErrorCode { return CodeConflict }
func (e *ConflictError) Retryable() bool { return false }

// InternalError indicates an unexpected internal failure.
type InternalError struct {
	Message string
	Cause   error
}

func (e *InternalError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("internal error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("internal error: %s", e.Message)
}

func (e *InternalError) Code() ErrorCode { return CodeInternal }
func (e *InternalError) Retryable() bool { return true }
func (e *InternalError) Unwrap() error   { return e.Cause }

// TimeoutError indicates an operation exceeded its deadline.
type TimeoutError struct {
	Operation string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout: %s", e.Operation)
}

func (e *TimeoutError) Code() ErrorCode { return CodeTimeout }
func (e *TimeoutError) Retryable() bool { return true }

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

// NewConflictError creates a new ConflictError.
func NewConflictError(resource, message string) *ConflictError {
	return &ConflictError{Resource: resource, Message: message}
}

// NewInternalError creates a new InternalError.
func NewInternalError(message string, cause error) *InternalError {
	return &InternalError{Message: message, Cause: cause}
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(operation string) *TimeoutError {
	return &TimeoutError{Operation: operation}
}
