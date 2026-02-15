package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	footerKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	footerDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	footerSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))
)

type footerBinding struct {
	key  string
	desc string
}

// RenderFooter returns the keybinding hints bar for the given focus mode.
func RenderFooter(width int, filtering bool, popupOpen bool) string {
	var bindings []footerBinding

	if popupOpen {
		bindings = []footerBinding{
			{"esc", "close"},
		}
	} else if filtering {
		bindings = []footerBinding{
			{"esc", "stop filter"},
			{"enter", "apply"},
		}
	} else {
		bindings = []footerBinding{
			{"j/k", "nav"},
			{"tab", "pane"},
			{"e", "env"},
			{"/", "filter"},
			{"enter", "view"},
			{"a", "add"},
			{"r", "edit"},
			{"d", "del"},
			{"c", "copy"},
			{"?", "help"},
			{"q", "quit"},
		}
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts, footerKey.Render(b.key)+footerDesc.Render(":"+b.desc))
	}

	line := strings.Join(parts, footerSep.Render("  "))

	if lipgloss.Width(line) > width {
		line = line[:width]
	}

	return lipgloss.NewStyle().Width(width).Render(line)
}
