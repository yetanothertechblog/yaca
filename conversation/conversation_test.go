package conversation

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helpers to build RawMessage slices for test data
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func makeUIMessages(msgs ...map[string]any) json.RawMessage {
	return mustMarshal(msgs)
}

func makeHistory(msgs ...map[string]any) json.RawMessage {
	return mustMarshal(msgs)
}

// TestSaveCreatesJsonlFile checks that Save writes a .jsonl file with the right name.
func TestSaveCreatesJsonlFile(t *testing.T) {
	dir := t.TempDir()
	d := New()

	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	expected := filepath.Join(dir, d.ID+".jsonl")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected file %s not found: %v", expected, err)
	}
}

// TestSaveDoesNotCreateJsonFile ensures we no longer write .json files.
func TestSaveDoesNotCreateJsonFile(t *testing.T) {
	dir := t.TempDir()
	d := New()

	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	jsonFile := filepath.Join(dir, d.ID+".json")
	if _, err := os.Stat(jsonFile); err == nil {
		t.Fatalf("unexpected .json file found: %s", jsonFile)
	}
}

// TestSaveFileIsValidJsonl checks that every line in the saved file is valid JSON.
func TestSaveFileIsValidJsonl(t *testing.T) {
	dir := t.TempDir()
	d := &Data{
		ID:           "test-id-123",
		UIMessages:   makeUIMessages(map[string]any{"role": "user", "content": "hello"}),
		AgentHistory: makeHistory(map[string]any{"role": "user", "content": "hello"}),
	}

	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f, err := os.Open(filepath.Join(dir, d.ID+".jsonl"))
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %q", lineNum, line)
		}
	}
	if lineNum == 0 {
		t.Error("file is empty")
	}
}

