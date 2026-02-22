package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestWriteMessage_Format(t *testing.T) {
	req := Request{JSONRPC: "2.0", ID: 1, Method: "initialize"}
	var buf bytes.Buffer
	if err := WriteMessage(&buf, req); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	output := buf.String()

	// Must have Content-Length header with correct value
	body, _ := json.Marshal(req)
	expected := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if !strings.HasPrefix(output, expected) {
		t.Errorf("output should start with %q, got %q", expected, output[:min(len(output), len(expected)+10)])
	}

	// Body must be valid JSON matching original
	bodyStr := output[len(expected):]
	var decoded Request
	if err := json.Unmarshal([]byte(bodyStr), &decoded); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if decoded.Method != "initialize" || decoded.ID != 1 {
		t.Errorf("body mismatch: %+v", decoded)
	}
}

func TestWriteMessage_ErrorOnBadWriter(t *testing.T) {
	err := WriteMessage(&errorWriter{}, Request{JSONRPC: "2.0", ID: 1, Method: "test"})
	if err == nil {
		t.Error("expected error writing to broken writer")
	}
}

func TestReadMessage_ParsesBody(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)

	data, err := ReadMessage(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(data) != body {
		t.Errorf("got %q, want %q", data, body)
	}
}

func TestReadMessage_MissingContentLength(t *testing.T) {
	input := "X-Custom: value\r\n\r\n{}"
	_, err := ReadMessage(bufio.NewReader(strings.NewReader(input)))
	if err == nil {
		t.Error("expected error for missing Content-Length")
	}
}

func TestReadMessage_InvalidContentLength(t *testing.T) {
	input := "Content-Length: abc\r\n\r\n{}"
	_, err := ReadMessage(bufio.NewReader(strings.NewReader(input)))
	if err == nil {
		t.Error("expected error for non-numeric Content-Length")
	}
}

func TestReadMessage_TruncatedBody(t *testing.T) {
	// Claim 100 bytes but only provide 5
	input := "Content-Length: 100\r\n\r\nhello"
	_, err := ReadMessage(bufio.NewReader(strings.NewReader(input)))
	if err == nil {
		t.Error("expected error for truncated body")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "textDocument/didOpen",
		Params:  map[string]string{"uri": "file:///test.go"},
	}

	var buf bytes.Buffer
	if err := WriteMessage(&buf, req); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	data, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.ID != 42 || decoded.Method != "textDocument/didOpen" {
		t.Errorf("round-trip mismatch: %+v", decoded)
	}
}

func TestNotification_NoIDField(t *testing.T) {
	notif := Notification{JSONRPC: "2.0", Method: "initialized"}
	data, _ := json.Marshal(notif)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, ok := raw["id"]; ok {
		t.Error("notification should not have 'id' field")
	}
}

func TestRPCError_Format(t *testing.T) {
	err := &RPCError{Code: -32600, Message: "Invalid Request"}
	if err.Error() != "rpc error -32600: Invalid Request" {
		t.Errorf("Error() = %q", err.Error())
	}
}

type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}
