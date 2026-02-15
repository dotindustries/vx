package components

import (
	"testing"
)

func TestNewWorkspaceList(t *testing.T) {
	names := []string{"web", "api", "db"}
	wl := NewWorkspaceList(names, true)

	if wl.Len() != 4 { // 3 + [root]
		t.Errorf("expected 4 items (3 + root), got %d", wl.Len())
	}

	if wl.Selected() != "web" {
		t.Errorf("expected initial selection 'web', got %q", wl.Selected())
	}
}

func TestWorkspaceList_Navigation(t *testing.T) {
	names := []string{"web", "api", "db"}
	wl := NewWorkspaceList(names, true)

	wl.MoveDown()
	if wl.Selected() != "api" {
		t.Errorf("expected 'api' after MoveDown, got %q", wl.Selected())
	}

	wl.MoveDown()
	if wl.Selected() != "db" {
		t.Errorf("expected 'db' after second MoveDown, got %q", wl.Selected())
	}

	wl.MoveDown()
	if wl.Selected() != "[root]" {
		t.Errorf("expected '[root]' at end, got %q", wl.Selected())
	}

	// Should not go past end
	wl.MoveDown()
	if wl.Selected() != "[root]" {
		t.Errorf("expected to stay at '[root]', got %q", wl.Selected())
	}

	wl.MoveUp()
	if wl.Selected() != "db" {
		t.Errorf("expected 'db' after MoveUp, got %q", wl.Selected())
	}

	// All the way up
	wl.MoveUp()
	wl.MoveUp()
	wl.MoveUp()
	if wl.Selected() != "web" {
		t.Errorf("expected to stay at 'web', got %q", wl.Selected())
	}
}

func TestWorkspaceList_NoRoot(t *testing.T) {
	names := []string{"web", "api"}
	wl := NewWorkspaceList(names, false)

	if wl.Len() != 2 {
		t.Errorf("expected 2 items (no root), got %d", wl.Len())
	}

	wl.MoveDown()
	if wl.Selected() != "api" {
		t.Errorf("expected 'api', got %q", wl.Selected())
	}

	// Should not go to [root]
	wl.MoveDown()
	if wl.Selected() != "api" {
		t.Errorf("expected to stay at 'api', got %q", wl.Selected())
	}
}

func TestWorkspaceList_Focus(t *testing.T) {
	names := []string{"web"}
	wl := NewWorkspaceList(names, false)

	if !wl.Focused {
		t.Error("expected initial Focused=true")
	}

	wl.Focused = false
	if wl.Focused {
		t.Error("expected Focused=false after setting")
	}
}

func TestWorkspaceList_Empty(t *testing.T) {
	wl := NewWorkspaceList([]string{}, false)

	if wl.Len() != 0 {
		t.Errorf("expected 0 items, got %d", wl.Len())
	}

	if wl.Selected() != "" {
		t.Errorf("expected empty selection, got %q", wl.Selected())
	}
}
