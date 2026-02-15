package tui

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/tui/bridge"
	"go.dot.industries/vx/internal/tui/components"
	"go.dot.industries/vx/internal/vault"
)

// Update handles all messages in the Elm architecture.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// --- Window ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// --- Config lifecycle ---
	case configLoadedMsg:
		return m.handleConfigLoaded(msg)

	case configErrorMsg:
		m.fatalError = msg.err.Error()
		return m, nil

	// --- Workspace data ---
	case workspaceSelectedMsg:
		return m.handleWorkspaceSelected(msg)

	case workspaceDataLoadedMsg:
		return m.handleWorkspaceDataLoaded(msg)

	case workspaceDataErrorMsg:
		m.statusBar.Message = "Error loading workspace: " + msg.err.Error()
		m.statusBar.IsError = true
		return m, clearStatusAfter(3 * time.Second)

	// --- Environment ---
	case envChangedMsg:
		return m.handleEnvChanged(msg)

	// --- Secret resolution ---
	case secretResolvedMsg:
		m.detailValue = msg.value
		m.detailLoading = false
		return m, nil

	case secretResolveErrorMsg:
		m.detailError = msg.err.Error()
		m.detailLoading = false
		return m, nil

	// --- Auth ---
	case authSucceededMsg:
		m.vaultClient = msg.client
		return m, nil

	case authFailedMsg:
		m.statusBar.Message = "Auth failed: " + msg.err.Error()
		m.statusBar.IsError = true
		return m, clearStatusAfter(5 * time.Second)

	// --- Vault browser ---
	case vaultListResultMsg:
		m.vaultBrowserEntries = msg.entries
		m.vaultBrowserLoading = false
		m.vaultBrowserCursor = 0
		return m, nil

	case vaultListErrorMsg:
		m.vaultBrowserError = msg.err.Error()
		m.vaultBrowserLoading = false
		return m, nil

	// --- CRUD ---
	case mappingSavedMsg:
		m.activePopup = popupNone
		m.statusBar.Message = "Mapping saved"
		m.statusBar.IsError = false
		return m, tea.Batch(
			loadConfigCmd(m.bridge),
			clearStatusAfter(3*time.Second),
		)

	case mappingSaveErrorMsg:
		m.statusBar.Message = "Save failed: " + msg.err.Error()
		m.statusBar.IsError = true
		return m, clearStatusAfter(5 * time.Second)

	case mappingDeletedMsg:
		m.activePopup = popupNone
		m.statusBar.Message = "Mapping deleted"
		m.statusBar.IsError = false
		return m, tea.Batch(
			loadConfigCmd(m.bridge),
			clearStatusAfter(3*time.Second),
		)

	case mappingDeleteErrorMsg:
		m.statusBar.Message = "Delete failed: " + msg.err.Error()
		m.statusBar.IsError = true
		return m, clearStatusAfter(5 * time.Second)

	// --- Status ---
	case statusMsg:
		m.statusBar.Message = msg.text
		m.statusBar.IsError = msg.isError
		return m, clearStatusAfter(3 * time.Second)

	case clearStatusMsg:
		m.statusBar.Message = ""
		m.statusBar.IsError = false
		return m, nil

	// --- Keyboard ---
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleConfigLoaded initializes the TUI state from the loaded config.
func (m model) handleConfigLoaded(msg configLoadedMsg) (tea.Model, tea.Cmd) {
	m.config = msg.config
	m.rootDir = msg.rootDir
	m.env = msg.config.Environments.Default
	m.environments = msg.config.Environments.Available

	wsNames := m.bridge.WorkspaceNames(msg.config)
	hasRootSecrets := len(msg.config.Secrets) > 0
	m.workspaces = components.NewWorkspaceList(wsNames, hasRootSecrets)

	// Try to authenticate with cached token (non-blocking)
	cmd := m.tryAuth()

	// Load data for the first workspace
	selected := m.workspaces.Selected()
	if selected != "" {
		return m, tea.Batch(
			cmd,
			loadWorkspaceDataCmd(m.bridge, m.config, m.rootDir, selected, m.env),
		)
	}

	return m, cmd
}

// handleWorkspaceSelected triggers data loading for the newly selected workspace.
func (m model) handleWorkspaceSelected(msg workspaceSelectedMsg) (tea.Model, tea.Cmd) {
	return m, loadWorkspaceDataCmd(m.bridge, m.config, m.rootDir, msg.name, m.env)
}

