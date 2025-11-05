package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateManager_Register(t *testing.T) {
	tm := NewTemplateManager()

	err := tm.Register("test", "Hello {{.Name}}")
	require.NoError(t, err)
	assert.True(t, tm.Exists("test"))
}

func TestTemplateManager_RegisterInvalid(t *testing.T) {
	tm := NewTemplateManager()

	err := tm.Register("invalid", "Hello {{.Name")
	assert.Error(t, err)
}

func TestTemplateManager_Execute(t *testing.T) {
	tm := NewTemplateManager()
	err := tm.Register("greeting", "Hello {{.Name}}!")
	require.NoError(t, err)

	result, err := tm.Execute("greeting", map[string]string{"Name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestTemplateManager_ExecuteNotFound(t *testing.T) {
	tm := NewTemplateManager()

	_, err := tm.Execute("nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTemplateManager_List(t *testing.T) {
	tm := NewTemplateManager()
	tm.Register("t1", "{{.A}}")
	tm.Register("t2", "{{.B}}")

	names := tm.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "t1")
	assert.Contains(t, names, "t2")
}

func TestTemplateManager_Delete(t *testing.T) {
	tm := NewTemplateManager()
	tm.Register("test", "{{.A}}")

	assert.True(t, tm.Exists("test"))
	tm.Delete("test")
	assert.False(t, tm.Exists("test"))
}

func TestPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder()

	req := pb.
		SetSystemPrompt("You are a helpful assistant").
		AddUserMessage("Hello").
		AddAssistantMessage("Hi there!").
		AddUserMessage("How are you?").
		BuildWithOptions(
			WithModel("test-model"),
			WithMaxTokens(100),
			WithTemperature(0.7),
		)

	assert.Equal(t, "You are a helpful assistant", req.SystemPrompt)
	assert.Len(t, req.Messages, 3)
	assert.Equal(t, RoleUser, req.Messages[0].Role)
	assert.Equal(t, "Hello", req.Messages[0].Content)
	assert.Equal(t, RoleAssistant, req.Messages[1].Role)
	assert.Equal(t, "Hi there!", req.Messages[1].Content)
	assert.Equal(t, "test-model", req.Model)
	assert.Equal(t, 100, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
}

func TestPromptBuilder_AppendSystemPrompt(t *testing.T) {
	pb := NewPromptBuilder()

	req := pb.
		SetSystemPrompt("Part 1").
		AppendSystemPrompt("Part 2").
		Build()

	assert.Equal(t, "Part 1\n\nPart 2", req.SystemPrompt)
}

func TestGetDefaultTemplates(t *testing.T) {
	tm, err := GetDefaultTemplates()
	require.NoError(t, err)

	// Check that default templates exist
	assert.True(t, tm.Exists("workflow_interpretation"))
	assert.True(t, tm.Exists("code_generation"))
	assert.True(t, tm.Exists("data_analysis"))
	assert.True(t, tm.Exists("summarization"))
}

func TestRequestOptions(t *testing.T) {
	req := &ChatRequest{}

	WithModel("test-model")(req)
	assert.Equal(t, "test-model", req.Model)

	WithMaxTokens(500)(req)
	assert.Equal(t, 500, req.MaxTokens)

	WithTemperature(0.8)(req)
	assert.Equal(t, 0.8, req.Temperature)

	WithTopP(0.9)(req)
	assert.Equal(t, 0.9, req.TopP)

	WithStopSequences("stop1", "stop2")(req)
	assert.Equal(t, []string{"stop1", "stop2"}, req.StopSequences)

	metadata := map[string]string{"key": "value"}
	WithMetadata(metadata)(req)
	assert.Equal(t, metadata, req.Metadata)
}
