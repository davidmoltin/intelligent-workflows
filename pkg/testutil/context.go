package testutil

import (
	"context"
	"testing"
	"time"
)

// ContextWithTimeout creates a context with timeout for testing
func ContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
}

// Context creates a basic context for testing
func Context(t *testing.T) context.Context {
	t.Helper()
	return ContextWithTimeout(t, 30*time.Second)
}

// CancelableContext creates a cancelable context for testing
func CancelableContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return ctx, cancel
}
