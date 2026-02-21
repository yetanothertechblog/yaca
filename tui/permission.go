package tui

import (
	"fmt"

	"go-tui/config"
	"go-tui/permissions"
)

// Styles are defined in theme.go

type PermissionPrompt struct {
	ToolName   string
	Args       string
	Cursor     int
	WorkingDir string
}

func (p PermissionPrompt) View(width int) string {
	boxWidth := width - config.BoxPadding
	if boxWidth < config.MinBoxWidth {
		boxWidth = config.MinBoxWidth
	}

	// Render the tool call preview with bullet/indent style
	command := p.ToolName + ": " + p.Args
	header := formatCommand(command)
	bullet := toolBulletStyle.Render("⏺ ") + toolCmdStyle.Render(header)
	var toolSection string
	if diff := getDiffForPermission(p.ToolName, p.Args, p.WorkingDir); diff != "" {
		toolSection = bullet + "\n" + indentBlock(diff)
	} else {
		toolSection = bullet
	}

	title := permTitleStyle.Render("Tool Permission Required")

	alwaysLabel := p.ToolName
	if p.ToolName == "bash" {
		if prefix := permissions.BashCommandPrefix(p.Args); prefix != "" {
			alwaysLabel = fmt.Sprintf("bash: %s", prefix)
		}
	}
	options := []string{
		"Allow",
		fmt.Sprintf("Always Allow %s", alwaysLabel),
		"Deny",
	}

	var optionsView string
	for i, opt := range options {
		cursor := "  "
		style := permOptionStyle
		if i == p.Cursor {
			cursor = "> "
			style = permSelectedStyle
		}
		optionsView += cursor + style.Render(opt) + "\n"
	}

	content := fmt.Sprintf("%s\n\n%s", title, optionsView)
	permBox := permBoxStyle.Width(boxWidth).Render(content)

	return toolSection + "\n" + permBox
}
