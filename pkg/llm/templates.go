package llm

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Template represents a prompt template
type Template struct {
	name     string
	content  string
	template *template.Template
}

// TemplateManager manages prompt templates
type TemplateManager struct {
	templates map[string]*Template
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*Template),
	}
}

// Register registers a new template
func (tm *TemplateManager) Register(name, content string) error {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	tm.templates[name] = &Template{
		name:     name,
		content:  content,
		template: tmpl,
	}

	return nil
}

// Execute executes a template with the given data
func (tm *TemplateManager) Execute(name string, data interface{}) (string, error) {
	tmpl, ok := tm.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// Exists checks if a template exists
func (tm *TemplateManager) Exists(name string) bool {
	_, ok := tm.templates[name]
	return ok
}

// List returns all template names
func (tm *TemplateManager) List() []string {
	names := make([]string, 0, len(tm.templates))
	for name := range tm.templates {
		names = append(names, name)
	}
	return names
}

// Delete removes a template
func (tm *TemplateManager) Delete(name string) {
	delete(tm.templates, name)
}

// PromptBuilder helps build complex prompts
type PromptBuilder struct {
	systemPrompt strings.Builder
	messages     []Message
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		messages: make([]Message, 0),
	}
}

// SetSystemPrompt sets the system prompt
func (pb *PromptBuilder) SetSystemPrompt(prompt string) *PromptBuilder {
	pb.systemPrompt.Reset()
	pb.systemPrompt.WriteString(prompt)
	return pb
}

// AppendSystemPrompt appends to the system prompt
func (pb *PromptBuilder) AppendSystemPrompt(prompt string) *PromptBuilder {
	if pb.systemPrompt.Len() > 0 {
		pb.systemPrompt.WriteString("\n\n")
	}
	pb.systemPrompt.WriteString(prompt)
	return pb
}

// AddUserMessage adds a user message
func (pb *PromptBuilder) AddUserMessage(content string) *PromptBuilder {
	pb.messages = append(pb.messages, Message{
		Role:    RoleUser,
		Content: content,
	})
	return pb
}

// AddAssistantMessage adds an assistant message
func (pb *PromptBuilder) AddAssistantMessage(content string) *PromptBuilder {
	pb.messages = append(pb.messages, Message{
		Role:    RoleAssistant,
		Content: content,
	})
	return pb
}

// AddMessages adds multiple messages
func (pb *PromptBuilder) AddMessages(messages ...Message) *PromptBuilder {
	pb.messages = append(pb.messages, messages...)
	return pb
}

// Build builds a ChatRequest
func (pb *PromptBuilder) Build() *ChatRequest {
	return &ChatRequest{
		Messages:     pb.messages,
		SystemPrompt: pb.systemPrompt.String(),
	}
}

// BuildWithOptions builds a ChatRequest with additional options
func (pb *PromptBuilder) BuildWithOptions(opts ...RequestOption) *ChatRequest {
	req := pb.Build()
	for _, opt := range opts {
		opt(req)
	}
	return req
}

// RequestOption is a function that modifies a ChatRequest
type RequestOption func(*ChatRequest)

// WithModel sets the model
func WithModel(model string) RequestOption {
	return func(req *ChatRequest) {
		req.Model = model
	}
}

// WithMaxTokens sets the max tokens
func WithMaxTokens(maxTokens int) RequestOption {
	return func(req *ChatRequest) {
		req.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature
func WithTemperature(temperature float64) RequestOption {
	return func(req *ChatRequest) {
		req.Temperature = temperature
	}
}

// WithTopP sets the top_p
func WithTopP(topP float64) RequestOption {
	return func(req *ChatRequest) {
		req.TopP = topP
	}
}

// WithStopSequences sets the stop sequences
func WithStopSequences(stopSequences ...string) RequestOption {
	return func(req *ChatRequest) {
		req.StopSequences = stopSequences
	}
}

// WithMetadata sets metadata
func WithMetadata(metadata map[string]string) RequestOption {
	return func(req *ChatRequest) {
		req.Metadata = metadata
	}
}

// Common prompt templates

const (
	// WorkflowInterpretationTemplate helps interpret natural language into workflows
	WorkflowInterpretationTemplate = `You are an AI assistant that helps users create workflow definitions.

Given a natural language description, generate a valid workflow definition in JSON format.

User's request: {{.Request}}

Return only valid JSON that matches the workflow schema.`

	// CodeGenerationTemplate helps generate code
	CodeGenerationTemplate = `You are an expert software engineer.

Generate {{.Language}} code for the following requirement:

{{.Requirement}}

{{if .Context}}
Context:
{{.Context}}
{{end}}

Provide clean, well-documented code following best practices.`

	// DataAnalysisTemplate helps analyze data
	DataAnalysisTemplate = `You are a data analyst assistant.

Analyze the following data and provide insights:

{{.Data}}

Focus on:
{{range .FocusAreas}}
- {{.}}
{{end}}

Provide clear, actionable insights.`

	// SummarizationTemplate helps summarize content
	SummarizationTemplate = `Summarize the following content in {{.MaxWords}} words or less:

{{.Content}}

Focus on the key points and main ideas.`
)

// GetDefaultTemplates returns a template manager with common templates pre-registered
func GetDefaultTemplates() (*TemplateManager, error) {
	tm := NewTemplateManager()

	templates := map[string]string{
		"workflow_interpretation": WorkflowInterpretationTemplate,
		"code_generation":         CodeGenerationTemplate,
		"data_analysis":           DataAnalysisTemplate,
		"summarization":           SummarizationTemplate,
	}

	for name, content := range templates {
		if err := tm.Register(name, content); err != nil {
			return nil, fmt.Errorf("failed to register default template %s: %w", name, err)
		}
	}

	return tm, nil
}
