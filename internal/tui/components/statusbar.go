package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	statusLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA"))

	statusCount = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	statusFilter = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))

	statusError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	statusSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))
)

// StatusBar holds the state for the status bar component.
type StatusBar struct {
	FilterText  string
	Filtering   bool
	SecretCount int
	Message     string
	IsError     bool
}

// View renders the status bar.
func (sb *StatusBar) View(width int) string {
	var left string
	if sb.Filtering {
		left = statusLabel.Render("Filter: ") + statusFilter.Render(sb.FilterText+"_")
	} else if sb.Message != "" {
		if sb.IsError {
			left = statusError.Render(sb.Message)
		} else {
			left = statusSuccess.Render(sb.Message)
		}
	} else if sb.FilterText != "" {
		left = statusLabel.Render("Filter: ") + statusFilter.Render(sb.FilterText)
	}

	right := statusCount.Render(fmt.Sprintf("%d secrets", sb.SecretCount))

	spacer := width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacer < 1 {
		spacer = 1
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		lipgloss.NewStyle().Width(spacer).Render(""),
		right,
	)
}
