package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/tui/bridge"
	"go.dot.industries/vx/internal/tui/components"
)

func testConfig() *config.RootConfig {
	return &config.RootConfig{
		Vault: config.VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
			AuthRole:   "admin",
			BasePath:   "secret",
		},
		Environments: config.EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "staging", "production"},
		},
		Workspaces: []string{
			"web/vx.toml",
			"packages/api/vx.toml",
		},
		Secrets: map[string]string{
			"SHARED_KEY": "${env}/shared/key",
		},
	}
}

func TestConfigLoadedMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)

	msg := configLoadedMsg{
		config:  testConfig(),
		rootDir: "/tmp/test",
	}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.config == nil {
		t.Fatal("config should be set after configLoadedMsg")
	}

	if mdl.env != "dev" {
		t.Errorf("expected env 'dev', got %q", mdl.env)
	}

	if len(mdl.environments) != 3 {
		t.Errorf("expected 3 environments, got %d", len(mdl.environments))
	}

	if mdl.workspaces.Len() != 3 { // web, api, [root]
		t.Errorf("expected 3 workspace entries (web, api, root), got %d", mdl.workspaces.Len())
	}
}

func TestConfigErrorMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)

	msg := configErrorMsg{err: errNoVaultClient}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.fatalError == "" {
		t.Error("fatalError should be set after configErrorMsg")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.width != 120 {
		t.Errorf("expected width 120, got %d", mdl.width)
	}
	if mdl.height != 40 {
		t.Errorf("expected height 40, got %d", mdl.height)
	}
}

func TestWorkspaceDataLoadedMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"

	secrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
		"API_KEY":      "${env}/api/key",
	}

	msg := workspaceDataLoadedMsg{
		secrets: secrets,
		source:  "web",
	}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.secrets.TotalLen() != 2 {
		t.Errorf("expected 2 secrets, got %d", mdl.secrets.TotalLen())
	}
}

func TestEnvChangedMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.rootDir = "/tmp/test"
	m.env = "dev"
	m.environments = testConfig().Environments.Available
	m.workspaces = testWorkspaceList()

	msg := envChangedMsg{env: "staging"}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.env != "staging" {
		t.Errorf("expected env 'staging', got %q", mdl.env)
	}
	if mdl.activePopup != popupNone {
		t.Error("popup should be closed after env change")
	}
}

func TestKeyQuit(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestKeyTab(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"
	m.focus = focusWorkspaces

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.focus != focusSecrets {
		t.Error("Tab should switch focus to secrets")
	}

	// Tab again should go back
	updated, _ = mdl.Update(msg)
	mdl = updated.(model)

	if mdl.focus != focusWorkspaces {
		t.Error("Tab should switch focus back to workspaces")
	}
}

func TestKeyFilterMode(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"

	// Enter filter mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if !mdl.filtering {
		t.Error("expected filtering mode after '/'")
	}

	// Type characters
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D', 'B'}}
	updated, _ = mdl.Update(msg)
	mdl = updated.(model)

	if mdl.filterText != "DB" {
		t.Errorf("expected filter text 'DB', got %q", mdl.filterText)
	}

	// Escape exits filter mode
	msg = tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = mdl.Update(msg)
	mdl = updated.(model)

	if mdl.filtering {
		t.Error("expected filtering to stop after Esc")
	}
}

func TestKeyHelpPopup(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.activePopup != popupHelp {
		t.Error("expected help popup after '?'")
	}

	// Escape closes popup
	msg = tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = mdl.Update(msg)
	mdl = updated.(model)

	if mdl.activePopup != popupNone {
		t.Error("expected popup closed after Esc")
	}
}

func TestKeyEnvPicker(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.config = testConfig()
	m.env = "dev"
	m.environments = testConfig().Environments.Available

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.activePopup != popupEnvPicker {
		t.Error("expected env picker popup after 'e'")
	}
}

func TestSecretResolvedMsg(t *testing.T) {
	b := bridge.New("", "", "", "", "")
	m := newModel(b)
	m.activePopup = popupDetail
	m.detailLoading = true

	msg := secretResolvedMsg{
		envVar: "DATABASE_URL",
		value:  "postgresql://localhost:5432/mydb",
	}

	updated, _ := m.Update(msg)
	mdl := updated.(model)

	if mdl.detailLoading {
		t.Error("expected loading to stop")
	}
	if mdl.detailValue != "postgresql://localhost:5432/mydb" {
		t.Errorf("expected resolved value, got %q", mdl.detailValue)
	}
}

func TestSuggestEnvVar(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"api_key", "API_KEY"},
		{"database-url", "DATABASE_URL"},
		{"secret", "SECRET"},
		{"api.key", "API_KEY"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
	}

	for _, tt := range tests {
		got := suggestEnvVar(tt.input)
		if got != tt.want {
			t.Errorf("suggestEnvVar(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// testWorkspaceList creates a workspace list for testing.
func testWorkspaceList() components.WorkspaceList {
	return components.NewWorkspaceList([]string{"web", "api"}, true)
}