// handleWorkspaceDataLoaded populates the secret table with merged data.
func (m model) handleWorkspaceDataLoaded(msg workspaceDataLoadedMsg) (tea.Model, tea.Cmd) {
	m.secrets.SetSecrets(msg.secrets, m.env)
	return m, nil
}

// handleEnvChanged switches to a new environment and reloads workspace data.
func (m model) handleEnvChanged(msg envChangedMsg) (tea.Model, tea.Cmd) {
	m.env = msg.env
	m.activePopup = popupNone

	selected := m.workspaces.Selected()
	if selected != "" {
		return m, loadWorkspaceDataCmd(m.bridge, m.config, m.rootDir, selected, m.env)
	}
	return m, nil
}

// handleKey dispatches keyboard events based on current state.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Force quit always works
	if key.Matches(msg, keys.ForceQuit) {
		return m, tea.Quit
	}

	// Delegate to popup handler if a popup is open
	if m.activePopup != popupNone {
		return m.handlePopupKey(msg)
	}

	// Filter mode
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	// Main view key handling
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Tab):
		if m.focus == focusWorkspaces {
			m.focus = focusSecrets
			m.workspaces.Focused = false
			m.secrets.Focused = true
		} else {
			m.focus = focusWorkspaces
			m.workspaces.Focused = true
			m.secrets.Focused = false
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		return m.handleNavUp()

	case key.Matches(msg, keys.Down):
		return m.handleNavDown()

	case key.Matches(msg, keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, keys.Filter):
		m.filtering = true
		m.filterText = ""
		return m, nil

	case key.Matches(msg, keys.Env):
		m.activePopup = popupEnvPicker
		m.envPickerCursor = 0
		for i, env := range m.environments {
			if env == m.env {
				m.envPickerCursor = i
				break
			}
		}
		return m, nil

	case key.Matches(msg, keys.Help):
		m.activePopup = popupHelp
		return m, nil

	case key.Matches(msg, keys.Copy):
		return m.handleCopy()

	case key.Matches(msg, keys.Add):
		return m.handleAdd()

	case key.Matches(msg, keys.Edit):
		return m.handleEdit()

	case key.Matches(msg, keys.Delete):
		return m.handleDelete()
	}

	return m, nil
}

// handleNavUp moves the cursor up in the focused pane.
func (m model) handleNavUp() (tea.Model, tea.Cmd) {
	if m.focus == focusWorkspaces {
		prev := m.workspaces.Selected()
		m.workspaces.MoveUp()
		if m.workspaces.Selected() != prev {
			return m, func() tea.Msg {
				return workspaceSelectedMsg{name: m.workspaces.Selected()}
			}
		}
	} else {
		m.secrets.MoveUp()
	}
	return m, nil
}

// handleNavDown moves the cursor down in the focused pane.
func (m model) handleNavDown() (tea.Model, tea.Cmd) {
	if m.focus == focusWorkspaces {
		prev := m.workspaces.Selected()
		m.workspaces.MoveDown()
		if m.workspaces.Selected() != prev {
			return m, func() tea.Msg {
				return workspaceSelectedMsg{name: m.workspaces.Selected()}
			}
		}
	} else {
		m.secrets.MoveDown()
	}
	return m, nil
}

// handleEnter opens the detail popup for the selected secret.
func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.focus != focusSecrets {
		return m, nil
	}

	selected := m.secrets.Selected()
	if selected == nil {
		return m, nil
	}

	m.activePopup = popupDetail
	m.detailEnvVar = selected.EnvVar
	m.detailPath = selected.VaultPath
	m.detailValue = ""
	m.detailError = ""
	m.detailLoading = true

	return m, resolveSecretCmd(m.bridge, m.vaultClient, m.config, selected.EnvVar, selected.RawPath, m.env)
}

// handleCopy copies the resolved value to clipboard.
func (m model) handleCopy() (tea.Model, tea.Cmd) {
	if m.activePopup == popupDetail && m.detailValue != "" {
		if err := clipboard.WriteAll(m.detailValue); err != nil {
			m.statusBar.Message = "Copy failed: " + err.Error()
			m.statusBar.IsError = true
		} else {
			m.statusBar.Message = "Copied to clipboard"
			m.statusBar.IsError = false
		}
		return m, clearStatusAfter(2 * time.Second)
	}
	return m, nil
}

