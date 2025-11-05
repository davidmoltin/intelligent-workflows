package handlers

import (
	"net/http"
	"os"
	"path/filepath"
)

// DocsHandler handles API documentation endpoints
type DocsHandler struct {
	docsPath string
}

// NewDocsHandler creates a new docs handler
func NewDocsHandler() *DocsHandler {
	// Get the project root directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	docsPath := filepath.Join(workDir, "docs", "api")

	return &DocsHandler{
		docsPath: docsPath,
	}
}

// ServeSwaggerUI serves the Swagger UI HTML page
func (h *DocsHandler) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	filePath := filepath.Join(h.docsPath, "swagger-ui.html")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Documentation not found", http.StatusNotFound)
		return
	}

	// Read and serve the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Error reading documentation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// ServeOpenAPISpec serves the OpenAPI specification YAML file
func (h *DocsHandler) ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	filePath := filepath.Join(h.docsPath, "openapi.yaml")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "OpenAPI specification not found", http.StatusNotFound)
		return
	}

	// Read and serve the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Error reading OpenAPI specification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// RedirectToDocs redirects /api/v1/docs to the Swagger UI
func (h *DocsHandler) RedirectToDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/api/v1/docs/ui", http.StatusMovedPermanently)
}
