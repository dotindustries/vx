package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	wsSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	wsNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	wsFocused = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))

	wsTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#6B7280")).
		MarginBottom(1)
)

// WorkspaceList holds the state for the workspace selector pane.
type WorkspaceList struct {
	Items    []string // workspace names (e.g. "web", "api")
	Cursor   int
	Focused  bool
	HasRoot  bool // whether to show "[root]" entry
}

// NewWorkspaceList creates a new workspace list from the given names.
func NewWorkspaceList(names []string, hasRootSecrets bool) WorkspaceList {
	return WorkspaceList{
		Items:   names,
		Cursor:  0,
		Focused: true,
		HasRoot: hasRootSecrets,
	}
}

// Selected returns the currently selected workspace name.
// Returns "[root]" if the cursor is past all workspace items.
func (wl *WorkspaceList) Selected() string {
	items := wl.allItems()
	if wl.Cursor >= 0 && wl.Cursor < len(items) {
		return items[wl.Cursor]
	}
	return ""
}

// MoveUp moves the cursor up by one.
func (wl *WorkspaceList) MoveUp() {
	if wl.Cursor > 0 {
		wl.Cursor--
	}
}

// MoveDown moves the cursor down by one.
func (wl *WorkspaceList) MoveDown() {
	items := wl.allItems()
	if wl.Cursor < len(items)-1 {
		wl.Cursor++
	}
}

// Len returns the total number of items including [root].
func (wl *WorkspaceList) Len() int {
	return len(wl.allItems())
}

// allItems returns Items plus "[root]" if applicable.
func (wl *WorkspaceList) allItems() []string {
	if wl.HasRoot {
		return append(wl.Items, "[root]")
	}
	return wl.Items
}

// View renders the workspace list pane.
func (wl *WorkspaceList) View(width, height int) string {
	var b strings.Builder

	b.WriteString(wsTitle.Render("Workspaces"))
	b.WriteString("\n")

	items := wl.allItems()
	for i, item := range items {
		if i >= height-2 { // leave room for title + margin
			break
		}

		prefix := "  "
		style := wsNormal
		if i == wl.Cursor {
			prefix = "> "
			if wl.Focused {
				style = wsSelected
			} else {
				style = wsFocused
			}
		}

		line := style.Render(prefix + item)
		b.WriteString(line)
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String())
}
