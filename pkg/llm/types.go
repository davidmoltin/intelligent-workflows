package llm

import (
	"context"
	"time"
)

// Provider represents the LLM provider type
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
)

// Client defines the interface for LLM providers
type Client interface {
	// Chat sends a chat completion request and returns the response
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a chat completion request and streams the response
	StreamChat(ctx context.Context, req *ChatRequest, handler StreamHandler) error

	// GetCapabilities returns the capabilities of the LLM provider
	GetCapabilities() *Capabilities

	// GetProvider returns the provider type
	GetProvider() Provider

	// Close closes the client and releases resources
	Close() error
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	// Messages contains the conversation history
	Messages []Message `json:"messages"`

	// Model specifies which model to use (e.g., "claude-3-5-sonnet-20241022", "gpt-4")
	Model string `json:"model,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 to 1.0)
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP float64 `json:"top_p,omitempty"`

	// StopSequences are sequences that will stop generation
	StopSequences []string `json:"stop_sequences,omitempty"`

	// SystemPrompt is the system message (for providers that support it)
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Metadata for tracking and debugging
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// ChatResponse represents a chat completion response
type ChatResponse struct {
	// ID is the unique identifier for this completion
	ID string `json:"id"`

	// Content is the generated text
	Content string `json:"content"`

	// Model is the model that was used
	Model string `json:"model"`

	// Provider is the LLM provider
	Provider Provider `json:"provider"`

	// Usage contains token usage information
	Usage *TokenUsage `json:"usage"`

	// FinishReason indicates why generation stopped
	FinishReason string `json:"finish_reason"`

	// CreatedAt is when the completion was created
	CreatedAt time.Time `json:"created_at"`

	// Metadata contains additional provider-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used
	TotalTokens int `json:"total_tokens"`

	// CacheCreationInputTokens (Anthropic-specific)
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`

	// CacheReadInputTokens (Anthropic-specific)
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
}

// StreamHandler is called for each chunk in a streaming response
type StreamHandler func(chunk *StreamChunk) error

// StreamChunk represents a chunk of a streaming response
type StreamChunk struct {
	// Delta is the incremental content
	Delta string `json:"delta"`

	// IsComplete indicates if this is the final chunk
	IsComplete bool `json:"is_complete"`

	// Usage is only present in the final chunk
	Usage *TokenUsage `json:"usage,omitempty"`

	// FinishReason is only present in the final chunk
	FinishReason string `json:"finish_reason,omitempty"`
}

// Capabilities describes what a provider supports
type Capabilities struct {
	// Provider is the provider type
	Provider Provider `json:"provider"`

	// Models lists available models
	Models []ModelInfo `json:"models"`

	// SupportsStreaming indicates if streaming is supported
	SupportsStreaming bool `json:"supports_streaming"`

	// SupportsSystemPrompt indicates if system prompts are supported
	SupportsSystemPrompt bool `json:"supports_system_prompt"`

	// MaxTokensLimit is the maximum tokens that can be generated
	MaxTokensLimit int `json:"max_tokens_limit"`

	// MaxContextWindow is the maximum context window size
	MaxContextWindow int `json:"max_context_window"`

	// SupportsFunctionCalling indicates if function calling is supported
	SupportsFunctionCalling bool `json:"supports_function_calling"`

	// SupportsVision indicates if vision/image inputs are supported
	SupportsVision bool `json:"supports_vision"`
}

// ModelInfo describes a specific model
type ModelInfo struct {
	// ID is the model identifier
	ID string `json:"id"`

	// Name is a human-readable name
	Name string `json:"name"`

	// Description describes the model
	Description string `json:"description"`

	// ContextWindow is the maximum context size
	ContextWindow int `json:"context_window"`

	// MaxOutputTokens is the maximum output size
	MaxOutputTokens int `json:"max_output_tokens"`

	// InputPricePerMillion is the cost per million input tokens (in USD)
	InputPricePerMillion float64 `json:"input_price_per_million"`

	// OutputPricePerMillion is the cost per million output tokens (in USD)
	OutputPricePerMillion float64 `json:"output_price_per_million"`

	// SupportsVision indicates if this model supports image inputs
	SupportsVision bool `json:"supports_vision"`

	// SupportsFunctions indicates if this model supports function calling
	SupportsFunctions bool `json:"supports_functions"`
}

// Config represents LLM client configuration
type Config struct {
	// Provider specifies which LLM provider to use
	Provider Provider `json:"provider"`

	// APIKey is the authentication key
	APIKey string `json:"api_key"`

	// BaseURL is the API base URL (optional, for custom endpoints)
	BaseURL string `json:"base_url,omitempty"`

	// DefaultModel is the model to use if not specified in requests
	DefaultModel string `json:"default_model,omitempty"`

	// Timeout for API requests
	Timeout time.Duration `json:"timeout"`

	// MaxRetries for failed requests
	MaxRetries int `json:"max_retries"`

	// RetryDelay between retries
	RetryDelay time.Duration `json:"retry_delay"`
}