// handleAdd opens the mapping form for adding a new mapping.
func (m model) handleAdd() (tea.Model, tea.Cmd) {
	if m.vaultClient == nil {
		// Vault browser needs auth — but the form itself doesn't
		m.activePopup = popupMappingForm
		m.mappingFormEnvVar = ""
		m.mappingFormPath = "${env}/"
		m.mappingFormTarget = 0
		m.mappingFormField = 0
		m.mappingFormIsEdit = false
		m.mappingFormOldEnvVar = ""
		return m, nil
	}

	// If we have vault access, open the vault browser first
	m.activePopup = popupVaultBrowser
	m.vaultBrowserPath = ""
	m.vaultBrowserLoading = true
	m.vaultBrowserCursor = 0
	return m, listVaultKeysCmd(m.bridge, m.vaultClient, "")
}

// handleEdit opens the mapping form for editing the selected secret.
func (m model) handleEdit() (tea.Model, tea.Cmd) {
	if m.focus != focusSecrets {
		return m, nil
	}

	selected := m.secrets.Selected()
	if selected == nil {
		return m, nil
	}

	m.activePopup = popupMappingForm
	m.mappingFormEnvVar = selected.EnvVar
	m.mappingFormPath = selected.RawPath
	m.mappingFormTarget = m.findTargetIndex(selected.EnvVar)
	m.mappingFormField = 0
	m.mappingFormIsEdit = true
	m.mappingFormOldEnvVar = selected.EnvVar
	return m, nil
}

// handleDelete opens the delete confirmation for the selected secret.
func (m model) handleDelete() (tea.Model, tea.Cmd) {
	if m.focus != focusSecrets {
		return m, nil
	}

	selected := m.secrets.Selected()
	if selected == nil {
		return m, nil
	}

	workspace := m.workspaces.Selected()
	source := m.bridge.SecretSource(m.config, m.rootDir, workspace, selected.EnvVar)
	if source == "" {
		m.statusBar.Message = "Cannot determine source file for this secret"
		m.statusBar.IsError = true
		return m, clearStatusAfter(3 * time.Second)
	}

	m.activePopup = popupConfirm
	m.confirmEnvVar = selected.EnvVar
	m.confirmFile = source
	m.confirmCursor = 0
	return m, nil
}

// handleFilterKey handles keyboard input while in filter mode.
func (m model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.filtering = false
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.filtering = false
		return m, nil

	case msg.Type == tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.secrets.ApplyFilter(m.filterText)
		}
		return m, nil

	case msg.Type == tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.secrets.ApplyFilter(m.filterText)
		return m, nil
	}

	return m, nil
}

// handlePopupKey dispatches keyboard events for the currently active popup.
func (m model) handlePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Escape) {
		m.activePopup = popupNone
		return m, nil
	}

	switch m.activePopup {
	case popupHelp:
		return m, nil // Esc handled above

	case popupEnvPicker:
		return m.handleEnvPickerKey(msg)

	case popupDetail:
		return m.handleDetailKey(msg)

	case popupVaultBrowser:
		return m.handleVaultBrowserKey(msg)

	case popupMappingForm:
		return m.handleMappingFormKey(msg)

	case popupConfirm:
		return m.handleConfirmKey(msg)
	}

	return m, nil
}

// handleEnvPickerKey handles keys within the environment picker popup.
func (m model) handleEnvPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.envPickerCursor > 0 {
			m.envPickerCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.envPickerCursor < len(m.environments)-1 {
			m.envPickerCursor++
		}
	case msg.Type == tea.KeyEnter:
		if m.envPickerCursor >= 0 && m.envPickerCursor < len(m.environments) {
			return m, func() tea.Msg {
				return envChangedMsg{env: m.environments[m.envPickerCursor]}
			}
		}
	}
	return m, nil
}

// handleDetailKey handles keys within the secret detail popup.
func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Copy) {
		return m.handleCopy()
	}
	return m, nil
}

