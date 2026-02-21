package slashcmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func init() {
	Register(Command{"/model", "Switch the active model"})
}

// ModelOverlay holds the state of the model selector overlay.
// When AwaitingKey is true, it shows an API key input prompt instead of the model list.
type ModelOverlay struct {
	Items       []string // model names
	Cursor      int
	Active      string // currently active model name
	AwaitingKey bool   // true when prompting for API key
	SelectedModel string // model that was selected (used during key input)
	KeyInput    string // accumulated key input
}

// View renders the model selector overlay as a centered box.
func (o *ModelOverlay) View(width, height int) string {
	if o.AwaitingKey {
		return o.viewKeyInput(width, height)
	}
	return o.viewModelList(width, height)
}

func (o *ModelOverlay) viewModelList(width, height int) string {
	title := overlayTitleStyle.Render("Select Model")

	var lines []string
	for i, name := range o.Items {
		label := name
		if name == o.Active {
			label += " (active)"
		}
		if i == o.Cursor {
			lines = append(lines, overlaySelectedStyle.Render("> "+label))
		} else {
			lines = append(lines, overlayOptionStyle.Render("  "+label))
		}
	}

	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		overlayOptionStyle.Render("↑↓ navigate · enter select · esc cancel")

	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	box := overlayBoxStyle.Width(boxWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (o *ModelOverlay) viewKeyInput(width, height int) string {
	title := overlayTitleStyle.Render("API Key for " + o.SelectedModel)

	masked := strings.Repeat("*", len(o.KeyInput)) + "█"

	content := title + "\n\n" +
		overlayOptionStyle.Render("Paste or type your API key:") + "\n\n" +
		overlaySelectedStyle.Render(masked) + "\n\n" +
		overlayOptionStyle.Render("enter confirm · esc cancel")

	boxWidth := 60
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	box := overlayBoxStyle.Width(boxWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
