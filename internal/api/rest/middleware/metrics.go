package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/metrics"
	"github.com/go-chi/chi/v5/middleware"
)

// Metrics returns a middleware that records HTTP metrics
func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code and size
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Record request size if available
			if r.ContentLength > 0 {
				m.HTTPRequestSize.WithLabelValues(
					r.Method,
					r.URL.Path,
				).Observe(float64(r.ContentLength))
			}

			// Process the request
			next.ServeHTTP(ww, r)

			// Record metrics after request is processed
			duration := time.Since(start).Seconds()
			status := ww.Status()
			statusStr := strconv.Itoa(status)

			// Record request count
			m.HTTPRequestsTotal.WithLabelValues(
				r.Method,
				r.URL.Path,
				statusStr,
			).Inc()

			// Record request duration
			m.HTTPDuration.WithLabelValues(
				r.Method,
				r.URL.Path,
			).Observe(duration)

			// Record response size
			responseSize := ww.BytesWritten()
			if responseSize > 0 {
				m.HTTPResponseSize.WithLabelValues(
					r.Method,
					r.URL.Path,
					statusStr,
				).Observe(float64(responseSize))
			}
		})
	}
}

// normalizeStatusCode converts status codes to ranges for high-cardinality reduction
func normalizeStatusCode(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return fmt.Sprintf("%d", status)
	}
}
