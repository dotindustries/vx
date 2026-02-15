package components

import (
	"testing"
)

func TestNewSecretTable(t *testing.T) {
	secrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
		"API_KEY":      "${env}/api/key",
		"AUTH_SECRET":  "shared/auth/secret",
	}

	table := NewSecretTable(secrets, "dev")

	if table.Len() != 3 {
		t.Errorf("expected 3 rows, got %d", table.Len())
	}

	// Should be sorted alphabetically by env var name
	if table.Rows[0].EnvVar != "API_KEY" {
		t.Errorf("expected first row to be API_KEY, got %s", table.Rows[0].EnvVar)
	}
	if table.Rows[1].EnvVar != "AUTH_SECRET" {
		t.Errorf("expected second row to be AUTH_SECRET, got %s", table.Rows[1].EnvVar)
	}
	if table.Rows[2].EnvVar != "DATABASE_URL" {
		t.Errorf("expected third row to be DATABASE_URL, got %s", table.Rows[2].EnvVar)
	}
}

func TestSecretTable_Interpolation(t *testing.T) {
	secrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
	}

	table := NewSecretTable(secrets, "staging")

	row := table.Rows[0]
	if row.VaultPath != "staging/database/url" {
		t.Errorf("expected interpolated path 'staging/database/url', got %q", row.VaultPath)
	}
	if row.RawPath != "${env}/database/url" {
		t.Errorf("expected raw path '${env}/database/url', got %q", row.RawPath)
	}
}

func TestSecretTable_ApplyFilter(t *testing.T) {
	secrets := map[string]string{
		"DATABASE_URL":       "${env}/database/url",
		"DATABASE_PASSWORD":  "${env}/database/password",
		"API_KEY":            "${env}/api/key",
		"STRIPE_SECRET_KEY":  "${env}/stripe/secret_key",
	}

	table := NewSecretTable(secrets, "dev")

	// Filter by "database" should match 2 rows
	table.ApplyFilter("database")
	if table.Len() != 2 {
		t.Errorf("expected 2 filtered rows, got %d", table.Len())
	}

	// Clear filter
	table.ApplyFilter("")
	if table.Len() != 4 {
		t.Errorf("expected 4 rows after clearing filter, got %d", table.Len())
	}

	// Case-insensitive filter
	table.ApplyFilter("STRIPE")
	if table.Len() != 1 {
		t.Errorf("expected 1 row for STRIPE filter, got %d", table.Len())
	}

	// Filter by vault path
	table.ApplyFilter("api")
	if table.Len() != 1 {
		t.Errorf("expected 1 row for 'api' path filter, got %d", table.Len())
	}
}

func TestSecretTable_Navigation(t *testing.T) {
	secrets := map[string]string{
		"A_KEY": "a/key",
		"B_KEY": "b/key",
		"C_KEY": "c/key",
	}

	table := NewSecretTable(secrets, "dev")

	if table.Cursor != 0 {
		t.Errorf("initial cursor should be 0, got %d", table.Cursor)
	}

	table.MoveDown()
	if table.Cursor != 1 {
		t.Errorf("cursor should be 1 after MoveDown, got %d", table.Cursor)
	}

	table.MoveDown()
	if table.Cursor != 2 {
		t.Errorf("cursor should be 2 after second MoveDown, got %d", table.Cursor)
	}

	// Should not go past end
	table.MoveDown()
	if table.Cursor != 2 {
		t.Errorf("cursor should stay at 2 at end, got %d", table.Cursor)
	}

	table.MoveUp()
	if table.Cursor != 1 {
		t.Errorf("cursor should be 1 after MoveUp, got %d", table.Cursor)
	}

	// Should not go below 0
	table.MoveUp()
	table.MoveUp()
	if table.Cursor != 0 {
		t.Errorf("cursor should stay at 0 at start, got %d", table.Cursor)
	}
}

func TestSecretTable_Selected(t *testing.T) {
	secrets := map[string]string{
		"A_KEY": "a/key",
		"B_KEY": "b/key",
	}

	table := NewSecretTable(secrets, "dev")

	selected := table.Selected()
	if selected == nil {
		t.Fatal("expected non-nil selected row")
	}
	if selected.EnvVar != "A_KEY" {
		t.Errorf("expected A_KEY selected, got %s", selected.EnvVar)
	}

	table.MoveDown()
	selected = table.Selected()
	if selected.EnvVar != "B_KEY" {
		t.Errorf("expected B_KEY selected, got %s", selected.EnvVar)
	}
}

func TestSecretTable_EmptyTable(t *testing.T) {
	table := NewSecretTable(map[string]string{}, "dev")

	if table.Len() != 0 {
		t.Errorf("expected 0 rows, got %d", table.Len())
	}

	if table.Selected() != nil {
		t.Error("expected nil selected on empty table")
	}
}

func TestSecretTable_FilterResetsCursor(t *testing.T) {
	secrets := map[string]string{
		"A_KEY": "a/key",
		"B_KEY": "b/key",
		"C_KEY": "c/key",
	}

	table := NewSecretTable(secrets, "dev")
	table.MoveDown()
	table.MoveDown()
	if table.Cursor != 2 {
		t.Fatalf("expected cursor at 2, got %d", table.Cursor)
	}

	// Filter down to 1 result — cursor should clamp
	table.ApplyFilter("A_KEY")
	if table.Cursor > table.Len()-1 {
		t.Errorf("cursor %d exceeds filtered length %d", table.Cursor, table.Len())
	}
}
