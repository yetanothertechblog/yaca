package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Copper-brass steampunk color palette.
var (
	colorBrass      = lipgloss.Color("#B5A642")
	colorCopper     = lipgloss.Color("#B87333")
	colorBronze     = lipgloss.Color("#CD7F32")
	colorRust       = lipgloss.Color("#FF0000")
	colorSteam      = lipgloss.Color("#C8C8B4")
	colorDarkSteel  = lipgloss.Color("#3B3B2E")
	colorPatina     = lipgloss.Color("#4A7C6F")
	colorAmber      = lipgloss.Color("#FFBF00")
	colorOxidized   = lipgloss.Color("#6B4226")
	colorParchment  = lipgloss.Color("#D4C5A9")
	colorDimBrass   = lipgloss.Color("#8B7D3C")
	colorForgedIron = lipgloss.Color("#555548")
	colorBlue       = lipgloss.Color("#4169E1")
)

// Shared styles used across TUI components.
var (
	// Layout
	separatorStyle = lipgloss.NewStyle().
			Foreground(colorForgedIron)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorDimBrass)

	bypassStyle = lipgloss.NewStyle().
			Foreground(colorRust).
			Bold(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorBrass)

	// Messages
	toolCmdStyle = lipgloss.NewStyle().
			Foreground(colorCopper).
			Bold(true)

	userMessageStyle = lipgloss.NewStyle().
				Background(colorDarkSteel)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRust)

	deniedStyle = lipgloss.NewStyle().
			Foreground(colorRust).
			Bold(true)

	// Diffs
	diffAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#2D5A27"))

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#6B1A1A"))

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(colorBrass).
			Bold(true)

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	diffLineNumStyle = lipgloss.NewStyle().
				Foreground(colorForgedIron)

	diffAddedLineNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4A8C45")).
				Background(lipgloss.Color("#2D5A27"))

	diffRemovedLineNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8C3A3A")).
				Background(lipgloss.Color("#6B1A1A"))

	diffAddedMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4A8C45")).
				Background(lipgloss.Color("#2D5A27"))

	diffRemovedMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8C3A3A")).
				Background(lipgloss.Color("#6B1A1A"))

	toolBulletStyle = lipgloss.NewStyle().
			Foreground(colorCopper).
			Bold(true)

	toolIndentStyle = lipgloss.NewStyle().
			Foreground(colorForgedIron)

	// Permission prompt
	permBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAmber).
			Padding(1, 2)

	permTitleStyle = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)

	permOptionStyle = lipgloss.NewStyle().
			Foreground(colorParchment)

	requirePermissionsStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	permSelectedStyle = lipgloss.NewStyle().
				Foreground(colorAmber).
				Bold(true)

	thinkingLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080"))

	thinkingContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080")).
				Italic(true)
)
