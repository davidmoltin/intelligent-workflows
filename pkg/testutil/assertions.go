package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertJSONResponse asserts that an HTTP response has the expected status code and JSON body
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "unexpected status code")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "unexpected content type")

	if expectedBody != nil {
		expectedJSON, err := json.Marshal(expectedBody)
		require.NoError(t, err, "failed to marshal expected body")

		assert.JSONEq(t, string(expectedJSON), w.Body.String(), "unexpected response body")
	}
}

// AssertErrorResponse asserts that an HTTP response contains an error
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "unexpected status code")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err, "failed to decode response")

	if expectedMessage != "" {
		assert.Contains(t, response["error"], expectedMessage, "unexpected error message")
	}
}

// AssertNoError asserts that there is no error
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// AssertError asserts that there is an error
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
}

// AssertHTTPStatus asserts that an HTTP response has the expected status code
func AssertHTTPStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	assert.Equal(t, expected, w.Code, "unexpected HTTP status code")
}

// AssertContains asserts that a string contains a substring
func AssertContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Contains(t, s, substr, msgAndArgs...)
}

// MakeJSONRequest creates an HTTP request with JSON body
func MakeJSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	t.Helper()

	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err, "failed to marshal request body")
	}

	req := httptest.NewRequest(method, url, nil)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Body = http.NoBody // httptest doesn't need actual body
	}

	return req
}
