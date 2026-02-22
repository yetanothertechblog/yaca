package tools

import (
	"context"
	"fmt"

	"go-tui/llm"
)

var registry = map[string]ToolImpl{}
var registryOrder []string

func Register(t ToolImpl) {
	name := t.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("duplicate tool registration: %s", name))
	}
	registry[name] = t
	registryOrder = append(registryOrder, name)
}

func All() []llm.Tool {
	out := make([]llm.Tool, 0, len(registryOrder))
	for _, name := range registryOrder {
		out = append(out, ToLLMTool(registry[name]))
	}
	return out
}

func Execute(name string, argsJSON string, workingDir string) (ToolResult, error) {
	t, ok := registry[name]
	if !ok {
		return ToolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(argsJSON, workingDir)
}

func ExecuteContext(ctx context.Context, name string, argsJSON string, workingDir string) (ToolResult, error) {
	t, ok := registry[name]
	if !ok {
		return ToolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
	return t.ExecuteContext(ctx, argsJSON, workingDir)
}
