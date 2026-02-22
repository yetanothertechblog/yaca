package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemPromptWithYacaMd(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a test YACA.md file
	yacaContent := `# Test Project Configuration

## Development Guidelines
- Follow Go best practices
- Write clean, maintainable code
- Add comprehensive tests

## Project Structure
- cmd/ - Application entry points
- internal/ - Private application code
- pkg/ - Public library code`
	
	yacaPath := filepath.Join(tempDir, "YACA.md")
	err := os.WriteFile(yacaPath, []byte(yacaContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create YACA.md file: %v", err)
	}
	
	// Change to the temp directory for this test
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	a := New(tempDir)
	prompt := a.SystemPrompt()

	// Verify YACA.md content is present and properly tagged
	if !strings.Contains(prompt, "<project_info>") {
		t.Error("System prompt should contain <project_info> tag when YACA.md exists")
	}

	if !strings.Contains(prompt, "</project_info>") {
		t.Error("System prompt should contain </project_info> tag when YACA.md exists")
	}

	if !strings.Contains(prompt, "Test Project Configuration") {
		t.Error("System prompt should contain YACA.md title")
	}

	if !strings.Contains(prompt, "Follow Go best practices") {
		t.Error("System prompt should contain YACA.md content")
	}

	if !strings.Contains(prompt, "cmd/ - Application entry points") {
		t.Error("System prompt should contain YACA.md project structure")
	}

	// Verify order: working directory -> rules -> project info
	wdPos := strings.Index(prompt, "Working directory: "+tempDir)
	rulesPos := strings.Index(prompt, "Rules:")
	projectInfoPos := strings.Index(prompt, "<project_info>")

	if wdPos == -1 || rulesPos == -1 || projectInfoPos == -1 {
		t.Error("Could not find required sections in prompt")
	}

	if wdPos > rulesPos {
		t.Error("Working directory should come before rules")
	}

	if rulesPos > projectInfoPos {
		t.Error("Rules should come before project info")
	}
}

func TestSystemPromptWithoutYacaMd(t *testing.T) {
	// Create a temporary directory without YACA.md
	tempDir := t.TempDir()
	
	// Change to the temp directory for this test
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	a := New(tempDir)
	prompt := a.SystemPrompt()

	// Basic structure should still be present
	if !strings.Contains(prompt, "You are an expert coding assistant") {
		t.Error("System prompt should contain introduction even without YACA.md")
	}

	if !strings.Contains(prompt, "Working directory: "+tempDir) {
		t.Error("System prompt should contain working directory even without YACA.md")
	}

	// Rules should be present
	if !strings.Contains(prompt, "Rules:") {
		t.Error("Rules section should be present even without YACA.md")
	}

	// YACA.md tags should NOT be present when file doesn't exist
	if strings.Contains(prompt, "<project_info>") {
		t.Error("System prompt should not contain <project_info> when YACA.md doesn't exist")
	}

	if strings.Contains(prompt, "</project_info>") {
		t.Error("System prompt should not contain </project_info> when YACA.md doesn't exist")
	}
}

func TestSystemPromptWithEmptyYacaMd(t *testing.T) {
	// Create a temporary directory with empty YACA.md
	tempDir := t.TempDir()
	
	// Create an empty YACA.md file
	yacaPath := filepath.Join(tempDir, "YACA.md")
	err := os.WriteFile(yacaPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty YACA.md file: %v", err)
	}
	
	// Change to the temp directory for this test
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	a := New(tempDir)
	prompt := a.SystemPrompt()

	// Tags should still be present even for empty file
	if !strings.Contains(prompt, "<project_info>") {
		t.Error("System prompt should contain <project_info> tag even for empty YACA.md")
	}

	if !strings.Contains(prompt, "</project_info>") {
		t.Error("System prompt should contain </project_info> tag even for empty YACA.md")
	}

	// But no actual content from YACA.md should be present
	if strings.Contains(prompt, "Test Project Configuration") {
		t.Error("System prompt should not contain YACA.md content for empty file")
	}
}

// Test with virtual YACA.md content by creating a real file in temp directory
func TestSystemPromptWithVirtualYacaMd(t *testing.T) {
	// Create a virtual YACA.md file in temp directory
	tempDir := t.TempDir()
	
	// Create a virtual YACA.md file
	yacaContent := `# Virtual Project
Virtual content
More virtual content`
	
	yacaPath := filepath.Join(tempDir, "YACA.md")
	err := os.WriteFile(yacaPath, []byte(yacaContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create virtual YACA.md file: %v", err)
	}
	
	// Change to the temp directory for this test
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Create new agent in temp directory
	a := New(tempDir)
	prompt := a.SystemPrompt()

	// Verify virtual YACA.md content is present
	if !strings.Contains(prompt, "<project_info>") {
		t.Error("System prompt should contain <project_info> tag")
	}

	if !strings.Contains(prompt, "</project_info>") {
		t.Error("System prompt should contain </project_info> tag")
	}

	if !strings.Contains(prompt, "Virtual Project") {
		t.Error("System prompt should contain virtual YACA.md content")
	}

	if !strings.Contains(prompt, "Virtual content") {
		t.Error("System prompt should contain virtual YACA.md content")
	}
}