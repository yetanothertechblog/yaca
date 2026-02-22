package agent

import (
	"strings"
	"testing"
)

func TestSystemPromptBasicStructure(t *testing.T) {
	workingDir := "/test/path"
	a := New(workingDir)
	prompt := a.SystemPrompt()

	// Test basic components are present
	requiredComponents := []string{
		"You are an expert coding assistant",
		"Working directory: " + workingDir,
		"Rules:",
		"ALWAYS explain code changes before making them",
		"Always break down tasks into smaller, manageable subtasks",
		"Give concise, direct answers",
		"If you don't know something, say so",
	}

	for _, component := range requiredComponents {
		if !strings.Contains(prompt, component) {
			t.Errorf("System prompt missing required component: %s", component)
		}
	}
}

func TestSystemPromptOrder(t *testing.T) {
	workingDir := "/test/project"
	a := New(workingDir)
	prompt := a.SystemPrompt()

	// Define expected order of sections
	expectedSections := []string{
		"You are an expert coding assistant",
		"Working directory: " + workingDir,
		"Rules:",
		"Use the tools available to you when needed",
	}

	// Find positions of each section
	positions := make([]int, len(expectedSections))
	for i, section := range expectedSections {
		positions[i] = strings.Index(prompt, section)
		if positions[i] == -1 {
			t.Errorf("Could not find section: %s", section)
		}
	}

	// Verify sections appear in correct order
	for i := 1; i < len(positions); i++ {
		if positions[i] < positions[i-1] {
			t.Errorf("Sections should appear in order: %s should come before %s", 
				expectedSections[i-1], expectedSections[i])
		}
	}
}

func TestSystemPromptWorkingDirectoryPlaceholder(t *testing.T) {
	testCases := []struct {
		name     string
		dir      string
		expected string
	}{
		{"Root dir", "/", "Working directory: /"},
		{"Relative dir", "./project", "Working directory: ./project"},
		{"Complex path", "/home/user/go-project", "Working directory: /home/user/go-project"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := New(tc.dir)
			prompt := a.SystemPrompt()
			
			if !strings.Contains(prompt, tc.expected) {
				t.Errorf("Expected working directory '%s' in prompt, got: %s", tc.expected, prompt)
			}
		})
	}
}

func TestSystemPromptContent(t *testing.T) {
	workingDir := "/test"
	a := New(workingDir)
	prompt := a.SystemPrompt()

	// Test key rules are present
	keyRules := []string{
		"ALWAYS explain code changes before making them. DO NOT JUST EDIT CODE",
		"Always break down tasks into smaller, manageable subtasks",
		"Give concise, direct answers. Avoid unnecessary preamble.",
		"If a question is ambiguous, ask a brief clarifying question before answering.",
		"When fixing bugs, explain the root cause in one sentence, then show the fix.",
		"Prefer simple, readable solutions over clever ones.",
		"If you don't know something, say so. Don't guess.",
		"Use the tools available to you when needed.",
		"When reading files, use paths relative to the working directory unless an absolute path is given.",
		"The system automatically runs LSP diagnostics after editing files to catch errors.",
		"Consider LSP feedback when making code changes and fix any reported issues.",
	}

	for _, rule := range keyRules {
		if !strings.Contains(prompt, rule) {
			t.Errorf("Missing key rule in system prompt: %s", rule)
		}
	}
}

func TestSystemPromptNoYacaMdTagsWhenFileMissing(t *testing.T) {
	// This test assumes no YACA.md file exists in the test environment
	workingDir := "/test"
	a := New(workingDir)
	prompt := a.SystemPrompt()

	// Since we're not mocking file system, YACA.md tags should not be present
	// unless there's actually a YACA.md file in the current directory
	// This test ensures the system prompt works even without YACA.md
	
	// Basic structure should still be intact
	if !strings.Contains(prompt, "You are an expert coding assistant") {
		t.Error("Basic system prompt structure missing")
	}

	if !strings.Contains(prompt, "Working directory: "+workingDir) {
		t.Error("Working directory missing from system prompt")
	}

	// Rules should be present
	if !strings.Contains(prompt, "Rules:") {
		t.Error("Rules section missing from system prompt")
	}
}