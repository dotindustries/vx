package exec

import (
	"context"
	"os/exec"
	"testing"
)

func TestRun_echoCommand(t *testing.T) {
	ctx := context.Background()

	err := Run(ctx, []string{"echo", "hello"}, nil)
	if err != nil {
		t.Fatalf("Run(echo hello) returned unexpected error: %v", err)
	}
}

func TestRun_envInjection(t *testing.T) {
	ctx := context.Background()

	env := map[string]string{
		"VX_TEST_VAR": "injected_value",
	}

	// Use env command to print a specific variable; sh -c reads it.
	err := Run(ctx, []string{"sh", "-c", "test \"$VX_TEST_VAR\" = \"injected_value\""}, env)
	if err != nil {
		t.Fatalf("Run() with env injection failed: %v", err)
	}
}

func TestRun_exitCodePropagation(t *testing.T) {
	ctx := context.Background()

	err := Run(ctx, []string{"sh", "-c", "exit 42"}, nil)
	if err == nil {
		t.Fatal("Run() expected error for non-zero exit code, got nil")
	}

	code := ExitCode(err)
	if code != 42 {
		t.Errorf("ExitCode() = %d, want 42", code)
	}
}

func TestRun_emptyCommand(t *testing.T) {
	ctx := context.Background()

	err := Run(ctx, []string{}, nil)
	if err == nil {
		t.Fatal("Run() expected error for empty command, got nil")
	}
}

func TestExitCode_nilError(t *testing.T) {
	code := ExitCode(nil)
	if code != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", code)
	}
}

func TestExitCode_exitError(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 7")
	err := cmd.Run()

	code := ExitCode(err)
	if code != 7 {
		t.Errorf("ExitCode() = %d, want 7", code)
	}
}

func TestExitCode_nonExitError(t *testing.T) {
	code := ExitCode(exec.ErrNotFound)
	if code != 1 {
		t.Errorf("ExitCode(ErrNotFound) = %d, want 1", code)
	}
}

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		name       string
		current    []string
		additional map[string]string
		wantKey    string
		wantValue  string
	}{
		{
			name:       "adds new variable",
			current:    []string{"EXISTING=value"},
			additional: map[string]string{"NEW_VAR": "new_value"},
			wantKey:    "NEW_VAR",
			wantValue:  "new_value",
		},
		{
			name:       "overrides existing variable",
			current:    []string{"MY_VAR=old"},
			additional: map[string]string{"MY_VAR": "new"},
			wantKey:    "MY_VAR",
			wantValue:  "new",
		},
		{
			name:       "nil additional map",
			current:    []string{"KEEP=this"},
			additional: nil,
			wantKey:    "KEEP",
			wantValue:  "this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeEnv(tt.current, tt.additional)
			found := findEnvValue(result, tt.wantKey)
			if found != tt.wantValue {
				t.Errorf("mergeEnv() %s=%q, want %q", tt.wantKey, found, tt.wantValue)
			}
		})
	}
}

func TestSplitEnvEntry(t *testing.T) {
	tests := []struct {
		entry     string
		wantKey   string
		wantValue string
	}{
		{"KEY=VALUE", "KEY", "VALUE"},
		{"KEY=", "KEY", ""},
		{"KEY=VAL=UE", "KEY", "VAL=UE"},
		{"NOEQUALS", "NOEQUALS", ""},
	}

	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			key, value := splitEnvEntry(tt.entry)
			if key != tt.wantKey || value != tt.wantValue {
				t.Errorf("splitEnvEntry(%q) = (%q, %q), want (%q, %q)",
					tt.entry, key, value, tt.wantKey, tt.wantValue)
			}
		})
	}
}

// findEnvValue searches a slice of "KEY=VALUE" entries for the given key.
func findEnvValue(env []string, key string) string {
	for _, entry := range env {
		k, v := splitEnvEntry(entry)
		if k == key {
			return v
		}
	}

	return ""
}
