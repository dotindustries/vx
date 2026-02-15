// Package tui implements an interactive terminal UI for browsing and managing
// vx secret mappings using the Charmbracelet Bubble Tea framework.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"go.dot.industries/vx/internal/tui/bridge"
)

// Run starts the interactive TUI. It blocks until the user quits.
func Run(configPath, vaultAddr, authMethod, roleID, secretID string) error {
	b := bridge.New(configPath, vaultAddr, authMethod, roleID, secretID)
	m := newModel(b)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
