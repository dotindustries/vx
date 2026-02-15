package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DetectWorkspace determines which workspace to use based on detection priority:
//  1. Explicit -w flag value from args
//  2. --cwd <path> argument pattern
//  3. Whether cwd is inside a known workspace directory
//  4. Empty string (load all workspaces)
func DetectWorkspace(args []string, cwd string, workspaces []string) (string, error) {
	if ws := findFlagValue(args, "-w"); ws != "" {
		return ws, nil
	}

	if cwdPath := findFlagValue(args, "--cwd"); cwdPath != "" {
		return matchWorkspaceByPath(cwdPath, workspaces)
	}

	ws, err := matchWorkspaceByCwd(cwd, workspaces)
	if err != nil {
		return "", err
	}

	return ws, nil
}

// ResolveWorkspacePath returns the absolute path to the vx.toml for a given workspace name.
// It searches workspacePaths for a path whose directory name matches the workspace argument.
func ResolveWorkspacePath(rootDir string, workspace string, workspacePaths []string) (string, error) {
	for _, wp := range workspacePaths {
		dir := filepath.Dir(wp)
		dirName := filepath.Base(dir)
		if dirName == workspace {
			return filepath.Join(rootDir, wp), nil
		}
	}

	return "", fmt.Errorf("workspace %q not found in configured workspace paths", workspace)
}

// findFlagValue extracts the value following a flag in the args slice.
func findFlagValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// matchWorkspaceByPath returns the workspace name whose directory contains the given path.
func matchWorkspaceByPath(path string, workspaces []string) (string, error) {
	for _, ws := range workspaces {
		dir := filepath.Dir(ws)
		if strings.HasPrefix(path, dir) {
			return filepath.Base(dir), nil
		}
	}
	return "", nil
}

// matchWorkspaceByCwd checks whether cwd falls inside one of the workspace directories.
func matchWorkspaceByCwd(cwd string, workspaces []string) (string, error) {
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path for cwd %s: %w", cwd, err)
	}

	for _, ws := range workspaces {
		absWsDir, err := filepath.Abs(filepath.Dir(ws))
		if err != nil {
			return "", fmt.Errorf("resolving absolute path for workspace %s: %w", ws, err)
		}

		if strings.HasPrefix(absCwd, absWsDir) {
			return filepath.Base(absWsDir), nil
		}
	}

	return "", nil
}