// handleVaultBrowserKey handles keys within the Vault tree browser popup.
func (m model) handleVaultBrowserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.vaultBrowserCursor > 0 {
			m.vaultBrowserCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.vaultBrowserCursor < len(m.vaultBrowserEntries)-1 {
			m.vaultBrowserCursor++
		}
	case msg.Type == tea.KeyEnter:
		if m.vaultBrowserCursor < len(m.vaultBrowserEntries) {
			entry := m.vaultBrowserEntries[m.vaultBrowserCursor]
			if entry.IsDir {
				newPath := m.vaultBrowserPath + entry.Name
				m.vaultBrowserPath = newPath
				m.vaultBrowserLoading = true
				m.vaultBrowserCursor = 0
				return m, listVaultKeysCmd(m.bridge, m.vaultClient, newPath)
			}
			// Selected a leaf key — open mapping form with this path
			m.activePopup = popupMappingForm
			m.mappingFormPath = "${env}/" + m.vaultBrowserPath + entry.Name
			m.mappingFormEnvVar = suggestEnvVar(entry.Name)
			m.mappingFormTarget = 0
			m.mappingFormField = 1 // focus on env var
			m.mappingFormIsEdit = false
			m.mappingFormOldEnvVar = ""
			return m, nil
		}
	case key.Matches(msg, keys.Backspace):
		return m.vaultBrowserGoUp()
	}
	return m, nil
}

// vaultBrowserGoUp navigates up one directory in the Vault browser.
func (m model) vaultBrowserGoUp() (tea.Model, tea.Cmd) {
	if m.vaultBrowserPath == "" {
		return m, nil
	}

	// Remove trailing slash, then find last slash
	trimmed := m.vaultBrowserPath
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '/' {
		trimmed = trimmed[:len(trimmed)-1]
	}

	lastSlash := -1
	for i := len(trimmed) - 1; i >= 0; i-- {
		if trimmed[i] == '/' {
			lastSlash = i
			break
		}
	}

	var newPath string
	if lastSlash >= 0 {
		newPath = trimmed[:lastSlash+1]
	} else {
		newPath = ""
	}

	m.vaultBrowserPath = newPath
	m.vaultBrowserLoading = true
	m.vaultBrowserCursor = 0
	return m, listVaultKeysCmd(m.bridge, m.vaultClient, newPath)
}

// handleMappingFormKey handles keys within the add/edit mapping form.
func (m model) handleMappingFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyTab:
		m.mappingFormField = (m.mappingFormField + 1) % 3
		return m, nil

	case msg.Type == tea.KeyEnter:
		return m.saveMappingForm()

	case msg.Type == tea.KeyBackspace:
		switch m.mappingFormField {
		case 0: // vault path
			if len(m.mappingFormPath) > 0 {
				m.mappingFormPath = m.mappingFormPath[:len(m.mappingFormPath)-1]
			}
		case 1: // env var
			if len(m.mappingFormEnvVar) > 0 {
				m.mappingFormEnvVar = m.mappingFormEnvVar[:len(m.mappingFormEnvVar)-1]
			}
		case 2: // target file — cycle backwards
			targets := m.bridge.WorkspaceFiles(m.config, m.rootDir)
			if len(targets) > 0 {
				m.mappingFormTarget = (m.mappingFormTarget - 1 + len(targets)) % len(targets)
			}
		}
		return m, nil

	case msg.Type == tea.KeyRunes:
		switch m.mappingFormField {
		case 0:
			m.mappingFormPath += string(msg.Runes)
		case 1:
			m.mappingFormEnvVar += string(msg.Runes)
		case 2:
			// Cycle forward through targets
			targets := m.bridge.WorkspaceFiles(m.config, m.rootDir)
			if len(targets) > 0 {
				m.mappingFormTarget = (m.mappingFormTarget + 1) % len(targets)
			}
		}
		return m, nil
	}

	return m, nil
}

// saveMappingForm validates and saves the current mapping form.
func (m model) saveMappingForm() (tea.Model, tea.Cmd) {
	if m.mappingFormEnvVar == "" || m.mappingFormPath == "" {
		m.statusBar.Message = "Env var and vault path are required"
		m.statusBar.IsError = true
		return m, clearStatusAfter(3 * time.Second)
	}

	targets := m.bridge.WorkspaceFiles(m.config, m.rootDir)
	if m.mappingFormTarget < 0 || m.mappingFormTarget >= len(targets) {
		m.statusBar.Message = "Invalid target file"
		m.statusBar.IsError = true
		return m, clearStatusAfter(3 * time.Second)
	}

	target := targets[m.mappingFormTarget]

	return m, saveMappingCmd(
		m.bridge,
		target.Path,
		m.mappingFormEnvVar,
		m.mappingFormPath,
		m.mappingFormIsEdit,
		m.mappingFormOldEnvVar,
	)
}

