package tools

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTypedAdapter_Execute(t *testing.T) {
	type Args struct {
		Message string `json:"message"`
	}

	tool := Typed[Args]{
		ToolName:        "test_tool",
		ToolDescription: "A test tool",
		ToolSchema:      json.RawMessage(`{"type":"object"}`),
		Run: func(args Args, workingDir string) (ToolResult, error) {
			return ToolResult{Output: fmt.Sprintf("%s from %s", args.Message, workingDir)}, nil
		},
	}

	result, err := tool.Execute(`{"message":"hello"}`, "/work")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Output != "hello from /work" {
		t.Errorf("Execute() = %q, want %q", result.Output, "hello from /work")
	}
}

func TestTypedAdapter_InvalidJSON(t *testing.T) {
	type Args struct{ X int `json:"x"` }
	tool := Typed[Args]{
		ToolName: "t", ToolDescription: "t", ToolSchema: json.RawMessage(`{}`),
		Run: func(args Args, workingDir string) (ToolResult, error) {
			t.Fatal("Run should not be called with invalid JSON")
			return ToolResult{}, nil
		},
	}

	_, err := tool.Execute("{bad json}", "")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTypedAdapter_RunErrorPropagates(t *testing.T) {
	type Args struct{}
	tool := Typed[Args]{
		ToolName: "t", ToolDescription: "t", ToolSchema: json.RawMessage(`{}`),
		Run: func(args Args, workingDir string) (ToolResult, error) {
			return ToolResult{}, fmt.Errorf("deliberate failure")
		},
	}

	_, err := tool.Execute(`{}`, "")
	if err == nil || err.Error() != "deliberate failure" {
		t.Errorf("expected 'deliberate failure', got %v", err)
	}
}

func TestToLLMTool(t *testing.T) {
	type Args struct{}
	tool := Typed[Args]{
		ToolName:        "my_tool",
		ToolDescription: "does things",
		ToolSchema:      json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}}}`),
		Run:             func(args Args, workingDir string) (ToolResult, error) { return ToolResult{}, nil },
	}

	llmTool := ToLLMTool(tool)
	if llmTool.Type != "function" {
		t.Errorf("Type = %q, want 'function'", llmTool.Type)
	}
	if llmTool.Function.Name != "my_tool" {
		t.Errorf("Name = %q", llmTool.Function.Name)
	}
	if llmTool.Function.Description != "does things" {
		t.Errorf("Description = %q", llmTool.Function.Description)
	}
	// Verify Parameters is valid JSON that round-trips correctly
	var params map[string]interface{}
	if err := json.Unmarshal(llmTool.Function.Parameters, &params); err != nil {
		t.Fatalf("Parameters is not valid JSON: %v", err)
	}
	if _, ok := params["properties"]; !ok {
		t.Error("Parameters should contain 'properties' key")
	}
}
