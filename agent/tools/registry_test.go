package tools

import (
	"encoding/json"
	"fmt"
	"testing"
)

type mockTool struct {
	name        string
	description string
	schema      string
	executeFn   func(argsJSON string, workingDir string) (ToolResult, error)
}

func (m *mockTool) Name() string           { return m.name }
func (m *mockTool) Description() string    { return m.description }
func (m *mockTool) Schema() json.RawMessage { return json.RawMessage(m.schema) }
func (m *mockTool) Execute(argsJSON string, workingDir string) (ToolResult, error) {
	if m.executeFn != nil {
		return m.executeFn(argsJSON, workingDir)
	}
	return ToolResult{Output: "default"}, nil
}

// Helper function to clear registry for isolated tests
func clearRegistry() {
	registry = make(map[string]ToolImpl)
	registryOrder = make([]string, 0)
}

func TestRegister(t *testing.T) {
	clearRegistry()
	Register(&mockTool{name: "a", schema: `{}`})
	Register(&mockTool{name: "b", schema: `{}`})

	if len(registry) != 2 {
		t.Errorf("registry has %d tools, want 2", len(registry))
	}
}

func TestRegister_PanicsOnDuplicate(t *testing.T) {
	clearRegistry()
	Register(&mockTool{name: "dup", schema: `{}`})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register(&mockTool{name: "dup", schema: `{}`})
}

func TestAll_PreservesOrder(t *testing.T) {
	clearRegistry()
	for _, name := range []string{"z", "a", "m"} {
		Register(&mockTool{name: name, schema: `{}`})
	}
	tools := All()
	want := []string{"z", "a", "m"}
	for i, tool := range tools {
		if tool.Function.Name != want[i] {
			t.Errorf("All()[%d].Name = %q, want %q", i, tool.Function.Name, want[i])
		}
	}
}

func TestExecute_PassesArgsAndWorkingDir(t *testing.T) {
	clearRegistry()
	var gotArgs, gotDir string
	Register(&mockTool{
		name: "capture", schema: `{}`,
		executeFn: func(argsJSON string, workingDir string) (ToolResult, error) {
			gotArgs = argsJSON
			gotDir = workingDir
			return ToolResult{Output: "ok"}, nil
		},
	})

	result, err := Execute("capture", `{"key":"val"}`, "/my/dir")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if gotArgs != `{"key":"val"}` {
		t.Errorf("argsJSON = %q, want %q", gotArgs, `{"key":"val"}`)
	}
	if gotDir != "/my/dir" {
		t.Errorf("workingDir = %q, want %q", gotDir, "/my/dir")
	}
	if result.Output != "ok" {
		t.Errorf("Output = %q, want %q", result.Output, "ok")
	}
}

func TestExecute_PropagatesError(t *testing.T) {
	clearRegistry()
	Register(&mockTool{
		name: "failing", schema: `{}`,
		executeFn: func(argsJSON string, workingDir string) (ToolResult, error) {
			return ToolResult{}, fmt.Errorf("tool broke")
		},
	})

	_, err := Execute("failing", `{}`, "")
	if err == nil || err.Error() != "tool broke" {
		t.Errorf("expected 'tool broke', got %v", err)
	}
}

func TestExecute_UnknownTool(t *testing.T) {
	clearRegistry()
	_, err := Execute("nonexistent", `{}`, "")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}
