# YACA (Yet Another Coding Agent)

A lightweight, terminal-based AI coding assistant with rich TUI interface. Written entirely in go, using the `BubbleTea` framework for TUI rendering, and `Glamour` for markdown rendering.

I have basically combined my favourite features from the 3 popular coding assistants (Claude Code, Codex, Opencode)

A lot of this code was written by YACA, essentially bootstrapping itself.


## Features

- **Lightweight**: Just ~40 MB while running (The LSP itself is much larger of course)
- **Supports Message Streaming**
- **Model Selector** - Works with GLM models + OpenAI models for now
- **LSP** - Runs a background LSP thread to help the model get feedback on edits and writes
- **The minimum essential /commands**
   - /resume
   - /rewind
   - /compact
   - /clear and /exit
- **Ask for permissions mode or Bypass Permissions Mode**

## Setup and Installation
To build from source, just clone this repo and run 
```
go run .
```

To download the release binary and run, follow the instructions in the release section

## Demo

Heres a demo using GLM-4.5 Air

![Demo](https://github.com/yetanothertechblog/yaca/blob/9359ddb22da65d4d16042e738d9c423898e2d846/yaca-demo.gif)  

### Available Tools
The AI assistant has access to these tools - adding more was just hurting performance, Keep It Simple, Stupid
- **📖 Read File**
- **📁 List Files**
- **✏️ Edit File**
- **📝 Write File**
- **💻 Bash**
- **🔍 Search**


## License

This project is open source and available under the MIT License.
