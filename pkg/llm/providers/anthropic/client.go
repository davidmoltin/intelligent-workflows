package anthropic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/llm"
	"github.com/liushuangls/go-anthropic/v2"
)

// Client implements the LLM Client interface for Anthropic
type Client struct {
	client       *anthropic.Client
	config       *llm.Config
	capabilities *llm.Capabilities
}

// NewClient creates a new Anthropic client
func NewClient(config *llm.Config) (*Client, error) {
	if config.APIKey == "" {
		return nil, llm.ErrInvalidAPIKey
	}

	opts := []anthropic.ClientOption{}

	if config.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(config.BaseURL))
	}

	if config.Timeout > 0 {
		httpClient := &http.Client{
			Timeout: config.Timeout,
		}
		opts = append(opts, anthropic.WithHTTPClient(httpClient))
	}

	client := anthropic.NewClient(config.APIKey, opts...)

	c := &Client{
		client:       client,
		config:       config,
		capabilities: buildCapabilities(),
	}

	return c, nil
}

// Chat sends a chat completion request
func (c *Client) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	// Build the Anthropic request
	anthropicReq := c.buildAnthropicRequest(req)

	// Execute the request with retries
	var resp anthropic.MessagesResponse
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}
		}

		resp, err = c.client.CreateMessages(ctx, anthropicReq)
		if err == nil {
			break
		}

		// Check if error is retryable
		if !llm.IsRetryable(c.mapError(err)) {
			break
		}
	}

	if err != nil {
		return nil, c.mapError(err)
	}

	return c.mapResponse(&resp, req.Model), nil
}

// StreamChat sends a streaming chat completion request
// Note: Streaming support can be added when SDK API is better understood
func (c *Client) StreamChat(ctx context.Context, req *llm.ChatRequest, handler llm.StreamHandler) error {
	// For now, fall back to regular chat and simulate streaming
	resp, err := c.Chat(ctx, req)
	if err != nil {
		return err
	}

	// Send content as single chunk
	if err := handler(&llm.StreamChunk{
		Delta:      resp.Content,
		IsComplete: false,
	}); err != nil {
		return err
	}

	// Send final chunk
	return handler(&llm.StreamChunk{
		Delta:        "",
		IsComplete:   true,
		Usage:        resp.Usage,
		FinishReason: resp.FinishReason,
	})
}

// GetCapabilities returns the capabilities of the Anthropic provider
func (c *Client) GetCapabilities() *llm.Capabilities {
	return c.capabilities
}

// GetProvider returns the provider type
func (c *Client) GetProvider() llm.Provider {
	return llm.ProviderAnthropic
}

// Close closes the client
func (c *Client) Close() error {
	// Anthropic client doesn't require explicit cleanup
	return nil
}

// buildAnthropicRequest converts our request to Anthropic format
func (c *Client) buildAnthropicRequest(req *llm.ChatRequest) anthropic.MessagesRequest {
	model := req.Model
	if model == "" {
		model = c.config.DefaultModel
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	// Convert messages
	messages := make([]anthropic.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		// Skip system messages (handled separately)
		if msg.Role == llm.RoleSystem {
			continue
		}

		messages = append(messages, anthropic.Message{
			Role: anthropic.ChatRole(msg.Role),
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		})
	}

	anthropicReq := anthropic.MessagesRequest{
		Model:    anthropic.Model(model),
		Messages: messages,
	}

	// Set system prompt if provided
	if req.SystemPrompt != "" {
		anthropicReq.System = req.SystemPrompt
	}

	// Set max tokens
	if req.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.MaxTokens
	} else {
		anthropicReq.MaxTokens = 4096 // Default
	}

	// Set temperature
	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		anthropicReq.Temperature = &temp
	}

	// Set top_p
	if req.TopP > 0 {
		topP := float32(req.TopP)
		anthropicReq.TopP = &topP
	}

	// Set stop sequences
	if len(req.StopSequences) > 0 {
		anthropicReq.StopSequences = req.StopSequences
	}

	return anthropicReq
}

// mapResponse converts Anthropic response to our format
func (c *Client) mapResponse(resp *anthropic.MessagesResponse, model string) *llm.ChatResponse {
	// Extract content
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.GetText()
		}
	}

	return &llm.ChatResponse{
		ID:       resp.ID,
		Content:  content,
		Model:    string(resp.Model),
		Provider: llm.ProviderAnthropic,
		Usage: &llm.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		FinishReason: string(resp.StopReason),
		CreatedAt:    time.Now(),
	}
}

// mapError converts Anthropic errors to our error format
func (c *Client) mapError(err error) error {
	if err == nil {
		return nil
	}

	// Try to extract Anthropic API error
	var apiErr *anthropic.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.IsInvalidRequestErr():
			return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeInvalidRequest, apiErr.Message, err)
		case apiErr.IsAuthenticationErr():
			return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeAuthentication, apiErr.Message, err)
		case apiErr.IsRateLimitErr():
			return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeRateLimit, apiErr.Message, err)
		case apiErr.IsOverloadedErr():
			return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeServiceUnavailable, apiErr.Message, err)
		default:
			return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeUnknown, apiErr.Message, err)
		}
	}

	// Check for context errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeTimeout, "request timeout", err)
	}

	// Default unknown error
	return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeUnknown, err.Error(), err)
}

// validateRequest validates the request
func (c *Client) validateRequest(req *llm.ChatRequest) error {
	if len(req.Messages) == 0 {
		return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeInvalidRequest, "messages cannot be empty", nil)
	}

	if req.MaxTokens > 8192 {
		return llm.NewError(llm.ProviderAnthropic, llm.ErrorTypeInvalidRequest,
			fmt.Sprintf("max_tokens %d exceeds limit of 8192", req.MaxTokens), nil)
	}

	return nil
}

// buildCapabilities returns the capabilities of Anthropic
func buildCapabilities() *llm.Capabilities {
	return &llm.Capabilities{
		Provider:                llm.ProviderAnthropic,
		SupportsStreaming:       true,
		SupportsSystemPrompt:    true,
		MaxTokensLimit:          8192,
		MaxContextWindow:        200000,
		SupportsFunctionCalling: true,
		SupportsVision:          true,
		Models: []llm.ModelInfo{
			{
				ID:                    "claude-3-5-sonnet-20241022",
				Name:                  "Claude 3.5 Sonnet",
				Description:           "Most intelligent model, ideal for complex tasks",
				ContextWindow:         200000,
				MaxOutputTokens:       8192,
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
				SupportsVision:        true,
				SupportsFunctions:     true,
			},
			{
				ID:                    "claude-3-5-haiku-20241022",
				Name:                  "Claude 3.5 Haiku",
				Description:           "Fastest and most compact model, ideal for near-instant responsiveness",
				ContextWindow:         200000,
				MaxOutputTokens:       8192,
				InputPricePerMillion:  0.8,
				OutputPricePerMillion: 4.0,
				SupportsVision:        false,
				SupportsFunctions:     true,
			},
			{
				ID:                    "claude-3-opus-20240229",
				Name:                  "Claude 3 Opus",
				Description:           "Powerful model for highly complex tasks",
				ContextWindow:         200000,
				MaxOutputTokens:       4096,
				InputPricePerMillion:  15.0,
				OutputPricePerMillion: 75.0,
				SupportsVision:        true,
				SupportsFunctions:     true,
			},
		},
	}
}
