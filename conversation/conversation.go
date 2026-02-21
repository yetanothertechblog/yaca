package conversation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gofrs/uuid/v5"
	"go-tui/config"
)

type Data struct {
	ID           string          `json:"id"`
	UIMessages   json.RawMessage `json:"ui_messages"`
	AgentHistory json.RawMessage `json:"agent_history"`
}

type header struct {
	ID string `json:"id"`
}

type record struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

func New() *Data {
	id, _ := uuid.NewV7()
	return &Data{
		ID:           id.String(),
		UIMessages:   []byte("[]"),
		AgentHistory: []byte("[]"),
	}
}

func Load(path string) (*Data, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading conversation file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	if !scanner.Scan() {
		return nil, fmt.Errorf("empty conversation file")
	}
	var h header
	if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {
		return nil, fmt.Errorf("parsing conversation header: %w", err)
	}

	var uiMessages []json.RawMessage
	var agentHistory []json.RawMessage

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var r record
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("parsing conversation record: %w", err)
		}
		switch r.Kind {
		case "ui":
			uiMessages = append(uiMessages, r.Data)
		case "history":
			agentHistory = append(agentHistory, r.Data)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading conversation file: %w", err)
	}

	uiJSON, err := json.Marshal(uiMessages)
	if err != nil {
		return nil, fmt.Errorf("marshaling ui messages: %w", err)
	}
	histJSON, err := json.Marshal(agentHistory)
	if err != nil {
		return nil, fmt.Errorf("marshaling agent history: %w", err)
	}

	if uiMessages == nil {
		uiJSON = []byte("[]")
	}
	if agentHistory == nil {
		histJSON = []byte("[]")
	}

	return &Data{
		ID:           h.ID,
		UIMessages:   uiJSON,
		AgentHistory: histJSON,
	}, nil
}

func (d *Data) Save(dir string) error {
	if err := os.MkdirAll(dir, config.DirPermissions); err != nil {
		return fmt.Errorf("creating conversation dir: %w", err)
	}

	var uiMessages []json.RawMessage
	if err := json.Unmarshal(d.UIMessages, &uiMessages); err != nil {
		return fmt.Errorf("unmarshaling ui messages: %w", err)
	}

	var agentHistory []json.RawMessage
	if err := json.Unmarshal(d.AgentHistory, &agentHistory); err != nil {
		return fmt.Errorf("unmarshaling agent history: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(header{ID: d.ID}); err != nil {
		return fmt.Errorf("encoding header: %w", err)
	}

	for _, msg := range uiMessages {
		if err := enc.Encode(record{Kind: "ui", Data: msg}); err != nil {
			return fmt.Errorf("encoding ui message: %w", err)
		}
	}

	for _, msg := range agentHistory {
		if err := enc.Encode(record{Kind: "history", Data: msg}); err != nil {
			return fmt.Errorf("encoding history message: %w", err)
		}
	}

	path := filepath.Join(dir, d.ID+".jsonl")
	return os.WriteFile(path, buf.Bytes(), config.FilePermissions)
}

func LatestInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jsonl" {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no conversation files found")
	}

	sort.Strings(files)
	return files[len(files)-1], nil
}

// Preview holds the minimal info needed to display a conversation in a picker.
type Preview struct {
	ID       string
	FirstMsg string // first user message, empty if none
}

// ReadPreview opens a conversation file and reads only enough to extract the
// ID and the first user message. It is cheaper than a full Load.
func ReadPreview(path string) (Preview, error) {
	f, err := os.Open(path)
	if err != nil {
		return Preview{}, fmt.Errorf("opening conversation: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	if !scanner.Scan() {
		return Preview{}, fmt.Errorf("empty conversation file")
	}
	var h header
	if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {
		return Preview{}, fmt.Errorf("parsing header: %w", err)
	}

	p := Preview{ID: h.ID}
	for scanner.Scan() {
		var r record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		if r.Kind != "ui" {
			continue
		}
		var entry struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if json.Unmarshal(r.Data, &entry) != nil {
			continue
		}
		if entry.Role == "user" && entry.Content != "" {
			p.FirstMsg = entry.Content
			break
		}
	}
	return p, nil
}

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".yaca", "conversations")
	}
	return filepath.Join(home, ".yaca", "conversations")
}
