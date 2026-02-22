# Claude API Response Format Analysis

## Overview
This document analyzes the Claude API response format for tool calls, specifically focusing on the `anthropic_edit_response.json` streaming response and how it relates to diff generation.

## 1. API Response Structure

### Streaming Event Format
The Claude API uses **Server-Sent Events (SSE)** format with `event:` and `data:` lines:

```json
event: message_start
data: {"type":"message_start",...}

event: content_block_start  
data: {"type":"content_block_start",...}

event: content_block_delta
data: {"type":"content_block_delta",...}

event: content_block_stop
data: {"type":"content_block_stop",...}

event: message_stop
data: {"type":"message_stop"}
```

### Tool Call Response Format

#### **Tool Use Block Start:**
```json
event: content_block_start
data: {
  "type": "content_block_start",
  "index": 2,
  "content_block": {
    "type": "tool_use",
    "id": "toolu_01MyXtFtPqRzKq5vgnf6dap4",
    "name": "Write",
    "input": {},
    "caller": {"type": "direct"}
  }
}
```

#### **Tool Input JSON Streaming:**
The tool input is streamed as `input_json_delta` events:
```json
event: content_block_delta
data: {
  "type": "content_block_delta", 
  "index": 2,
  "delta": {
    "type": "input_json_delta",
    "partial_json": "{\"f"
  }
}
```

#### **Tool Use Block End:**
```json
event: content_block_stop
data: {
  "type": "content_block_stop",
  "index": 2
}
```

## 2. Tool Call Content Structure

### **Tool Metadata:**
- **`id`**: Unique tool call ID (`toolu_...`)
- **`name`**: Tool name (`Write`, `Edit`, etc.)
- **`input`**: Empty object initially (filled via streaming)
- **`caller`**: Type of caller (`"direct"`)

### **Streaming Input:**
- **`index: 2`**: Indicates this is the tool call content block
- **`input_json_delta`**: Type for streaming JSON input
- **`partial_json`**: JSON chunks that need to be concatenated

### **Complete Tool Input:**
```json
{
  "file_path": "/Users/hp/workspace/go-tui/conversation/conversation.go",
  "content": "package conversationimport (\t\"bufio\"\t\"bytes\"..."
}
```

**Key Point:** The tool input only contains:
1. **`file_path`**: Where to write the file
2. **`content`**: The NEW content to write

**NO old content, NO line numbers, NO context about existing code.**

## 3. Diff Generation Process

### **How the Agent Actually Works:**

1. **Read existing file** from the `file_path` on disk
2. **Compare** the new content (from API) with the existing content (from disk)
3. **Generate diff** showing differences
4. **Apply changes** based on the diff's +/- markers and context lines

### **Diff Generation Algorithm**

#### **1. Tokenization**
Both old and new content are split into lines:

```go
// Old file content (read from disk)
old_lines := []string{
    "func Load(path string) (*Data, error) {",
    "  b, err := os.ReadFile(path)",
    "  if err != nil {",
    "    return nil, fmt.Errorf(\"reading conversation file: %w\", err)",
    "  }",
    "  var d Data",
    "  if err := json.Unmarshal(b, &d); err != nil {",
    "    return nil, fmt.Errorf(\"parsing conversation file: %w\", err)",
    "  }",
    "  return &d, nil",
    "}",
}

// New content (from API)
new_lines := []string{
    "func Load(path string) (*Data, error) {",
    "  f, err := os.Open(path)",
    "  defer f.Close()",
    "  scanner := bufio.NewScanner(f)",
    "  scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)",
    "  if !scanner.Scan() {",
    "    return nil, fmt.Errorf(\"empty conversation file\")",
    "  }",
    "  var h header",
    "  if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {",
    "    return nil, fmt.Errorf(\"parsing conversation header: %w\", err)",
    "  }",
    // ... more lines
    "}",
}
```

#### **2. Longest Common Subsequence (LCS)**
The algorithm finds the longest sequence of lines that appear in both files in the same order:

```go
// LCS would identify common lines like:
- "func Load(path string) (*Data, error) {"
- "  if err != nil {"
- "}"
```

#### **3. Difference Calculation**
Once LCS is found, differences are identified:

```go
// Lines only in old file (to be removed):
- "  b, err := os.ReadFile(path)"
- "  var d Data"
- "  if err := json.Unmarshal(b, &d); err != nil {"
- "    return nil, fmt.Errorf(\"parsing conversation file: %w\", err)"
- "  }"
- "  return &d, nil"

// Lines only in new file (to be added):
+ "  f, err := os.Open(path)"
+ "  defer f.Close()"
+ "  scanner := bufio.NewScanner(f)"
+ // ... many more lines
```

#### **4. Hunk Formation**
Related changes are grouped into "hunks":

```diff
@@ -30,8 +41,63 @@
func Load(path string) (*Data, error) {
-  b, err := os.ReadFile(path)
+  f, err := os.Open(path)
+  defer f.Close()
+  scanner := bufio.NewScanner(f)
+  scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
+  if !scanner.Scan() {
+    return nil, fmt.Errorf("empty conversation file")
+  }
+  var h header
+  // ... 60 more lines
  if err != nil {
    return nil, fmt.Errorf("reading conversation file: %w", err)
  }
-  var d Data
-  if err := json.Unmarshal(b, &d); err != nil {
-    return nil, fmt.Errorf("parsing conversation file: %w\", err)"
-  }
-  return &d, nil
}
```

#### **5. Application Algorithm**
When applying the diff:

1. **Parse the diff** to identify hunks
2. **Find anchor points** using context lines
3. **Calculate exact line numbers** in the current file
4. **Apply changes**:
   - Remove lines marked with `-`
   - Insert lines marked with `+` at the calculated positions

## 4. Key Insights

### **What the API Provides:**
- Only the **new content** to write
- Tool metadata (ID, name, etc.)
- Streaming JSON chunks

### **What the Agent Must Do:**
- Read the **existing file** from disk
- Generate the **diff** by comparing old vs new
- Use **context lines** to find change locations
- Apply changes based on **+/- markers**

### **How Line Numbers Are Determined:**
- **Not provided by API**
- **Calculated by the agent** using LCS algorithm
- **Context lines** serve as anchor points
- **Line numbers change** as changes are applied sequentially

### **Why This Design:**
- **Efficiency**: API only sends what's needed (new content)
- **Safety**: Agent has full context of existing file
- **Flexibility**: Can handle any type of change (insert, delete, replace)
- **Precision**: LCS algorithm ensures accurate change detection

## 5. Practical Example

### **Tool Call Content:**
```json
{
  "file_path": "/Users/hp/workspace/go-tui/conversation/conversation.go",
  "content": "package conversation\n\nimport (\n\t\"bufio\"\n\t\"bytes\"\n\t// ... rest of new code\n)\n\nfunc Load(path string) (*Data, error) {\n\t// New JSONL implementation\n}\n"
}
```

### **Agent Process:**
1. Reads existing `conversation.go` from disk
2. Compares with new content from API
3. Generates diff showing:
   - Added imports (`bufio`, `bytes`)
   - Replaced `Load` function
   - Added new structs (`header`, `record`)
4. Applies changes using diff markers

### **Result:**
The diff you see in the UI is **generated by the agent**, not provided by Claude. The API only provides the raw new content.

## 6. Summary

- **API Response**: Contains only new content and tool metadata
- **Old Content**: Comes from reading the file on disk
- **Diff Generation**: Done by agent using LCS algorithm
- **Line Numbers**: Calculated, not provided
- **Application**: Uses context lines and +/- markers to make precise changes

This design ensures efficient communication while maintaining the ability to make precise code changes.