package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a domain-level error with code and HTTP status.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError.
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an existing error with AppError context.
func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// Pre-defined common errors
var (
	ErrBadRequest          = New("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrUnauthorized        = New("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden           = New("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrNotFound            = New("NOT_FOUND", "Not found", http.StatusNotFound)
	ErrInternalServerError = New("INTERNAL_SERVER_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrConflict            = New("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrValidation          = New("VALIDATION_ERROR", "Validation failed", http.StatusBadRequest)
	ErrRateLimitExceeded   = New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests)
)

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*AppError)
	return ok
}
