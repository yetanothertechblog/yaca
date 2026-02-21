package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"go-tui/agent/tools"
	"go-tui/llm"
	"go-tui/lsp"
)

const systemPromptTemplate = `You are an expert coding assistant with integrated LSP support. You help users write, debug, and improve code.

Working directory: %s

Rules:
- ALWAYS explain code changes before making them. DO NOT JUST EDIT CODE
- Always break down tasks into smaller, manageable subtasks
- Give concise, direct answers. Avoid unnecessary preamble.
- When showing code, include only the relevant parts unless the user asks for the full file.
- If a question is ambiguous, ask a brief clarifying question before answering.
- When fixing bugs, explain the root cause in one sentence, then show the fix.
- Prefer simple, readable solutions over clever ones.
- If you don't know something, say so. Don't guess.
- Use the tools available to you when needed.
- When reading files, use paths relative to the working directory unless an absolute path is given.
- The system automatically runs LSP diagnostics after editing files to catch errors.
- Consider LSP feedback when making code changes and fix any reported issues.

%s`

type Agent struct {
	workingDir string
}

func New(workingDir string) *Agent {
	lsp.Start(workingDir)
	return &Agent{
		workingDir: workingDir,
	}
}

func (a *Agent) Shutdown() {
	lsp.Stop()
}

func (a *Agent) WorkingDir() string {
	return a.workingDir
}

func (a *Agent) SystemPrompt() string {
	yacaContent := a.readYacaMarkdown()
	return fmt.Sprintf(systemPromptTemplate, a.workingDir, yacaContent)
}

// readYacaMarkdown reads the YACA.md file and returns it wrapped in project_info tags
func (a *Agent) readYacaMarkdown() string {
	yacaPath := filepath.Join(a.workingDir, "YACA.md")
	if data, err := os.ReadFile(yacaPath); err == nil {
		return "\n<project_info>\n" + string(data) + "\n</project_info>"
	}
	return ""
}

func (a *Agent) ExecuteTool(name, argsJSON string) (tools.ToolResult, error) {
	return tools.Execute(name, argsJSON, a.workingDir)
}

func (a *Agent) Tools() []llm.Tool {
	return tools.All()
}
