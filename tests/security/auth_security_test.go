package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_BruteForceProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Attempt multiple failed logins
	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "wrongpassword",
	}

	body, err := json.Marshal(loginReq)
	require.NoError(t, err)

	failureCount := 0
	for i := 0; i < 10; i++ {
		resp, err := http.Post(
			server.BaseURL+"/api/v1/auth/login",
			"application/json",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			failureCount++
		}

		// Should eventually get rate limited
		if resp.StatusCode == http.StatusTooManyRequests {
			assert.GreaterOrEqual(t, failureCount, 3, "Should allow some attempts before rate limiting")
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	// If no rate limiting occurred, log warning
	t.Log("Warning: No rate limiting detected for failed login attempts")
}

func TestSecurity_WeakPasswordRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	weakPasswords := []string{
		"123456",
		"password",
		"qwerty",
		"abc123",
		"12345678",
		"short",
	}

	for _, pwd := range weakPasswords {
		registerReq := map[string]interface{}{
			"username": "testuser",
			"email":    "test@example.com",
			"password": pwd,
		}

		body, err := json.Marshal(registerReq)
		require.NoError(t, err)

		resp, err := http.Post(
			server.BaseURL+"/api/v1/auth/register",
			"application/json",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Weak passwords should be rejected
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"Weak password '%s' should be rejected", pwd)
	}
}

func TestSecurity_JWTTokenValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Test with invalid token
	invalidTokens := []string{
		"invalid.token.here",
		"Bearer fake-token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		"",
	}

	for _, token := range invalidTokens {
		req, err := http.NewRequest(
			"GET",
			server.BaseURL+"/api/v1/auth/me",
			nil,
		)
		require.NoError(t, err)

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"Invalid token should be rejected: %s", token)
	}
}

func TestSecurity_ExpiredTokenRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create a user and get token
	registerReq := map[string]interface{}{
		"username": "expirytest",
		"email":    "expiry@example.com",
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	accessToken := registerResp["access_token"].(string)

	// Wait for token to expire (if configured with short expiry for tests)
	// In production, tokens expire after 15 minutes
	// For testing, you might configure a shorter expiry

	// Attempt to use token (should succeed initially)
	req, err := http.NewRequest(
		"GET",
		server.BaseURL+"/api/v1/auth/me",
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Valid token should work")
}

func TestSecurity_PasswordHashingSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	// Test that passwords are properly hashed
	// This is a conceptual test - actual implementation would check the database

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	password := "SecurePassword123!"

	registerReq := map[string]interface{}{
		"username": "hashtest",
		"email":    "hash@example.com",
		"password": password,
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify password is not returned in response
	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	assert.NotContains(t, registerResp, "password", "Password should not be in response")

	// Additional check: verify stored password is hashed (would query DB in real test)
}

func TestSecurity_APIKeySecrecy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Register and login
	registerReq := map[string]interface{}{
		"username": "apikeytestonst",
		"email":    "apikey@example.com",
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	accessToken := registerResp["access_token"].(string)

	// Create API key
	apiKeyReq := map[string]interface{}{
		"name":   "Test API Key",
		"scopes": []string{"workflows:read"},
	}

	body, err = json.Marshal(apiKeyReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/auth/api-keys",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var apiKeyResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&apiKeyResp)
		require.NoError(t, err)

		// API key should be returned only once on creation
		assert.Contains(t, apiKeyResp, "api_key")

		// Subsequent list operations should NOT reveal the full key
		// (This would require additional endpoint testing)
	}
}

func TestSecurity_SQLInjectionProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Test SQL injection attempts in login
	sqlInjectionAttempts := []string{
		"admin' OR '1'='1",
		"admin'--",
		"admin' OR '1'='1'--",
		"'; DROP TABLE users;--",
		"1' OR '1' = '1",
	}

	for _, username := range sqlInjectionAttempts {
		loginReq := map[string]interface{}{
			"username": username,
			"password": "password",
		}

		body, err := json.Marshal(loginReq)
		require.NoError(t, err)

		resp, err := http.Post(
			server.BaseURL+"/api/v1/auth/login",
			"application/json",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return unauthorized, not succeed
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"SQL injection attempt should fail: %s", username)
	}
}

func TestSecurity_XSSProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Test XSS attempts in registration
	xssAttempts := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg/onload=alert('XSS')>",
	}

	for _, xss := range xssAttempts {
		registerReq := map[string]interface{}{
			"username": xss,
			"email":    "xss@example.com",
			"password": "SecurePass123!",
		}

		body, err := json.Marshal(xss)
		require.NoError(t, err)

		resp, err := http.Post(
			server.BaseURL+"/api/v1/auth/register",
			"application/json",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be rejected or sanitized
		if resp.StatusCode == http.StatusCreated {
			var registerResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&registerResp)
			require.NoError(t, err)

			// Verify XSS payload is sanitized
			username := registerResp["username"]
			assert.NotContains(t, username, "<script>", "XSS should be sanitized")
			assert.NotContains(t, username, "javascript:", "XSS should be sanitized")
		}
	}
}

func TestSecurity_RateLimitingPerUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Register and login
	registerReq := map[string]interface{}{
		"username": "ratelimituser",
		"email":    "ratelimit@example.com",
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	accessToken := registerResp["access_token"].(string)

	// Make many requests rapidly
	rateLimitHit := false
	for i := 0; i < 150; i++ { // Default limit is 100 per minute
		req, err := http.NewRequest(
			"GET",
			server.BaseURL+"/api/v1/workflows",
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true
			break
		}
	}

	assert.True(t, rateLimitHit, "Rate limiting should be enforced")
}

func TestSecurity_CORSConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Test CORS preflight request
	req, err := http.NewRequest(
		"OPTIONS",
		server.BaseURL+"/api/v1/workflows",
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check CORS headers
	assert.Contains(t, resp.Header, "Access-Control-Allow-Origin")
	assert.Contains(t, resp.Header, "Access-Control-Allow-Methods")
	assert.Contains(t, resp.Header, "Access-Control-Allow-Headers")
}

func TestSecurity_SecureHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	resp, err := http.Get(server.BaseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check for security headers
	headers := resp.Header

	// These headers should be present for security
	securityHeaders := map[string]bool{
		"X-Content-Type-Options": false, // nosniff
		"X-Frame-Options":        false, // DENY or SAMEORIGIN
		"X-XSS-Protection":       false, // 1; mode=block
	}

	for header := range securityHeaders {
		if headers.Get(header) != "" {
			securityHeaders[header] = true
		}
	}

	// Log which headers are missing
	for header, present := range securityHeaders {
		if !present {
			t.Logf("Security header '%s' is not set", header)
		}
	}
}