// TestSaveFileStructure verifies the header line and record lines have the expected shape.
func TestSaveFileStructure(t *testing.T) {
	dir := t.TempDir()
	uiMsg := map[string]any{"role": "user", "content": "hi"}
	histMsg := map[string]any{"role": "assistant", "content": "hello"}
	d := &Data{
		ID:           "struct-test-id",
		UIMessages:   makeUIMessages(uiMsg),
		AgentHistory: makeHistory(histMsg),
	}

	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f, err := os.Open(filepath.Join(dir, d.ID+".jsonl"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Line 1: header
	if !scanner.Scan() {
		t.Fatal("file has no lines")
	}
	var h header
	if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {
		t.Fatalf("header parse: %v", err)
	}
	if h.ID != d.ID {
		t.Errorf("header ID = %q, want %q", h.ID, d.ID)
	}

	// Remaining lines: records
	var uiCount, histCount int
	for scanner.Scan() {
		var r record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatalf("record parse: %v", err)
		}
		switch r.Kind {
		case "ui":
			uiCount++
		case "history":
			histCount++
		default:
			t.Errorf("unknown record kind %q", r.Kind)
		}
	}

	if uiCount != 1 {
		t.Errorf("ui records = %d, want 1", uiCount)
	}
	if histCount != 1 {
		t.Errorf("history records = %d, want 1", histCount)
	}
}

// TestRoundTrip saves a conversation and loads it back, verifying all data is preserved.
func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()

	uiMsgs := []map[string]any{
		{"role": "user", "content": "what is 2+2?"},
		{"role": "assistant", "content": "4"},
	}
	histMsgs := []map[string]any{
		{"role": "user", "content": "what is 2+2?"},
		{"role": "assistant", "content": "4"},
	}

	original := &Data{
		ID:           "roundtrip-id",
		UIMessages:   mustMarshal(uiMsgs),
		AgentHistory: mustMarshal(histMsgs),
	}

	if err := original.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(filepath.Join(dir, original.ID+".jsonl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, original.ID)
	}

	// Unmarshal both sides and compare
	var gotUI, wantUI []map[string]any
	if err := json.Unmarshal(loaded.UIMessages, &gotUI); err != nil {
		t.Fatalf("unmarshal loaded UIMessages: %v", err)
	}
	if err := json.Unmarshal(original.UIMessages, &wantUI); err != nil {
		t.Fatalf("unmarshal original UIMessages: %v", err)
	}
	if len(gotUI) != len(wantUI) {
		t.Errorf("UIMessages len = %d, want %d", len(gotUI), len(wantUI))
	}
	for i := range wantUI {
		if gotUI[i]["role"] != wantUI[i]["role"] || gotUI[i]["content"] != wantUI[i]["content"] {
			t.Errorf("UIMessages[%d] = %v, want %v", i, gotUI[i], wantUI[i])
		}
	}

	var gotHist, wantHist []map[string]any
	if err := json.Unmarshal(loaded.AgentHistory, &gotHist); err != nil {
		t.Fatalf("unmarshal loaded AgentHistory: %v", err)
	}
	if err := json.Unmarshal(original.AgentHistory, &wantHist); err != nil {
		t.Fatalf("unmarshal original AgentHistory: %v", err)
	}
	if len(gotHist) != len(wantHist) {
		t.Errorf("AgentHistory len = %d, want %d", len(gotHist), len(wantHist))
	}
	for i := range wantHist {
		if gotHist[i]["role"] != wantHist[i]["role"] || gotHist[i]["content"] != wantHist[i]["content"] {
			t.Errorf("AgentHistory[%d] = %v, want %v", i, gotHist[i], wantHist[i])
		}
	}
}

// TestRoundTripEmptyMessages verifies that empty message arrays survive a round-trip.
func TestRoundTripEmptyMessages(t *testing.T) {
	dir := t.TempDir()
	d := New()

	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(filepath.Join(dir, d.ID+".jsonl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var ui []any
	if err := json.Unmarshal(loaded.UIMessages, &ui); err != nil {
		t.Fatalf("unmarshal UIMessages: %v", err)
	}
	if len(ui) != 0 {
		t.Errorf("UIMessages = %v, want []", ui)
	}

	var hist []any
	if err := json.Unmarshal(loaded.AgentHistory, &hist); err != nil {
		t.Fatalf("unmarshal AgentHistory: %v", err)
	}
	if len(hist) != 0 {
		t.Errorf("AgentHistory = %v, want []", hist)
	}
}

// TestLatestInDir returns the lexicographically last .jsonl file.
func TestLatestInDir(t *testing.T) {
	dir := t.TempDir()

	names := []string{"aaa", "bbb", "ccc"}
	for _, name := range names {
		d := &Data{
			ID:           name,
			UIMessages:   []byte("[]"),
			AgentHistory: []byte("[]"),
		}
		if err := d.Save(dir); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
	}

	latest, err := LatestInDir(dir)
	if err != nil {
		t.Fatalf("LatestInDir: %v", err)
	}
	if latest != "ccc.jsonl" {
		t.Errorf("LatestInDir = %q, want %q", latest, "ccc.jsonl")
	}
}

// TestLatestInDirIgnoresNonJsonl ensures .json files are not returned.
func TestLatestInDirIgnoresNonJsonl(t *testing.T) {
	dir := t.TempDir()

	// Write a stale .json file with a name that would sort last
	if err := os.WriteFile(filepath.Join(dir, "zzz.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write .json: %v", err)
	}

	d := &Data{
		ID:           "aaa",
		UIMessages:   []byte("[]"),
		AgentHistory: []byte("[]"),
	}
	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	latest, err := LatestInDir(dir)
	if err != nil {
		t.Fatalf("LatestInDir: %v", err)
	}
	if latest != "aaa.jsonl" {
		t.Errorf("LatestInDir = %q, want aaa.jsonl", latest)
	}
}

// TestLatestInDirEmpty returns an error when no .jsonl files exist.
func TestLatestInDirEmpty(t *testing.T) {
	dir := t.TempDir()
	_, err := LatestInDir(dir)
	if err == nil {
		t.Error("expected error for empty dir, got nil")
	}
}

// TestDirContainsYacaConversations checks that Dir() returns a path under .yaca/conversations.
func TestDirContainsYacaConversations(t *testing.T) {
	d := Dir()
	if !strings.HasSuffix(d, filepath.Join(".yaca", "conversations")) {
		t.Errorf("Dir() = %q, want suffix %q", d, filepath.Join(".yaca", "conversations"))
	}
}

// TestSaveCreatesDir verifies Save creates the target directory if it doesn't exist.
func TestSaveCreatesDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "conversations")

	d := New()
	if err := d.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, d.ID+".jsonl")); err != nil {
		t.Fatalf("file not found after Save into non-existent dir: %v", err)
	}
}
