package tui

import "github.com/charmbracelet/lipgloss"

// Colors used throughout the TUI.
var (
	colorPrimary   = lipgloss.Color("#7C3AED") // violet-600
	colorSecondary = lipgloss.Color("#A78BFA") // violet-400
	colorMuted     = lipgloss.Color("#6B7280") // gray-500
	colorBorder    = lipgloss.Color("#374151") // gray-700
	colorBg        = lipgloss.Color("#111827") // gray-900
	colorBgAlt     = lipgloss.Color("#1F2937") // gray-800
	colorFg        = lipgloss.Color("#F9FAFB") // gray-50
	colorFgDim     = lipgloss.Color("#9CA3AF") // gray-400
	colorSuccess   = lipgloss.Color("#10B981") // emerald-500
	colorWarning   = lipgloss.Color("#F59E0B") // amber-500
	colorError     = lipgloss.Color("#EF4444") // red-500
	colorHighlight = lipgloss.Color("#312E81") // indigo-900
)

// Style presets for reuse across components.
var (
	styleBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleNormal = lipgloss.NewStyle().
			Foreground(colorFg)

	styleDim = lipgloss.NewStyle().
			Foreground(colorFgDim)

	styleBadge = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorPrimary).
			Padding(0, 1)

	styleErrorText = lipgloss.NewStyle().
			Foreground(colorError)

	styleSuccessText = lipgloss.NewStyle().
				Foreground(colorSuccess)

	styleWarningText = lipgloss.NewStyle().
				Foreground(colorWarning)

	styleKey = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleDesc = lipgloss.NewStyle().
			Foreground(colorFgDim)

	stylePopup = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)
)

// workspaceListWidth is the fixed width of the left pane.
const workspaceListWidth = 18

// minTermWidth is the minimum usable terminal width.
const minTermWidth = 60

// minTermHeight is the minimum usable terminal height.
const minTermHeight = 16
