package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jsgv/mcp-domain-checker/internal/pkg/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockService implements tool.Service[In, Out] for testing.
type mockService struct {
	name        string
	description string
	executeFunc func(in mockInput) (mockOutput, error)
}

type mockInput struct {
	Value string `json:"value"`
}

type mockOutput struct {
	Result string `json:"result"`
}

func (m *mockService) Name() string {
	return m.name
}

func (m *mockService) Description() string {
	return m.description
}

func (m *mockService) Execute(in mockInput) (mockOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(in)
	}

	return mockOutput{
		Result: "processed: " + in.Value,
	}, nil
}

func TestNewTool(t *testing.T) {
	t.Parallel()

	service := &mockService{
		name:        "test-tool",
		description: "A test tool",
		executeFunc: nil,
	}

	testTool := tool.NewTool(service)

	if testTool == nil {
		t.Fatal("NewTool() returned nil")
	}
}

func TestToolNameAndDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serviceName     string
		serviceDesc     string
		wantName        string
		wantDescription string
	}{
		{
			name:            "simple name and description",
			serviceName:     "my-tool",
			serviceDesc:     "My tool description",
			wantName:        "my-tool",
			wantDescription: "My tool description",
		},
		{
			name:            "empty values",
			serviceName:     "",
			serviceDesc:     "",
			wantName:        "",
			wantDescription: "",
		},
		{
			name:            "special characters",
			serviceName:     "tool-with-dashes_and_underscores",
			serviceDesc:     "Description with special chars: @#$%",
			wantName:        "tool-with-dashes_and_underscores",
			wantDescription: "Description with special chars: @#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &mockService{
				name:        tt.serviceName,
				description: tt.serviceDesc,
				executeFunc: nil,
			}

			testTool := tool.NewTool(service)

			if got := testTool.Name(); got != tt.wantName {
				t.Errorf("Tool.Name() = %v, want %v", got, tt.wantName)
			}

			if got := testTool.Description(); got != tt.wantDescription {
				t.Errorf("Tool.Description() = %v, want %v", got, tt.wantDescription)
			}
		})
	}
}

//nolint:cyclop
func TestToolHandler_Success(t *testing.T) {
	t.Parallel()

	service := &mockService{
		name:        "test",
		description: "test",
		executeFunc: func(in mockInput) (mockOutput, error) {
			return mockOutput{Result: "success: " + in.Value}, nil
		},
	}

	testTool := tool.NewTool(service)

	result, output, err := testTool.Handler(context.Background(), nil, mockInput{Value: "test-input"})
	if err != nil {
		t.Fatalf("Handler() unexpected error: %v", err)
	}

	if output.Result != "success: test-input" {
		t.Errorf("Handler() output.Result = %v, want success: test-input", output.Result)
	}

	if result == nil {
		t.Fatal("Handler() returned nil result")
	}

	if result.IsError {
		t.Error("Handler() result.IsError = true, want false")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Handler() result.Content length = %d, want 1", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Handler() result.Content[0] is not TextContent")
	}

	// Verify JSON content
	var jsonOutput mockOutput

	err = json.Unmarshal([]byte(textContent.Text), &jsonOutput)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON content: %v", err)
	}

	if jsonOutput.Result != "success: test-input" {
		t.Errorf("JSON content result = %v, want success: test-input", jsonOutput.Result)
	}

	// Verify annotations
	if textContent.Annotations == nil {
		t.Fatal("Handler() annotations is nil")
	}

	if len(textContent.Annotations.Audience) != 1 || textContent.Annotations.Audience[0] != "assistant" {
		t.Errorf("Handler() audience = %v, want [assistant]", textContent.Annotations.Audience)
	}

	if textContent.Annotations.Priority != 1 {
		t.Errorf("Handler() priority = %v, want 1", textContent.Annotations.Priority)
	}
}

var errServiceError = errors.New("service error")

func TestToolHandler_ServiceError(t *testing.T) {
	t.Parallel()

	service := &mockService{
		name:        "test",
		description: "test",
		executeFunc: func(_ mockInput) (mockOutput, error) {
			return mockOutput{Result: ""}, errServiceError
		},
	}

	testTool := tool.NewTool(service)

	result, output, err := testTool.Handler(context.Background(), nil, mockInput{Value: "test"})

	if !errors.Is(err, errServiceError) {
		t.Errorf("Handler() error = %v, want %v", err, errServiceError)
	}

	if result != nil {
		t.Error("Handler() result should be nil on error")
	}

	if output.Result != "" {
		t.Errorf("Handler() output should be zero value, got %v", output)
	}
}

// unmarshalableOutput is a type that cannot be marshaled to JSON.
type unmarshalableOutput struct {
	Channel chan int `json:"channel"`
}

type unmarshalableService struct{}

func (u *unmarshalableService) Name() string {
	return "unmarshalable"
}

func (u *unmarshalableService) Description() string {
	return "unmarshalable"
}

func (u *unmarshalableService) Execute(_ mockInput) (unmarshalableOutput, error) {
	return unmarshalableOutput{Channel: make(chan int)}, nil
}

func TestToolHandler_JSONMarshalError(t *testing.T) {
	t.Parallel()

	service := &unmarshalableService{}
	testTool := tool.NewTool(service)

	result, _, err := testTool.Handler(context.Background(), nil, mockInput{Value: "test"})
	if err == nil {
		t.Error("Handler() expected error for unmarshalable output")
	}

	if result != nil {
		t.Error("Handler() result should be nil on JSON marshal error")
	}
}