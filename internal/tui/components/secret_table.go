package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	stSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	stNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))

	stPath = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	stTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#6B7280"))

	stFocusedRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			Bold(true)
)

// SecretRow represents a single secret mapping for display.
type SecretRow struct {
	EnvVar    string
	VaultPath string // interpolated path for display
	RawPath   string // template path with ${env} for editing
}

// SecretTable holds the state for the secret list pane.
type SecretTable struct {
	AllRows  []SecretRow // all rows before filtering
	Rows     []SecretRow // visible rows after filtering
	Cursor   int
	Focused  bool
	Filter   string
	Offset   int // scroll offset for viewport
}

// NewSecretTable creates a table from secret mappings.
func NewSecretTable(secrets map[string]string, env string) SecretTable {
	rows := make([]SecretRow, 0, len(secrets))
	for envVar, rawPath := range secrets {
		interpolated := strings.ReplaceAll(rawPath, "${env}", env)
		rows = append(rows, SecretRow{
			EnvVar:    envVar,
			VaultPath: interpolated,
			RawPath:   rawPath,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].EnvVar < rows[j].EnvVar
	})

	return SecretTable{
		AllRows: rows,
		Rows:    rows,
		Cursor:  0,
		Focused: false,
	}
}

// SetSecrets replaces the table data and resets the cursor.
func (st *SecretTable) SetSecrets(secrets map[string]string, env string) {
	rows := make([]SecretRow, 0, len(secrets))
	for envVar, rawPath := range secrets {
		interpolated := strings.ReplaceAll(rawPath, "${env}", env)
		rows = append(rows, SecretRow{
			EnvVar:    envVar,
			VaultPath: interpolated,
			RawPath:   rawPath,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].EnvVar < rows[j].EnvVar
	})

	st.AllRows = rows
	st.ApplyFilter(st.Filter)
	st.Cursor = 0
	st.Offset = 0
}

// ApplyFilter filters rows by the given string (case-insensitive match on
// env var name or vault path).
func (st *SecretTable) ApplyFilter(filter string) {
	st.Filter = filter
	if filter == "" {
		st.Rows = st.AllRows
		return
	}

	lower := strings.ToLower(filter)
	filtered := make([]SecretRow, 0)
	for _, row := range st.AllRows {
		if strings.Contains(strings.ToLower(row.EnvVar), lower) ||
			strings.Contains(strings.ToLower(row.VaultPath), lower) {
			filtered = append(filtered, row)
		}
	}
	st.Rows = filtered

	if st.Cursor >= len(st.Rows) {
		st.Cursor = max(0, len(st.Rows)-1)
	}
	st.Offset = 0
}

// Selected returns the currently selected row, or nil if empty.
func (st *SecretTable) Selected() *SecretRow {
	if st.Cursor >= 0 && st.Cursor < len(st.Rows) {
		return &st.Rows[st.Cursor]
	}
	return nil
}

// MoveUp moves the cursor up by one.
func (st *SecretTable) MoveUp() {
	if st.Cursor > 0 {
		st.Cursor--
	}
}

// MoveDown moves the cursor down by one.
func (st *SecretTable) MoveDown() {
	if st.Cursor < len(st.Rows)-1 {
		st.Cursor++
	}
}

// Len returns the number of visible rows.
func (st *SecretTable) Len() int {
	return len(st.Rows)
}

// TotalLen returns the number of total (unfiltered) rows.
func (st *SecretTable) TotalLen() int {
	return len(st.AllRows)
}

// View renders the secret table pane.
func (st *SecretTable) View(width, height int) string {
	var b strings.Builder

	// Header
	workspace := ""
	if st.Focused {
		workspace = " (focused)"
	}
	countStr := fmt.Sprintf("%d keys", len(st.Rows))
	titleLeft := stTitle.Render("Secrets" + workspace)

	spacer := width - lipgloss.Width(titleLeft) - lipgloss.Width(countStr) - 2
	if spacer < 1 {
		spacer = 1
	}

	b.WriteString(titleLeft)
	b.WriteString(lipgloss.NewStyle().Width(spacer).Render(""))
	b.WriteString(stTitle.Render(countStr))
	b.WriteString("\n")

	if len(st.Rows) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Render("  No secrets found"))
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render(b.String())
	}

	// Ensure cursor is visible within the viewport
	viewportHeight := height - 2 // title + margin
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if st.Cursor < st.Offset {
		st.Offset = st.Cursor
	}
	if st.Cursor >= st.Offset+viewportHeight {
		st.Offset = st.Cursor - viewportHeight + 1
	}

	// Column widths: envVar gets ~40% of space, path gets the rest
	envVarWidth := width * 2 / 5
	pathWidth := width - envVarWidth - 3 // 3 for prefix + space

	for i := st.Offset; i < len(st.Rows) && i < st.Offset+viewportHeight; i++ {
		row := st.Rows[i]
		prefix := "  "
		nameStyle := stNormal
		pathStyle := stPath

		if i == st.Cursor {
			prefix = "> "
			if st.Focused {
				nameStyle = stSelected
				pathStyle = stSelected
			} else {
				nameStyle = stFocusedRow
			}
		}

		envVar := truncate(row.EnvVar, envVarWidth)
		vaultPath := truncate(row.VaultPath, pathWidth)

		line := prefix + nameStyle.Render(padRight(envVar, envVarWidth)) + " " + pathStyle.Render(vaultPath)
		b.WriteString(line)
		if i < st.Offset+viewportHeight-1 {
			b.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String())
}

// truncate shortens a string to maxLen with ellipsis.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen || maxLen < 4 {
		return s
	}
	return s[:maxLen-1] + "…"
}

// padRight pads a string with spaces to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
