package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/tui/bridge"
	"go.dot.industries/vx/internal/tui/components"
	"go.dot.industries/vx/internal/vault"
)

// focusPane tracks which pane has focus.
type focusPane int

const (
	focusWorkspaces focusPane = iota
	focusSecrets
)

// popup identifies which popup is currently open.
type popup int

const (
	popupNone popup = iota
	popupHelp
	popupEnvPicker
	popupDetail
	popupVaultBrowser
	popupMappingForm
	popupConfirm
)

// model is the root Bubble Tea model for the vx TUI.
type model struct {
	// Dimensions
	width  int
	height int

	// Core data
	bridge      *bridge.Bridge
	config      *config.RootConfig
	rootDir     string
	env         string
	environments []string
	vaultClient *vault.Client

	// UI state
	focus       focusPane
	activePopup popup
	filtering   bool
	filterText  string

	// Components
	workspaces components.WorkspaceList
	secrets    components.SecretTable
	statusBar  components.StatusBar

	// Popup state
	helpContent     string
	envPickerCursor int

	// Detail popup
	detailEnvVar  string
	detailPath    string
	detailValue   string
	detailLoading bool
	detailError   string

	// Vault browser state
	vaultBrowserPath    string
	vaultBrowserEntries []VaultEntry
	vaultBrowserCursor  int
	vaultBrowserLoading bool
	vaultBrowserError   string

	// Mapping form state
	mappingFormEnvVar    string
	mappingFormPath      string
	mappingFormTarget    int // index into WorkspaceFiles()
	mappingFormField     int // 0=path, 1=envvar, 2=target
	mappingFormIsEdit    bool
	mappingFormOldEnvVar string

	// Confirm dialog state
	confirmEnvVar  string
	confirmFile    string
	confirmCursor  int // 0=cancel, 1=confirm

	// Status message timer
	statusClearTimer *time.Timer

	// Error state
	fatalError string
}

// newModel creates the initial model with the given bridge.
func newModel(b *bridge.Bridge) model {
	return model{
		bridge: b,
		focus:  focusWorkspaces,
	}
}

// Init loads the config on startup.
func (m model) Init() tea.Cmd {
	return loadConfigCmd(m.bridge)
}

// loadConfigCmd creates a command that loads the root config.
func loadConfigCmd(b *bridge.Bridge) tea.Cmd {
	return func() tea.Msg {
		cfg, rootDir, err := b.LoadConfig()
		if err != nil {
			return configErrorMsg{err: err}
		}
		return configLoadedMsg{config: cfg, rootDir: rootDir}
	}
}

// loadWorkspaceDataCmd creates a command that loads merged data for a workspace.
func loadWorkspaceDataCmd(b *bridge.Bridge, cfg *config.RootConfig, rootDir, workspace, env string) tea.Cmd {
	return func() tea.Msg {
		var merged *config.MergedConfig
		var err error

		if workspace == "[root]" || workspace == "" {
			merged, err = b.MergeRootOnly(cfg, env)
		} else {
			merged, err = b.MergeForWorkspace(cfg, rootDir, workspace, env)
		}

		if err != nil {
			return workspaceDataErrorMsg{err: err}
		}

		return workspaceDataLoadedMsg{
			secrets: merged.Secrets,
			source:  workspace,
		}
	}
}

// View renders the entire TUI.
func (m model) View() string {
	if m.fatalError != "" {
		return lipgloss.NewStyle().
			Foreground(colorError).
			Padding(1, 2).
			Render("Error: " + m.fatalError + "\n\nPress q to quit.")
	}

	if m.config == nil {
		return lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(1, 2).
			Render("Loading configuration...")
	}

	if m.width < minTermWidth || m.height < minTermHeight {
		return lipgloss.NewStyle().
			Foreground(colorWarning).
			Padding(1, 2).
			Render("Terminal too small. Please resize.")
	}

	dims := components.CalculateLayout(m.width, m.height)

	// Header
	header := components.RenderHeader(m.width, m.env)

	// Dual pane
	leftContent := m.workspaces.View(dims.LeftWidth-2, dims.ContentHeight-2)
	rightContent := m.secrets.View(dims.RightWidth-2, dims.ContentHeight-2)
	panes := components.RenderDualPane(
		leftContent,
		rightContent,
		m.focus == focusWorkspaces,
		dims,
	)

	// Status bar
	m.statusBar.SecretCount = m.secrets.TotalLen()
	m.statusBar.Filtering = m.filtering
	m.statusBar.FilterText = m.filterText
	statusLine := m.statusBar.View(m.width)

	// Footer
	footer := components.RenderFooter(m.width, m.filtering, m.activePopup != popupNone)

	// Compose full layout
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		panes,
		statusLine,
		footer,
	)

	// Overlay popup if active
	if m.activePopup != popupNone {
		view = m.overlayPopup(view)
	}

	return view
}

// overlayPopup renders the active popup centered on the screen.
func (m model) overlayPopup(base string) string {
	var popupContent string

	switch m.activePopup {
	case popupHelp:
		popupContent = m.renderHelpPopup()
	case popupEnvPicker:
		popupContent = m.renderEnvPickerPopup()
	case popupDetail:
		popupContent = m.renderDetailPopup()
	case popupVaultBrowser:
		popupContent = m.renderVaultBrowserPopup()
	case popupMappingForm:
		popupContent = m.renderMappingFormPopup()
	case popupConfirm:
		popupContent = m.renderConfirmPopup()
	default:
		return base
	}

	return placeOverlay(m.width, m.height, popupContent, base)
}

// placeOverlay centers the overlay string on top of base.
func placeOverlay(width, height int, overlay, base string) string {
	overlayWidth := lipgloss.Width(overlay)
	overlayHeight := lipgloss.Height(overlay)

	x := (width - overlayWidth) / 2
	if x < 0 {
		x = 0
	}
	y := (height - overlayHeight) / 2
	if y < 0 {
		y = 0
	}

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
	)
}
