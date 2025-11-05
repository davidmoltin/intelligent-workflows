package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/llm"
	"github.com/sashabaranov/go-openai"
)

// Client implements the LLM Client interface for OpenAI
type Client struct {
	client       *openai.Client
	config       *llm.Config
	capabilities *llm.Capabilities
}

// NewClient creates a new OpenAI client
func NewClient(config *llm.Config) (*Client, error) {
	if config.APIKey == "" {
		return nil, llm.ErrInvalidAPIKey
	}

	clientConfig := openai.DefaultConfig(config.APIKey)

	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	if config.Timeout > 0 {
		clientConfig.HTTPClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	client := openai.NewClientWithConfig(clientConfig)

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

	// Build the OpenAI request
	openaiReq := c.buildOpenAIRequest(req)

	// Execute the request with retries
	var resp openai.ChatCompletionResponse
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

		resp, err = c.client.CreateChatCompletion(ctx, openaiReq)
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

	return c.mapResponse(&resp), nil
}

// StreamChat sends a streaming chat completion request
func (c *Client) StreamChat(ctx context.Context, req *llm.ChatRequest, handler llm.StreamHandler) error {
	if err := c.validateRequest(req); err != nil {
		return err
	}

	// Build the OpenAI request
	openaiReq := c.buildOpenAIRequest(req)

	// Create streaming request
	stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return c.mapError(err)
	}
	defer stream.Close()

	var totalUsage *llm.TokenUsage
	var finishReason string

	// Process stream
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// Stream ended successfully
			break
		}
		if err != nil {
			return c.mapError(err)
		}

		// Extract delta content
		if len(response.Choices) > 0 {
			choice := response.Choices[0]
			delta := choice.Delta.Content

			if delta != "" {
				chunk := &llm.StreamChunk{
					Delta:      delta,
					IsComplete: false,
				}
				if err := handler(chunk); err != nil {
					return err
				}
			}

			// Check for finish reason
			if choice.FinishReason != "" {
				finishReason = string(choice.FinishReason)
			}
		}

		// Extract usage (typically in the last chunk)
		if response.Usage.TotalTokens > 0 {
			totalUsage = &llm.TokenUsage{
				PromptTokens:     response.Usage.PromptTokens,
				CompletionTokens: response.Usage.CompletionTokens,
				TotalTokens:      response.Usage.TotalTokens,
			}
		}
	}

	// Send final chunk
	finalChunk := &llm.StreamChunk{
		Delta:        "",
		IsComplete:   true,
		Usage:        totalUsage,
		FinishReason: finishReason,
	}
	if err := handler(finalChunk); err != nil {
		return err
	}

	return nil
}

// GetCapabilities returns the capabilities of the OpenAI provider
func (c *Client) GetCapabilities() *llm.Capabilities {
	return c.capabilities
}

// GetProvider returns the provider type
func (c *Client) GetProvider() llm.Provider {
	return llm.ProviderOpenAI
}

// Close closes the client
func (c *Client) Close() error {
	// OpenAI client doesn't require explicit cleanup
	return nil
}

// buildOpenAIRequest converts our request to OpenAI format
func (c *Client) buildOpenAIRequest(req *llm.ChatRequest) openai.ChatCompletionRequest {
	model := req.Model
	if model == "" {
		model = c.config.DefaultModel
	}
	if model == "" {
		model = openai.GPT4o
	}

	// Convert messages
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	// Add system prompt first if provided
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(llm.RoleSystem),
			Content: req.SystemPrompt,
		})
	}

	// Add conversation messages
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	openaiReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	// Set max tokens
	if req.MaxTokens > 0 {
		openaiReq.MaxTokens = req.MaxTokens
	}

	// Set temperature
	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		openaiReq.Temperature = temp
	}

	// Set top_p
	if req.TopP > 0 {
		topP := float32(req.TopP)
		openaiReq.TopP = topP
	}

	// Set stop sequences
	if len(req.StopSequences) > 0 {
		openaiReq.Stop = req.StopSequences
	}

	// Add user identifier if present in metadata
	if userID, ok := req.Metadata["user_id"]; ok {
		openaiReq.User = userID
	}

	return openaiReq
}

