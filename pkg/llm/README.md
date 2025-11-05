# LLM Client Abstraction

A unified interface for working with multiple LLM providers (Anthropic Claude and OpenAI).

## Features

- ✅ **Multi-provider support**: Anthropic Claude and OpenAI
- ✅ **Unified interface**: Write once, use with any provider
- ✅ **Streaming support**: Real-time response streaming
- ✅ **Token tracking**: Built-in token usage tracking
- ✅ **Error handling**: Comprehensive error types with retry logic
- ✅ **Prompt templates**: Reusable prompt templates with variable substitution
- ✅ **Type-safe**: Strong typing throughout

## Quick Start

### Creating a Client

```go
import (
    "github.com/davidmoltin/intelligent-workflows/pkg/llm"
    "github.com/davidmoltin/intelligent-workflows/pkg/llm/providers/anthropic"
    "github.com/davidmoltin/intelligent-workflows/pkg/llm/providers/openai"
)

// Anthropic Claude
config := &llm.Config{
    Provider:   llm.ProviderAnthropic,
    APIKey:     "your-api-key",
    DefaultModel: "claude-3-5-sonnet-20241022",
    Timeout:    60 * time.Second,
    MaxRetries: 3,
    RetryDelay: time.Second,
}
client, err := anthropic.NewClient(config)

// OpenAI
config := &llm.Config{
    Provider:   llm.ProviderOpenAI,
    APIKey:     "your-api-key",
    DefaultModel: "gpt-4o",
    Timeout:    60 * time.Second,
    MaxRetries: 3,
    RetryDelay: time.Second,
}
client, err := openai.NewClient(config)
```

### Basic Chat

```go
req := &llm.ChatRequest{
    Messages: []llm.Message{
        {Role: llm.RoleUser, Content: "Hello!"},
    },
    MaxTokens:   1000,
    Temperature: 0.7,
}

resp, err := client.Chat(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Content)
fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)
```

### Streaming Chat

```go
handler := func(chunk *llm.StreamChunk) error {
    if !chunk.IsComplete {
        fmt.Print(chunk.Delta)
    } else {
        fmt.Printf("\nTotal tokens: %d\n", chunk.Usage.TotalTokens)
    }
    return nil
}

err := client.StreamChat(ctx, req, handler)
```

### Using Prompt Templates

```go
// Get default templates
tm, _ := llm.GetDefaultTemplates()

// Execute a template
prompt, _ := tm.Execute("code_generation", map[string]interface{}{
    "Language":    "Go",
    "Requirement": "Create a HTTP server",
})

// Or use PromptBuilder
req := llm.NewPromptBuilder().
    SetSystemPrompt("You are a helpful assistant").
    AddUserMessage("What is Go?").
    BuildWithOptions(
        llm.WithModel("claude-3-5-sonnet-20241022"),
        llm.WithMaxTokens(2000),
        llm.WithTemperature(0.7),
    )
```

### Error Handling

```go
resp, err := client.Chat(ctx, req)
if err != nil {
    if llm.IsRetryable(err) {
        // Retry the request
    }

    if errors.Is(err, llm.ErrRateLimitExceeded) {
        // Handle rate limit
    }
}
```

### Getting Capabilities

```go
caps := client.GetCapabilities()
fmt.Printf("Provider: %s\n", caps.Provider)
fmt.Printf("Supports streaming: %v\n", caps.SupportsStreaming)
fmt.Printf("Max tokens: %d\n", caps.MaxTokensLimit)

for _, model := range caps.Models {
    fmt.Printf("- %s: $%.2f/M input, $%.2f/M output\n",
        model.Name,
        model.InputPricePerMillion,
        model.OutputPricePerMillion,
    )
}
```

## Configuration

Environment variables for LLM configuration:

```bash
LLM_PROVIDER=anthropic        # or "openai"
LLM_API_KEY=your-api-key
LLM_DEFAULT_MODEL=            # Optional, provider-specific default
LLM_TIMEOUT=60s
LLM_MAX_RETRIES=3
LLM_RETRY_DELAY=1s
LLM_BASE_URL=                 # Optional, for custom endpoints
```

## API Endpoints

When integrated into the Intelligent Workflows API:

### POST /api/v1/ai/chat
Send a chat completion request.

**Request:**
```json
{
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "model": "claude-3-5-sonnet-20241022",
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": false
}
```

**Response:**
```json
{
  "id": "msg_123",
  "content": "Hello! How can I help you?",
  "model": "claude-3-5-sonnet-20241022",
  "provider": "anthropic",
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  },
  "finish_reason": "end_turn",
  "created_at": "2025-11-05T10:00:00Z"
}
```

### GET /api/v1/ai/capabilities
Get provider capabilities.

**Response:**
```json
{
  "provider": "anthropic",
  "models": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "name": "Claude 3.5 Sonnet",
      "context_window": 200000,
      "max_output_tokens": 8192,
      "input_price_per_million": 3.0,
      "output_price_per_million": 15.0
    }
  ],
  "supports_streaming": true,
  "supports_system_prompt": true,
  "max_tokens_limit": 8192,
  "max_context_window": 200000
}
```

### POST /api/v1/ai/interpret
Interpret natural language into a workflow definition.

**Request:**
```json
{
  "description": "Create a workflow that sends an email when a user signs up"
}
```

**Response:**
```json
{
  "workflow": "{\"name\": \"User Signup Email\", ...}"
}
```

## Supported Models

### Anthropic
- Claude 3.5 Sonnet (claude-3-5-sonnet-20241022) - Most intelligent
- Claude 3.5 Haiku (claude-3-5-haiku-20241022) - Fastest
- Claude 3 Opus (claude-3-opus-20240229) - Most powerful

### OpenAI
- GPT-4o (gpt-4o) - Multimodal flagship
- GPT-4o Mini (gpt-4o-mini) - Fast and affordable
- GPT-4 Turbo (gpt-4-turbo) - Previous generation
- o1 (o1) - Reasoning model
- o1-mini (o1-mini) - Faster reasoning

## Testing

Run tests:
```bash
go test ./pkg/llm/...
```

## Architecture

```
pkg/llm/
├── types.go              # Core types and interfaces
├── errors.go             # Error types and handling
├── templates.go          # Prompt template system
├── providers/
│   ├── anthropic/
│   │   └── client.go     # Anthropic implementation
│   └── openai/
│       └── client.go     # OpenAI implementation
└── README.md             # This file
```

## License

Part of the Intelligent Workflows project.
