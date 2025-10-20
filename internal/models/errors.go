package models

import (
	"fmt"
	"net/http"
)

// ErrorCode represents a custom error code for the application
type ErrorCode string

const (
	// General errors
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeConflict     ErrorCode = "CONFLICT"
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"

	// Database errors
	ErrCodeDatabaseConnection  ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrCodeDatabaseQuery       ErrorCode = "DATABASE_QUERY_ERROR"
	ErrCodeDatabaseTransaction ErrorCode = "DATABASE_TRANSACTION_ERROR"

	// Node-specific errors
	ErrCodeNodeNotReachable   ErrorCode = "NODE_NOT_REACHABLE"
	ErrCodeNodeTimeout        ErrorCode = "NODE_TIMEOUT"
	ErrCodeNodeInvalidAddress ErrorCode = "NODE_INVALID_ADDRESS"
	ErrCodeNodeCheckFailed    ErrorCode = "NODE_CHECK_FAILED"

	// Service errors
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
	Internal   error                  `json:"-"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the internal error for error chain support
func (e *AppError) Unwrap() error {
	return e.Internal
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithMetadata adds metadata to the error
func (e *AppError) WithMetadata(key string, value interface{}) *AppError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// Common error constructors

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Internal:   err,
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeBadRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewValidationError(message string, details string) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

func NewDatabaseError(message string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeDatabaseQuery,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Internal:   err,
	}
}

func NewNodeNotReachableError(address string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeNodeNotReachable,
		Message:    "Node is not reachable",
		StatusCode: http.StatusServiceUnavailable,
		Internal:   err,
		Metadata: map[string]interface{}{
			"address": address,
		},
	}
}

func NewRateLimitError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeRateLimitExceeded,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
	}
}
