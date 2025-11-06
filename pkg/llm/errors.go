package llm

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidProvider is returned when an unsupported provider is specified
	ErrInvalidProvider = errors.New("invalid provider")

	// ErrInvalidAPIKey is returned when the API key is missing or invalid
	ErrInvalidAPIKey = errors.New("invalid or missing API key")

	// ErrInvalidRequest is returned when the request is malformed
	ErrInvalidRequest = errors.New("invalid request")

	// ErrRateLimitExceeded is returned when rate limits are hit
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrQuotaExceeded is returned when quota is exceeded
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrModelNotFound is returned when the specified model doesn't exist
	ErrModelNotFound = errors.New("model not found")

	// ErrContextLengthExceeded is returned when the context is too long
	ErrContextLengthExceeded = errors.New("context length exceeded")

	// ErrTimeout is returned when a request times out
	ErrTimeout = errors.New("request timeout")

	// ErrServiceUnavailable is returned when the service is unavailable
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrUnknown is returned for unknown errors
	ErrUnknown = errors.New("unknown error")
)

// Error represents an LLM API error
type Error struct {
	// Type is the error type
	Type ErrorType

	// Message is the error message
	Message string

	// Provider is the LLM provider
	Provider Provider

	// StatusCode is the HTTP status code (if applicable)
	StatusCode int

	// OriginalError is the underlying error
	OriginalError error
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeInvalidRequest        ErrorType = "invalid_request"
	ErrorTypeAuthentication        ErrorType = "authentication"
	ErrorTypeRateLimit             ErrorType = "rate_limit"
	ErrorTypeQuota                 ErrorType = "quota_exceeded"
	ErrorTypeModelNotFound         ErrorType = "model_not_found"
	ErrorTypeContextLengthExceeded ErrorType = "context_length_exceeded"
	ErrorTypeTimeout               ErrorType = "timeout"
	ErrorTypeServiceUnavailable    ErrorType = "service_unavailable"
	ErrorTypeUnknown               ErrorType = "unknown"
)

// Error implements the error interface
func (e *Error) Error() string {
	if e.OriginalError != nil {
		return fmt.Sprintf("%s error from %s: %s (original: %v)",
			e.Type, e.Provider, e.Message, e.OriginalError)
	}
	return fmt.Sprintf("%s error from %s: %s", e.Type, e.Provider, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.OriginalError
}

// Is checks if the error matches the target error
func (e *Error) Is(target error) bool {
	switch e.Type {
	case ErrorTypeInvalidRequest:
		return errors.Is(target, ErrInvalidRequest)
	case ErrorTypeAuthentication:
		return errors.Is(target, ErrInvalidAPIKey)
	case ErrorTypeRateLimit:
		return errors.Is(target, ErrRateLimitExceeded)
	case ErrorTypeQuota:
		return errors.Is(target, ErrQuotaExceeded)
	case ErrorTypeModelNotFound:
		return errors.Is(target, ErrModelNotFound)
	case ErrorTypeContextLengthExceeded:
		return errors.Is(target, ErrContextLengthExceeded)
	case ErrorTypeTimeout:
		return errors.Is(target, ErrTimeout)
	case ErrorTypeServiceUnavailable:
		return errors.Is(target, ErrServiceUnavailable)
	default:
		return errors.Is(target, ErrUnknown)
	}
}

// NewError creates a new LLM error
func NewError(provider Provider, errType ErrorType, message string, originalErr error) *Error {
	return &Error{
		Type:          errType,
		Message:       message,
		Provider:      provider,
		OriginalError: originalErr,
	}
}

// IsRetryable returns true if the error is retryable
func IsRetryable(err error) bool {
	var llmErr *Error
	if errors.As(err, &llmErr) {
		switch llmErr.Type {
		case ErrorTypeRateLimit, ErrorTypeTimeout, ErrorTypeServiceUnavailable:
			return true
		}
	}
	return false
}

// IsTemporary returns true if the error is temporary
func IsTemporary(err error) bool {
	var llmErr *Error
	if errors.As(err, &llmErr) {
		switch llmErr.Type {
		case ErrorTypeRateLimit, ErrorTypeTimeout, ErrorTypeServiceUnavailable:
			return true
		}
	}
	return false
}
