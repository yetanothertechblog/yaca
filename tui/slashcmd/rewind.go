package slashcmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func init() {
	Register(Command{"/rewind", "Rewind to a previous message"})
}

// RewindOverlay holds the state of the rewind message picker overlay.
// It is also reused as a generic list picker (e.g. conversation resume).
type RewindOverlay struct {
	Title  string // overlay heading; defaults to "Rewind to message" if empty
	Items  []RewindItem
	Cursor int
}

// RewindItem represents a selectable item in the picker overlay.
type RewindItem struct {
	Text         string // display label (truncated)
	FullText     string // full original text (used when rewinding)
	MessageIndex int    // index into m.messages (rewind only)
	HistoryIndex int    // index into m.history  (rewind only)
	Path         string // file path             (conversation resume only)
}

// View renders the picker overlay as a centered box.
func (r *RewindOverlay) View(width, height int) string {
	heading := r.Title
	if heading == "" {
		heading = "Rewind to message"
	}
	title := overlayTitleStyle.Render(heading)

	// Show a scrollable window of ~10 items around cursor
	const windowSize = 10
	start := r.Cursor - windowSize/2
	end := start + windowSize
	if start < 0 {
		start = 0
		end = windowSize
	}
	if end > len(r.Items) {
		end = len(r.Items)
		start = end - windowSize
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		item := r.Items[i]
		label := fmt.Sprintf("%d. %s", i+1, item.Text)
		if i == r.Cursor {
			lines = append(lines, overlaySelectedStyle.Render("> "+label))
		} else {
			lines = append(lines, overlayOptionStyle.Render("  "+label))
		}
	}

	// Add scroll indicators
	if start > 0 {
		lines = append([]string{overlayOptionStyle.Render("  ↑ more")}, lines...)
	}
	if end < len(r.Items) {
		lines = append(lines, overlayOptionStyle.Render("  ↓ more"))
	}

	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		overlayOptionStyle.Render("↑↓ navigate · enter select · esc cancel")

	boxWidth := 70
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	box := overlayBoxStyle.Width(boxWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
