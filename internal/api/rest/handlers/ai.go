package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/llm"
	"go.uber.org/zap"
)

// AIHandler handles AI-related HTTP requests
type AIHandler struct {
	aiService *services.AIService
	logger    *zap.Logger
}

// NewAIHandler creates a new AI handler
func NewAIHandler(aiService *services.AIService, logger *zap.Logger) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		logger:    logger,
	}
}

// ChatRequest represents a chat API request
type ChatRequest struct {
	Messages      []MessageRequest `json:"messages"`
	Model         string           `json:"model,omitempty"`
	MaxTokens     int              `json:"max_tokens,omitempty"`
	Temperature   float64          `json:"temperature,omitempty"`
	TopP          float64          `json:"top_p,omitempty"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
	SystemPrompt  string           `json:"system_prompt,omitempty"`
	Stream        bool             `json:"stream,omitempty"`
}

// MessageRequest represents a message in the chat request
type MessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat API response
type ChatResponse struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Usage        *TokenUsageResponse    `json:"usage"`
	FinishReason string                 `json:"finish_reason"`
	CreatedAt    string                 `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsageResponse represents token usage in the response
type TokenUsageResponse struct {
	PromptTokens             int `json:"prompt_tokens"`
	CompletionTokens         int `json:"completion_tokens"`
	TotalTokens              int `json:"total_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// CapabilitiesResponse represents the capabilities response
type CapabilitiesResponse struct {
	Provider                string      `json:"provider"`
	Models                  []ModelInfo `json:"models"`
	SupportsStreaming       bool        `json:"supports_streaming"`
	SupportsSystemPrompt    bool        `json:"supports_system_prompt"`
	MaxTokensLimit          int         `json:"max_tokens_limit"`
	MaxContextWindow        int         `json:"max_context_window"`
	SupportsFunctionCalling bool        `json:"supports_function_calling"`
	SupportsVision          bool        `json:"supports_vision"`
}

// ModelInfo represents model information
type ModelInfo struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Description           string  `json:"description"`
	ContextWindow         int     `json:"context_window"`
	MaxOutputTokens       int     `json:"max_output_tokens"`
	InputPricePerMillion  float64 `json:"input_price_per_million"`
	OutputPricePerMillion float64 `json:"output_price_per_million"`
	SupportsVision        bool    `json:"supports_vision"`
	SupportsFunctions     bool    `json:"supports_functions"`
}

// InterpretRequest represents a workflow interpretation request
type InterpretRequest struct {
	Description string `json:"description"`
}

// InterpretResponse represents a workflow interpretation response
type InterpretResponse struct {
	Workflow string `json:"workflow"`
}

// Chat handles chat completion requests
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert to internal format
	messages := make([]llm.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = llm.Message{
			Role:    llm.Role(msg.Role),
			Content: msg.Content,
		}
	}

	chatReq := &llm.ChatRequest{
		Messages:      messages,
		Model:         req.Model,
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: req.StopSequences,
		SystemPrompt:  req.SystemPrompt,
	}

	// Handle streaming
	if req.Stream {
		h.handleStreamingChat(w, r, chatReq)
		return
	}

	// Handle regular chat
	resp, err := h.aiService.Chat(r.Context(), chatReq)
	if err != nil {
		h.logger.Error("chat request failed", zap.Error(err))
		respondLLMError(w, err)
		return
	}

	// Convert response
	chatResp := &ChatResponse{
		ID:           resp.ID,
		Content:      resp.Content,
		Model:        resp.Model,
		Provider:     string(resp.Provider),
		FinishReason: resp.FinishReason,
		CreatedAt:    resp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Metadata:     resp.Metadata,
	}

	if resp.Usage != nil {
		chatResp.Usage = &TokenUsageResponse{
			PromptTokens:             resp.Usage.PromptTokens,
			CompletionTokens:         resp.Usage.CompletionTokens,
			TotalTokens:              resp.Usage.TotalTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		}
	}

	respondJSON(w, http.StatusOK, chatResp)
}

// handleStreamingChat handles streaming chat requests
func (h *AIHandler) handleStreamingChat(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	encoder := json.NewEncoder(w)

	handler := func(chunk *llm.StreamChunk) error {
		// Send chunk as SSE
		if err := encoder.Encode(chunk); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := h.aiService.StreamChat(r.Context(), req, handler); err != nil {
		h.logger.Error("streaming chat failed", zap.Error(err))
		// Can't send error response after streaming started
		return
	}
}

// GetCapabilities returns the LLM provider capabilities
func (h *AIHandler) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	caps := h.aiService.GetCapabilities()

	models := make([]ModelInfo, len(caps.Models))
	for i, m := range caps.Models {
		models[i] = ModelInfo{
			ID:                    m.ID,
			Name:                  m.Name,
			Description:           m.Description,
			ContextWindow:         m.ContextWindow,
			MaxOutputTokens:       m.MaxOutputTokens,
			InputPricePerMillion:  m.InputPricePerMillion,
			OutputPricePerMillion: m.OutputPricePerMillion,
			SupportsVision:        m.SupportsVision,
			SupportsFunctions:     m.SupportsFunctions,
		}
	}

	resp := &CapabilitiesResponse{
		Provider:                string(caps.Provider),
		Models:                  models,
		SupportsStreaming:       caps.SupportsStreaming,
		SupportsSystemPrompt:    caps.SupportsSystemPrompt,
		MaxTokensLimit:          caps.MaxTokensLimit,
		MaxContextWindow:        caps.MaxContextWindow,
		SupportsFunctionCalling: caps.SupportsFunctionCalling,
		SupportsVision:          caps.SupportsVision,
	}

	respondJSON(w, http.StatusOK, resp)
}

// InterpretWorkflow interprets natural language into a workflow
func (h *AIHandler) InterpretWorkflow(w http.ResponseWriter, r *http.Request) {
	var req InterpretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Description == "" {
		respondError(w, http.StatusBadRequest, "description is required")
		return
	}

	workflow, err := h.aiService.InterpretWorkflow(r.Context(), req.Description)
	if err != nil {
		h.logger.Error("workflow interpretation failed", zap.Error(err))
		respondLLMError(w, err)
		return
	}

	resp := &InterpretResponse{
		Workflow: workflow,
	}

	respondJSON(w, http.StatusOK, resp)
}

// Helper functions

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, ErrorResponse{Error: message})
}

// respondLLMError handles LLM-specific errors
func respondLLMError(w http.ResponseWriter, err error) {
	var llmErr *llm.Error
	if ok := llm.IsTemporary(err); ok {
		respondError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// Check specific error types
	switch {
	case llm.ErrInvalidAPIKey == err:
		respondError(w, http.StatusUnauthorized, "invalid API key")
	case llm.ErrInvalidRequest == err:
		respondError(w, http.StatusBadRequest, err.Error())
	case llm.ErrRateLimitExceeded == err:
		respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
	case llm.ErrModelNotFound == err:
		respondError(w, http.StatusNotFound, "model not found")
	case llm.ErrContextLengthExceeded == err:
		respondError(w, http.StatusBadRequest, "context length exceeded")
	default:
		if llmErr != nil {
			respondError(w, http.StatusInternalServerError, llmErr.Message)
		} else {
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
	}
}
