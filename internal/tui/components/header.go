package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	headerEnvBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 1)
)

// RenderHeader returns the header bar with title and environment badge.
func RenderHeader(width int, env string) string {
	title := headerTitle.Render("vx — Secret Browser")
	badge := headerEnvBadge.Render(fmt.Sprintf("env: %s", env))

	spacer := width - lipgloss.Width(title) - lipgloss.Width(badge)
	if spacer < 1 {
		spacer = 1
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		lipgloss.NewStyle().Width(spacer).Render(""),
		badge,
	)
}
