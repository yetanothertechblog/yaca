package llm

import (
	"encoding/json"
	"testing"
)

func TestMessage_OmitsEmptyFields(t *testing.T) {
	msg := Message{Role: "user", Content: "test"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, ok := raw["tool_calls"]; ok {
		t.Error("empty tool_calls should be omitted")
	}
	if _, ok := raw["tool_call_id"]; ok {
		t.Error("empty tool_call_id should be omitted")
	}
}

func TestMessage_ToolCallsRoundTrip(t *testing.T) {
	msg := Message{
		Role: "assistant",
		ToolCalls: []ToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "bash",
				Arguments: `{"command":"ls"}`,
			},
		}},
	}
	data, _ := json.Marshal(msg)

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.ToolCalls) != 1 || decoded.ToolCalls[0].Function.Arguments != `{"command":"ls"}` {
		t.Errorf("tool call lost in round-trip: %+v", decoded.ToolCalls)
	}
}

func TestChatRequest_OmitsEmptyTools(t *testing.T) {
	req := ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}}
	data, _ := json.Marshal(req)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, ok := raw["tools"]; ok {
		t.Error("empty tools should be omitted from JSON")
	}
}

func TestChatRequest_IncludesTools(t *testing.T) {
	req := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools:    []Tool{{Type: "function", Function: ToolFunction{Name: "bash", Parameters: json.RawMessage(`{}`)}}},
		Stream:   true,
	}
	data, _ := json.Marshal(req)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	tools, ok := raw["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Errorf("expected 1 tool in JSON, got %v", raw["tools"])
	}
	if raw["stream"] != true {
		t.Error("stream should be true")
	}
}

func TestDelta_ToolCallsRoundTrip(t *testing.T) {
	delta := Delta{
		Role:             "assistant",
		Content:          "hello",
		ReasoningContent: "thinking",
		ToolCalls: []ToolCall{{
			ID: "c1", Type: "function",
			Function: ToolCallFunction{Name: "test", Arguments: `{"a":1}`},
		}},
	}
	data, _ := json.Marshal(delta)

	var decoded Delta
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.ReasoningContent != "thinking" {
		t.Errorf("ReasoningContent = %q", decoded.ReasoningContent)
	}
	if len(decoded.ToolCalls) != 1 || decoded.ToolCalls[0].Function.Arguments != `{"a":1}` {
		t.Errorf("ToolCalls lost: %+v", decoded.ToolCalls)
	}
}

func TestTool_ParametersPreservedAsRawJSON(t *testing.T) {
	schema := `{"type":"object","properties":{"cmd":{"type":"string"}}}`
	tool := Tool{
		Type:     "function",
		Function: ToolFunction{Name: "bash", Description: "run cmd", Parameters: json.RawMessage(schema)},
	}
	data, _ := json.Marshal(tool)

	var decoded Tool
	json.Unmarshal(data, &decoded)

	if string(decoded.Function.Parameters) != schema {
		t.Errorf("Parameters mutated: got %s", decoded.Function.Parameters)
	}
}