// handleConfirmKey handles keys within the delete confirmation popup.
func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up), key.Matches(msg, keys.Down):
		m.confirmCursor = 1 - m.confirmCursor
	case msg.Type == tea.KeyEnter:
		if m.confirmCursor == 1 { // Delete confirmed
			return m, deleteMappingCmd(m.bridge, m.confirmFile, m.confirmEnvVar)
		}
		m.activePopup = popupNone
	}
	return m, nil
}

// --- Command factories ---

// resolveSecretCmd creates a command that resolves a single secret from Vault.
func resolveSecretCmd(b *bridge.Bridge, client *vault.Client, cfg *config.RootConfig, envVar, vaultPath, env string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			// Try to get a client from cached token
			var err error
			client, err = b.Authenticate(cfg)
			if err != nil {
				return secretResolveErrorMsg{envVar: envVar, err: err}
			}
		}

		val, err := b.ResolveSingle(client, envVar, vaultPath, env)
		if err != nil {
			return secretResolveErrorMsg{envVar: envVar, err: err}
		}
		return secretResolvedMsg{envVar: envVar, value: val}
	}
}

// listVaultKeysCmd creates a command that lists Vault keys at a path.
func listVaultKeysCmd(b *bridge.Bridge, client *vault.Client, path string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return vaultListErrorMsg{path: path, err: errNoVaultClient}
		}

		entries, err := b.ListVaultKeys(client, path)
		if err != nil {
			return vaultListErrorMsg{path: path, err: err}
		}

		tuiEntries := make([]VaultEntry, len(entries))
		for i, e := range entries {
			tuiEntries[i] = VaultEntry{Name: e.Name, IsDir: e.IsDir}
		}

		return vaultListResultMsg{path: path, entries: tuiEntries}
	}
}

// saveMappingCmd creates a command that saves a mapping to a vx.toml file.
func saveMappingCmd(b *bridge.Bridge, filePath, envVar, vaultPath string, isEdit bool, oldEnvVar string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if isEdit {
			err = b.EditMapping(filePath, oldEnvVar, envVar, vaultPath)
		} else {
			err = b.AddMapping(filePath, envVar, vaultPath)
		}
		if err != nil {
			return mappingSaveErrorMsg{err: err}
		}
		return mappingSavedMsg{}
	}
}

// deleteMappingCmd creates a command that deletes a mapping from a vx.toml file.
func deleteMappingCmd(b *bridge.Bridge, filePath, envVar string) tea.Cmd {
	return func() tea.Msg {
		err := b.DeleteMapping(filePath, envVar)
		if err != nil {
			return mappingDeleteErrorMsg{err: err}
		}
		return mappingDeletedMsg{}
	}
}

// clearStatusAfter returns a command that sends clearStatusMsg after a delay.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// tryAuth attempts to authenticate with a cached token.
func (m model) tryAuth() tea.Cmd {
	return func() tea.Msg {
		client, err := m.bridge.Authenticate(m.config)
		if err != nil {
			return authFailedMsg{err: err}
		}
		return authSucceededMsg{client: client}
	}
}

// findTargetIndex returns the index of the target file for the given env var.
func (m model) findTargetIndex(envVar string) int {
	workspace := m.workspaces.Selected()
	source := m.bridge.SecretSource(m.config, m.rootDir, workspace, envVar)
	if source == "" {
		return 0
	}

	targets := m.bridge.WorkspaceFiles(m.config, m.rootDir)
	for i, t := range targets {
		if t.Path == source {
			return i
		}
	}
	return 0
}

// suggestEnvVar converts a Vault key name to a suggested environment variable
// name (e.g. "api_key" -> "API_KEY", "database-url" -> "DATABASE_URL").
func suggestEnvVar(key string) string {
	result := ""
	for _, c := range key {
		if c == '-' || c == '.' {
			result += "_"
		} else if c >= 'a' && c <= 'z' {
			result += string(c - 32) // uppercase
		} else {
			result += string(c)
		}
	}
	return result
}

// errNoVaultClient is returned when a Vault operation is attempted without auth.
var errNoVaultClient = fmt.Errorf("no Vault client: run `vx login` first")
