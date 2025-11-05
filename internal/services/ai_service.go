package services

import (
	"context"
	"fmt"

	"github.com/davidmoltin/intelligent-workflows/pkg/llm"
	"go.uber.org/zap"
)

// AIService handles AI-related operations
type AIService struct {
	llmClient        llm.Client
	templateManager  *llm.TemplateManager
	logger           *zap.Logger
}

// NewAIService creates a new AI service
func NewAIService(llmClient llm.Client, logger *zap.Logger) (*AIService, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("llm client cannot be nil")
	}

	templateManager, err := llm.GetDefaultTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return &AIService{
		llmClient:       llmClient,
		templateManager: templateManager,
		logger:          logger,
	}, nil
}

// Chat sends a chat completion request
func (s *AIService) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	s.logger.Info("processing chat request",
		zap.String("provider", string(s.llmClient.GetProvider())),
		zap.String("model", req.Model),
		zap.Int("message_count", len(req.Messages)),
	)

	resp, err := s.llmClient.Chat(ctx, req)
	if err != nil {
		s.logger.Error("chat request failed",
			zap.Error(err),
			zap.String("provider", string(s.llmClient.GetProvider())),
		)
		return nil, err
	}

	s.logger.Info("chat request completed",
		zap.String("id", resp.ID),
		zap.String("provider", string(resp.Provider)),
		zap.String("model", resp.Model),
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
		zap.Int("total_tokens", resp.Usage.TotalTokens),
	)

	return resp, nil
}

// StreamChat sends a streaming chat completion request
func (s *AIService) StreamChat(ctx context.Context, req *llm.ChatRequest, handler llm.StreamHandler) error {
	s.logger.Info("processing streaming chat request",
		zap.String("provider", string(s.llmClient.GetProvider())),
		zap.String("model", req.Model),
		zap.Int("message_count", len(req.Messages)),
	)

	err := s.llmClient.StreamChat(ctx, req, handler)
	if err != nil {
		s.logger.Error("streaming chat request failed",
			zap.Error(err),
			zap.String("provider", string(s.llmClient.GetProvider())),
		)
		return err
	}

	s.logger.Info("streaming chat request completed",
		zap.String("provider", string(s.llmClient.GetProvider())),
	)

	return nil
}

// GetCapabilities returns the capabilities of the LLM provider
func (s *AIService) GetCapabilities() *llm.Capabilities {
	return s.llmClient.GetCapabilities()
}

// InterpretWorkflow interprets natural language into a workflow definition
func (s *AIService) InterpretWorkflow(ctx context.Context, description string) (string, error) {
	// Use the workflow interpretation template
	prompt, err := s.templateManager.Execute("workflow_interpretation", map[string]interface{}{
		"Request": description,
	})
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	req := llm.NewPromptBuilder().
		SetSystemPrompt(prompt).
		AddUserMessage(description).
		BuildWithOptions(
			llm.WithMaxTokens(4096),
			llm.WithTemperature(0.7),
		)

	resp, err := s.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to interpret workflow: %w", err)
	}

	return resp.Content, nil
}

// RegisterTemplate registers a custom template
func (s *AIService) RegisterTemplate(name, content string) error {
	return s.templateManager.Register(name, content)
}

// ExecuteTemplate executes a template with the given data
func (s *AIService) ExecuteTemplate(name string, data interface{}) (string, error) {
	return s.templateManager.Execute(name, data)
}

// ListTemplates returns all available template names
func (s *AIService) ListTemplates() []string {
	return s.templateManager.List()
}

// Close closes the service and releases resources
func (s *AIService) Close() error {
	if s.llmClient != nil {
		return s.llmClient.Close()
	}
	return nil
}
