package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Run executes a child process with injected environment variables.
// Provided env vars are merged with the current process environment;
// provided values override existing ones. Stdin, Stdout, and Stderr are
// inherited from the parent process. The returned error preserves the
// child's exit code when available.
func Run(ctx context.Context, command []string, env map[string]string) error {
	if len(command) == 0 {
		return fmt.Errorf("command must not be empty")
	}

	merged := mergeEnv(os.Environ(), env)

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = merged
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command %q: %w", command[0], err)
	}

	cleanup := ForwardSignals(ctx, cmd.Process)
	defer cleanup()

	return cmd.Wait()
}

// ExitCode extracts the exit code from an error returned by Run.
// Returns 0 if err is nil. Returns the process exit code if err is an
// *exec.ExitError. Returns 1 for all other error types.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 1
}

// mergeEnv combines the current process environment with additional
// env vars. Additional values override existing ones with the same key.
// Neither input slice nor map is mutated.
func mergeEnv(current []string, additional map[string]string) []string {
	envMap := parseEnvSlice(current)

	for k, v := range additional {
		envMap[k] = v
	}

	return formatEnvMap(envMap)
}

// parseEnvSlice converts a slice of "KEY=VALUE" strings into a map.
func parseEnvSlice(envSlice []string) map[string]string {
	result := make(map[string]string, len(envSlice))

	for _, entry := range envSlice {
		key, value := splitEnvEntry(entry)
		if key != "" {
			result[key] = value
		}
	}

	return result
}

// splitEnvEntry splits a "KEY=VALUE" string into key and value.
// If there is no "=" separator, returns the full string as key with
// an empty value.
func splitEnvEntry(entry string) (string, string) {
	for i := range entry {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:]
		}
	}

	return entry, ""
}

// formatEnvMap converts a map back into a slice of "KEY=VALUE" strings.
func formatEnvMap(envMap map[string]string) []string {
	result := make([]string, 0, len(envMap))

	for k, v := range envMap {
		result = append(result, k+"="+v)
	}

	return result
}