// mapResponse converts OpenAI response to our format
func (c *Client) mapResponse(resp *openai.ChatCompletionResponse) *llm.ChatResponse {
	// Extract content from first choice
	var content string
	var finishReason string

	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		finishReason = string(resp.Choices[0].FinishReason)
	}

	return &llm.ChatResponse{
		ID:       resp.ID,
		Content:  content,
		Model:    resp.Model,
		Provider: llm.ProviderOpenAI,
		Usage: &llm.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		FinishReason: finishReason,
		CreatedAt:    time.Unix(int64(resp.Created), 0),
	}
}

// mapError converts OpenAI errors to our error format
func (c *Client) mapError(err error) error {
	if err == nil {
		return nil
	}

	// Try to extract OpenAI API error
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case http.StatusUnauthorized:
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeAuthentication, apiErr.Message, err)
		case http.StatusTooManyRequests:
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeRateLimit, apiErr.Message, err)
		case http.StatusBadRequest:
			if apiErr.Code == "context_length_exceeded" {
				return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeContextLengthExceeded, apiErr.Message, err)
			}
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeInvalidRequest, apiErr.Message, err)
		case http.StatusNotFound:
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeModelNotFound, apiErr.Message, err)
		case http.StatusServiceUnavailable, http.StatusBadGateway:
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeServiceUnavailable, apiErr.Message, err)
		default:
			return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeUnknown, apiErr.Message, err)
		}
	}

	// Check for context errors
	if err == context.DeadlineExceeded || err == context.Canceled {
		return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeTimeout, "request timeout", err)
	}

	// Default unknown error
	return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeUnknown, err.Error(), err)
}

// validateRequest validates the request
func (c *Client) validateRequest(req *llm.ChatRequest) error {
	if len(req.Messages) == 0 {
		return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeInvalidRequest, "messages cannot be empty", nil)
	}

	if req.MaxTokens > 16384 {
		return llm.NewError(llm.ProviderOpenAI, llm.ErrorTypeInvalidRequest,
			fmt.Sprintf("max_tokens %d exceeds limit of 16384", req.MaxTokens), nil)
	}

	return nil
}

// buildCapabilities returns the capabilities of OpenAI
func buildCapabilities() *llm.Capabilities {
	return &llm.Capabilities{
		Provider:                llm.ProviderOpenAI,
		SupportsStreaming:       true,
		SupportsSystemPrompt:    true,
		MaxTokensLimit:          16384,
		MaxContextWindow:        128000,
		SupportsFunctionCalling: true,
		SupportsVision:          true,
		Models: []llm.ModelInfo{
			{
				ID:                    openai.GPT4o,
				Name:                  "GPT-4o",
				Description:           "Most advanced, multimodal flagship model",
				ContextWindow:         128000,
				MaxOutputTokens:       16384,
				InputPricePerMillion:  2.5,
				OutputPricePerMillion: 10.0,
				SupportsVision:        true,
				SupportsFunctions:     true,
			},
			{
				ID:                    openai.GPT4oMini,
				Name:                  "GPT-4o Mini",
				Description:           "Affordable and intelligent small model for fast, lightweight tasks",
				ContextWindow:         128000,
				MaxOutputTokens:       16384,
				InputPricePerMillion:  0.15,
				OutputPricePerMillion: 0.6,
				SupportsVision:        true,
				SupportsFunctions:     true,
			},
			{
				ID:                    openai.GPT4Turbo,
				Name:                  "GPT-4 Turbo",
				Description:           "Previous generation high-intelligence model",
				ContextWindow:         128000,
				MaxOutputTokens:       4096,
				InputPricePerMillion:  10.0,
				OutputPricePerMillion: 30.0,
				SupportsVision:        true,
				SupportsFunctions:     true,
			},
			{
				ID:                    openai.O1,
				Name:                  "o1",
				Description:           "Reasoning model designed to solve hard problems",
				ContextWindow:         200000,
				MaxOutputTokens:       100000,
				InputPricePerMillion:  15.0,
				OutputPricePerMillion: 60.0,
				SupportsVision:        false,
				SupportsFunctions:     false,
			},
			{
				ID:                    openai.O1Mini,
				Name:                  "o1-mini",
				Description:           "Faster and cheaper reasoning model",
				ContextWindow:         128000,
				MaxOutputTokens:       65536,
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 12.0,
				SupportsVision:        false,
				SupportsFunctions:     false,
			},
		},
	}
}
