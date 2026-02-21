package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go-tui/config"
	"go-tui/conversation"
	"go-tui/llm"
	"go-tui/settings"
	"go-tui/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	resume := flag.Bool("resume", false, "resume a conversation (pass UUID as positional arg for specific conversation)")
	flag.Parse()

	var resumeID string
	if flag.NArg() > 0 {
		resumeID = flag.Arg(0)
	}

	s, err := settings.Load()
	if err != nil {
		log.Printf("failed to load settings: %v", err)
	}

	if active := s.ActiveModel(); active != "" {
		if md := config.ModelByName(active); md != nil {
			if key := s.APIKey(active); key != "" {
				llm.Configure(md.APIURL, key, active)
			}
		}
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	logDir := filepath.Join(workingDir, "log")
	if err := os.MkdirAll(logDir, config.DirPermissions); err != nil {
		fmt.Printf("Error creating log dir: %v\n", err)
		os.Exit(1)
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, config.LogPermissions)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("starting go-tui")

	convDir := conversation.Dir()
	var conv *conversation.Data

	if *resume && resumeID == "" {
		// -resume with no UUID: load latest conversation
		latestFile, err := conversation.LatestInDir(convDir)
		if err != nil {
			fmt.Printf("No conversations to resume.\n")
			os.Exit(1)
		}
		resumeID = strings.TrimSuffix(latestFile, ".jsonl")
		path := filepath.Join(convDir, resumeID+".jsonl")
		conv, err = conversation.Load(path)
		if err != nil {
			fmt.Printf("Error loading conversation: %v\n", err)
			os.Exit(1)
		}
		log.Printf("resumed latest conversation: %s", conv.ID)
	} else if resumeID != "" {
		// Explicit UUID provided: go run . -resume <uuid>
		path := filepath.Join(convDir, resumeID+".jsonl")
		conv, err = conversation.Load(path)
		if err != nil {
			fmt.Printf("Error loading conversation: %v\n", err)
			os.Exit(1)
		}
		log.Printf("resumed conversation: %s", conv.ID)
	} else {
		// No resume ID: create new conversation
		conv = conversation.New()
		log.Printf("new conversation: %s", conv.ID)
	}

	m := tui.New(workingDir, conv, s)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	m.Shutdown()
}
