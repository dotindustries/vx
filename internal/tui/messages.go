package tui

import (
	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/vault"
)

// --- Config loading ---

// configLoadedMsg is sent after the root config is successfully loaded.
type configLoadedMsg struct {
	config  *config.RootConfig
	rootDir string
}

// configErrorMsg is sent when config loading fails.
type configErrorMsg struct{ err error }

// --- Workspace selection ---

// workspaceSelectedMsg signals that the user selected a workspace.
type workspaceSelectedMsg struct {
	name string
}

// workspaceDataLoadedMsg carries the merged config for the selected workspace.
type workspaceDataLoadedMsg struct {
	secrets map[string]string // env var -> vault path template
	source  string           // workspace name or "[root]"
}

// workspaceDataErrorMsg is sent when workspace data loading fails.
type workspaceDataErrorMsg struct{ err error }

// --- Environment ---

// envChangedMsg signals that the user picked a new environment.
type envChangedMsg struct {
	env string
}

// --- Secret resolution (Phase 2) ---

// resolveSecretMsg requests on-demand resolution of a single secret.
type resolveSecretMsg struct {
	envVar    string
	vaultPath string
}

// secretResolvedMsg carries the resolved secret value.
type secretResolvedMsg struct {
	envVar string
	value  string
}

// secretResolveErrorMsg is sent when secret resolution fails.
type secretResolveErrorMsg struct {
	envVar string
	err    error
}

// --- Authentication ---

// authRequiredMsg signals that Vault auth is needed before an operation.
type authRequiredMsg struct{}

// authSucceededMsg is sent after successful Vault authentication.
type authSucceededMsg struct {
	client *vault.Client
}

// authFailedMsg is sent when Vault authentication fails.
type authFailedMsg struct{ err error }

// --- Vault tree browsing (Phase 3) ---

// VaultEntry represents a key or directory in the Vault KV tree.
type VaultEntry struct {
	Name  string
	IsDir bool
}

// vaultListMsg requests listing keys at a Vault path.
type vaultListMsg struct {
	path string
}

// vaultListResultMsg carries the result of a Vault LIST operation.
type vaultListResultMsg struct {
	path    string
	entries []VaultEntry
}

// vaultListErrorMsg is sent when Vault listing fails.
type vaultListErrorMsg struct {
	path string
	err  error
}

// --- CRUD operations (Phase 3) ---

// saveMappingMsg requests writing a new or updated mapping to a vx.toml file.
type saveMappingMsg struct {
	filePath  string
	envVar    string
	vaultPath string
	isEdit    bool
	oldEnvVar string // only set when isEdit is true and envVar changed
}

// mappingSavedMsg signals that a mapping was successfully saved.
type mappingSavedMsg struct{}

// mappingSaveErrorMsg is sent when saving a mapping fails.
type mappingSaveErrorMsg struct{ err error }

// deleteMappingMsg requests deletion of a mapping from a vx.toml file.
type deleteMappingMsg struct {
	filePath string
	envVar   string
}

// mappingDeletedMsg signals that a mapping was successfully deleted.
type mappingDeletedMsg struct{}

// mappingDeleteErrorMsg is sent when deleting a mapping fails.
type mappingDeleteErrorMsg struct{ err error }

// --- UI state ---

// statusMsg shows a temporary status message in the status bar.
type statusMsg struct {
	text    string
	isError bool
}

// clearStatusMsg clears the status message.
type clearStatusMsg struct{}
