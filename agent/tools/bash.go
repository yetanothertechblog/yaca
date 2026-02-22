package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const bashTimeout = 30 * time.Second

type BashArgs struct {
	Command string `json:"command"`
}

func init() {
	Register(Typed[BashArgs]{
		ToolName:        "bash",
		ToolDescription: "Execute a bash command and return the output. Use this to run shell commands, read files, list directories, search code, etc.",
		ToolSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The bash command to execute"
				}
			},
			"required": ["command"]
		}`),
		Run:        executeBash,
		RunContext: executeBashContext,
	})
}

func executeBash(args BashArgs, workingDir string) (ToolResult, error) {
	return executeBashContext(context.Background(), args, workingDir)
}

func executeBashContext(ctx context.Context, args BashArgs, workingDir string) (ToolResult, error) {
	if args.Command == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "command is required")
	}

	// Create a context with timeout for the command execution
	cmdCtx, cancel := context.WithTimeout(ctx, bashTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", args.Command)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Check if context was cancelled
	if cmdCtx.Err() == context.DeadlineExceeded {
		return ToolResult{}, fmt.Errorf("command timed out after %v", bashTimeout)
	}
	if cmdCtx.Err() == context.Canceled {
		return ToolResult{}, ctx.Err()
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	if err != nil {
		if output != "" {
			output += "\n"
		}
		output += fmt.Sprintf("exit status: %v", err)
	}
	if output == "" {
		output = "(no output)"
	}
	return ToolResult{Output: output}, nil
}
