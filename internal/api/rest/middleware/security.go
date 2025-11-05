package middleware

import (
	"net/http"
	"os"
	"strconv"
)

// RequestSizeLimit limits the maximum size of request bodies
func RequestSizeLimit(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Enable XSS protection (legacy browsers)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Strict Transport Security (HSTS) - only if HTTPS
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Content Security Policy
			csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'"
			if cspEnv := os.Getenv("CSP_POLICY"); cspEnv != "" {
				csp = cspEnv
			}
			w.Header().Set("Content-Security-Policy", csp)

			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions Policy (formerly Feature Policy)
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(w, r)
		})
	}
}

// GetMaxRequestSize returns the maximum request size from environment or default
func GetMaxRequestSize() int64 {
	const defaultMaxSize = 10 * 1024 * 1024 // 10MB default

	if maxSizeEnv := os.Getenv("MAX_REQUEST_SIZE_MB"); maxSizeEnv != "" {
		if maxSizeMB, err := strconv.ParseInt(maxSizeEnv, 10, 64); err == nil {
			return maxSizeMB * 1024 * 1024
		}
	}

	return defaultMaxSize
}
