package components

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	paneStyle = lipgloss.NewStyle().
			Padding(0, 1)

	paneBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151"))

	paneBorderFocused = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED"))

	divider = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151"))
)

// LayoutDimensions holds the calculated dimensions for the dual-pane layout.
type LayoutDimensions struct {
	LeftWidth   int
	RightWidth  int
	ContentHeight int
}

// CalculateLayout computes the pane dimensions from the terminal size.
// Header takes 1 line, status bar 1 line, footer 1 line, borders ~4 lines.
func CalculateLayout(termWidth, termHeight int) LayoutDimensions {
	leftWidth := 18 // fixed workspace list width

	// Account for borders: 2 (left border) + 2 (right border) + 1 (divider)
	rightWidth := termWidth - leftWidth - 5
	if rightWidth < 20 {
		rightWidth = 20
	}

	// Header (1) + status bar (1) + footer (1) + top/bottom borders (2) + padding
	contentHeight := termHeight - 6
	if contentHeight < 4 {
		contentHeight = 4
	}

	return LayoutDimensions{
		LeftWidth:     leftWidth,
		RightWidth:    rightWidth,
		ContentHeight: contentHeight,
	}
}

// RenderDualPane combines the left and right panes with borders and styling.
func RenderDualPane(
	left string,
	right string,
	leftFocused bool,
	dims LayoutDimensions,
) string {
	leftBorder := paneBorder
	rightBorder := paneBorder

	if leftFocused {
		leftBorder = paneBorderFocused
	} else {
		rightBorder = paneBorderFocused
	}

	leftPane := leftBorder.
		Width(dims.LeftWidth).
		Height(dims.ContentHeight).
		Render(left)

	rightPane := rightBorder.
		Width(dims.RightWidth).
		Height(dims.ContentHeight).
		Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
