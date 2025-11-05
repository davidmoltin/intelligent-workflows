package llm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	err := NewError(ProviderAnthropic, ErrorTypeRateLimit, "too many requests", nil)
	assert.Contains(t, err.Error(), "rate_limit")
	assert.Contains(t, err.Error(), "anthropic")
	assert.Contains(t, err.Error(), "too many requests")
}

func TestError_ErrorWithOriginal(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewError(ProviderOpenAI, ErrorTypeTimeout, "request timeout", originalErr)
	assert.Contains(t, err.Error(), "original error")
}

func TestError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewError(ProviderAnthropic, ErrorTypeUnknown, "something went wrong", originalErr)

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, originalErr, unwrapped)
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		name      string
		errType   ErrorType
		target    error
		shouldBe  bool
	}{
		{
			name:     "invalid request",
			errType:  ErrorTypeInvalidRequest,
			target:   ErrInvalidRequest,
			shouldBe: true,
		},
		{
			name:     "authentication",
			errType:  ErrorTypeAuthentication,
			target:   ErrInvalidAPIKey,
			shouldBe: true,
		},
		{
			name:     "rate limit",
			errType:  ErrorTypeRateLimit,
			target:   ErrRateLimitExceeded,
			shouldBe: true,
		},
		{
			name:     "quota",
			errType:  ErrorTypeQuota,
			target:   ErrQuotaExceeded,
			shouldBe: true,
		},
		{
			name:     "model not found",
			errType:  ErrorTypeModelNotFound,
			target:   ErrModelNotFound,
			shouldBe: true,
		},
		{
			name:     "context length exceeded",
			errType:  ErrorTypeContextLengthExceeded,
			target:   ErrContextLengthExceeded,
			shouldBe: true,
		},
		{
			name:     "timeout",
			errType:  ErrorTypeTimeout,
			target:   ErrTimeout,
			shouldBe: true,
		},
		{
			name:     "service unavailable",
			errType:  ErrorTypeServiceUnavailable,
			target:   ErrServiceUnavailable,
			shouldBe: true,
		},
		{
			name:     "unknown",
			errType:  ErrorTypeUnknown,
			target:   ErrUnknown,
			shouldBe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(ProviderAnthropic, tt.errType, "test error", nil)
			assert.Equal(t, tt.shouldBe, errors.Is(err, tt.target))
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryable  bool
	}{
		{
			name:      "rate limit is retryable",
			err:       NewError(ProviderAnthropic, ErrorTypeRateLimit, "rate limited", nil),
			retryable: true,
		},
		{
			name:      "timeout is retryable",
			err:       NewError(ProviderOpenAI, ErrorTypeTimeout, "timeout", nil),
			retryable: true,
		},
		{
			name:      "service unavailable is retryable",
			err:       NewError(ProviderAnthropic, ErrorTypeServiceUnavailable, "unavailable", nil),
			retryable: true,
		},
		{
			name:      "authentication error is not retryable",
			err:       NewError(ProviderOpenAI, ErrorTypeAuthentication, "invalid key", nil),
			retryable: false,
		},
		{
			name:      "invalid request is not retryable",
			err:       NewError(ProviderAnthropic, ErrorTypeInvalidRequest, "bad request", nil),
			retryable: false,
		},
		{
			name:      "non-llm error is not retryable",
			err:       errors.New("some error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryable(tt.err))
		})
	}
}

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		temporary bool
	}{
		{
			name:      "rate limit is temporary",
			err:       NewError(ProviderAnthropic, ErrorTypeRateLimit, "rate limited", nil),
			temporary: true,
		},
		{
			name:      "timeout is temporary",
			err:       NewError(ProviderOpenAI, ErrorTypeTimeout, "timeout", nil),
			temporary: true,
		},
		{
			name:      "service unavailable is temporary",
			err:       NewError(ProviderAnthropic, ErrorTypeServiceUnavailable, "unavailable", nil),
			temporary: true,
		},
		{
			name:      "model not found is not temporary",
			err:       NewError(ProviderOpenAI, ErrorTypeModelNotFound, "model not found", nil),
			temporary: false,
		},
		{
			name:      "non-llm error is not temporary",
			err:       errors.New("some error"),
			temporary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.temporary, IsTemporary(tt.err))
		})
	}
}
