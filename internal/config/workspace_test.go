package config

import (
	"testing"
)

func TestDetectWorkspace_ExplicitFlag(t *testing.T) {
	args := []string{"exec", "-w", "web", "--", "bun", "dev"}
	workspaces := []string{"web/vx.toml", "packages/api/vx.toml"}

	ws, err := DetectWorkspace(args, "/some/path", workspaces)
	if err != nil {
		t.Fatalf("DetectWorkspace() error = %v", err)
	}
	if ws != "web" {
		t.Errorf("DetectWorkspace() = %q, want %q", ws, "web")
	}
}

func TestDetectWorkspace_CwdFlag(t *testing.T) {
	args := []string{"exec", "--cwd", "packages/api/src", "--", "bun", "dev"}
	workspaces := []string{"web/vx.toml", "packages/api/vx.toml"}

	ws, err := DetectWorkspace(args, "/some/path", workspaces)
	if err != nil {
		t.Fatalf("DetectWorkspace() error = %v", err)
	}
	if ws != "api" {
		t.Errorf("DetectWorkspace() = %q, want %q", ws, "api")
	}
}

func TestDetectWorkspace_CwdInsideWorkspace(t *testing.T) {
	args := []string{"exec", "--", "bun", "dev"}
	workspaces := []string{"web/vx.toml", "packages/api/vx.toml"}

	// Use an absolute path that starts with the workspace directory prefix.
	// matchWorkspaceByCwd uses filepath.Abs, so we build paths accordingly.
	cwd := "packages/api/src"

	ws, err := DetectWorkspace(args, cwd, workspaces)
	if err != nil {
		t.Fatalf("DetectWorkspace() error = %v", err)
	}
	if ws != "api" {
		t.Errorf("DetectWorkspace() = %q, want %q", ws, "api")
	}
}

func TestDetectWorkspace_Fallback(t *testing.T) {
	args := []string{"exec", "--", "bun", "dev"}
	workspaces := []string{"web/vx.toml", "packages/api/vx.toml"}

	ws, err := DetectWorkspace(args, "/completely/unrelated/path", workspaces)
	if err != nil {
		t.Fatalf("DetectWorkspace() error = %v", err)
	}
	if ws != "" {
		t.Errorf("DetectWorkspace() = %q, want empty string", ws)
	}
}

func TestDetectWorkspace_WFlagTakesPrecedence(t *testing.T) {
	args := []string{"exec", "-w", "web", "--cwd", "packages/api/src", "--", "bun", "dev"}
	workspaces := []string{"web/vx.toml", "packages/api/vx.toml"}

	ws, err := DetectWorkspace(args, "packages/api/src", workspaces)
	if err != nil {
		t.Fatalf("DetectWorkspace() error = %v", err)
	}
	if ws != "web" {
		t.Errorf("DetectWorkspace() = %q, want %q (should prefer -w over --cwd)", ws, "web")
	}
}

func TestResolveWorkspacePath(t *testing.T) {
	workspacePaths := []string{"web/vx.toml", "packages/api/vx.toml"}

	got, err := ResolveWorkspacePath("/project", "web", workspacePaths)
	if err != nil {
		t.Fatalf("ResolveWorkspacePath() error = %v", err)
	}
	want := "/project/web/vx.toml"
	if got != want {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_NestedWorkspace(t *testing.T) {
	workspacePaths := []string{"web/vx.toml", "packages/api/vx.toml"}

	got, err := ResolveWorkspacePath("/project", "api", workspacePaths)
	if err != nil {
		t.Fatalf("ResolveWorkspacePath() error = %v", err)
	}
	want := "/project/packages/api/vx.toml"
	if got != want {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_NotFound(t *testing.T) {
	workspacePaths := []string{"web/vx.toml", "packages/api/vx.toml"}

	_, err := ResolveWorkspacePath("/project", "nonexistent", workspacePaths)
	if err == nil {
		t.Fatal("ResolveWorkspacePath() expected error for unknown workspace")
	}
}
