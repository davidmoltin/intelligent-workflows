package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_RegisterAndLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Test user registration
	registerReq := map[string]interface{}{
		"email":    "test@example.com",
		"username": "testuser",
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

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)
	assert.Contains(t, registerResp, "access_token")
	assert.Contains(t, registerResp, "refresh_token")

	// Test user login
	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "SecurePass123!",
	}

	body, err = json.Marshal(loginReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)
	assert.Contains(t, loginResp, "access_token")
	assert.Contains(t, loginResp, "refresh_token")
}

func TestAuth_InvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Test login with invalid credentials
	loginReq := map[string]interface{}{
		"username": "nonexistent",
		"password": "wrongpassword",
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

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_TokenRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Register a user
	registerReq := map[string]interface{}{
		"email":    "refresh@example.com",
		"username": "refreshuser",
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

	refreshToken := registerResp["refresh_token"].(string)

	// Test token refresh
	refreshReq := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, err = json.Marshal(refreshReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/auth/refresh",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var refreshResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&refreshResp)
	require.NoError(t, err)
	assert.Contains(t, refreshResp, "access_token")
	assert.Contains(t, refreshResp, "refresh_token")
}

func TestAuth_APIKeyCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Register and login
	accessToken := registerAndGetToken(t, server)

	// Create API key
	apiKeyReq := map[string]interface{}{
		"name":   "Test API Key",
		"scopes": []string{"workflows:read", "workflows:write"},
	}

	body, err := json.Marshal(apiKeyReq)
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
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var apiKeyResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiKeyResp)
	require.NoError(t, err)
	assert.Contains(t, apiKeyResp, "api_key")
	assert.Contains(t, apiKeyResp, "id")
}

func TestAuth_ProtectedEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Try to access protected endpoint without token
	resp, err := http.Get(server.BaseURL + "/api/v1/auth/me")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Register and get token
	accessToken := registerAndGetToken(t, server)

	// Try with valid token
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

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var userResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userResp)
	require.NoError(t, err)
	assert.Contains(t, userResp, "username")
	assert.Contains(t, userResp, "email")
}

// Helper function to register a user and get access token
func registerAndGetToken(t *testing.T, server *TestServer) string {
	t.Helper()

	registerReq := map[string]interface{}{
		"email":    "helper@example.com",
		"username": "helperuser",
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

	return registerResp["access_token"].(string)
}
