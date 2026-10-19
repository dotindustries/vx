package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dot.industries/vx/internal/config"
)

func setEnvFileFlags(t *testing.T, files []string, noEnvLocal bool) {
	t.Helper()

	prevFiles, prevNoEnvLocal := flagEnvFiles, flagNoEnvLocal
	flagEnvFiles, flagNoEnvLocal = files, noEnvLocal
	t.Cleanup(func() {
		flagEnvFiles, flagNoEnvLocal = prevFiles, prevNoEnvLocal
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestLoadEnvFileLayers_noFiles(t *testing.T) {
	setEnvFileFlags(t, nil, false)
	t.Setenv(noEnvLocalEnvVar, "")

	layers, err := loadEnvFileLayers(&config.RootConfig{}, t.TempDir())
	if err != nil {
		t.Fatalf("loadEnvFileLayers() error = %v", err)
	}
	if len(layers) != 0 {
		t.Errorf("loadEnvFileLayers() = %v, want no layers", layers)
	}
}

func TestLoadEnvFileLayers_precedence(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv(noEnvLocalEnvVar, "")

	writeFile(t, filepath.Join(root, ".env.local"), "DATABASE_URL=from-local\nLOCAL_ONLY=1\n")
	writeFile(t, filepath.Join(root, "first.env"), "DATABASE_URL=from-first\nFIRST_ONLY=1\n")
	writeFile(t, filepath.Join(root, "second.env"), "DATABASE_URL=from-second\n")
	setEnvFileFlags(t, []string{"first.env", "second.env"}, false)

	layers, err := loadEnvFileLayers(&config.RootConfig{}, root)
	if err != nil {
		t.Fatalf("loadEnvFileLayers() error = %v", err)
	}

	gotSources := make([]string, 0, len(layers))
	for _, l := range layers {
		gotSources = append(gotSources, l.Source)
	}
	if strings.Join(gotSources, ",") != ".env.local,first.env,second.env" {
		t.Errorf("layer sources = %v, want [.env.local first.env second.env]", gotSources)
	}

	values, sources := flattenEnvFileLayers(layers)

	wantValues := map[string]string{"DATABASE_URL": "from-second", "LOCAL_ONLY": "1", "FIRST_ONLY": "1"}
	wantSources := map[string]string{"DATABASE_URL": "second.env", "LOCAL_ONLY": ".env.local", "FIRST_ONLY": "first.env"}
	for k, want := range wantValues {
		if values[k] != want {
			t.Errorf("values[%s] = %q, want %q", k, values[k], want)
		}
		if sources[k] != wantSources[k] {
			t.Errorf("sources[%s] = %q, want %q", k, sources[k], wantSources[k])
		}
	}
	if len(values) != len(wantValues) {
		t.Errorf("values = %v, want %v", values, wantValues)
	}
}

func TestLoadEnvFileLayers_localDisplayedRelativeToCwd(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "packages", "web")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	t.Setenv(noEnvLocalEnvVar, "")
	setEnvFileFlags(t, nil, false)

	writeFile(t, filepath.Join(root, ".env.local"), "FOO=bar\n")

	layers, err := loadEnvFileLayers(&config.RootConfig{}, root)
	if err != nil {
		t.Fatalf("loadEnvFileLayers() error = %v", err)
	}

	want := filepath.Join("..", "..", ".env.local")
	if len(layers) != 1 || layers[0].Source != want {
		t.Errorf("layers = %v, want single layer with source %q", layers, want)
	}
}

func TestLoadEnvFileLayers_missingExplicitFileFails(t *testing.T) {
	t.Setenv(noEnvLocalEnvVar, "")
	setEnvFileFlags(t, []string{filepath.Join(t.TempDir(), "missing.env")}, false)

	_, err := loadEnvFileLayers(&config.RootConfig{}, t.TempDir())
	if err == nil {
		t.Fatal("loadEnvFileLayers() expected error for missing --env-file")
	}
}

func TestLoadEnvFileLayers_invalidLocalFileFails(t *testing.T) {
	root := t.TempDir()
	t.Setenv(noEnvLocalEnvVar, "")
	setEnvFileFlags(t, nil, false)
	writeFile(t, filepath.Join(root, ".env.local"), "not a dotenv line\n")

	_, err := loadEnvFileLayers(&config.RootConfig{}, root)
	if err == nil {
		t.Fatal("loadEnvFileLayers() expected error for invalid .env.local")
	}
}

func TestLoadEnvFileLayers_autoLoadDisabled(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.RootConfig
		noEnvLocal bool
		envVar     string
	}{
		{name: "flag", noEnvLocal: true},
		{name: "env var", envVar: "1"},
		{name: "config", cfg: config.RootConfig{EnvFile: config.EnvFileConfig{AutoLoad: boolPtr(false)}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".env.local"), "FOO=local\n")
			explicit := filepath.Join(root, "explicit.env")
			writeFile(t, explicit, "BAR=explicit\n")

			setEnvFileFlags(t, []string{explicit}, tt.noEnvLocal)
			t.Setenv(noEnvLocalEnvVar, tt.envVar)

			layers, err := loadEnvFileLayers(&tt.cfg, root)
			if err != nil {
				t.Fatalf("loadEnvFileLayers() error = %v", err)
			}

			values, _ := flattenEnvFileLayers(layers)
			if _, ok := values["FOO"]; ok {
				t.Error(".env.local was loaded although auto-loading is disabled")
			}
			if values["BAR"] != "explicit" {
				t.Errorf("BAR = %q, want explicit --env-file to still load", values["BAR"])
			}
		})
	}
}

func TestLocalEnvAutoLoadEnabled_falseyEnvVar(t *testing.T) {
	setEnvFileFlags(t, nil, false)

	for _, v := range []string{"", "0", "false"} {
		t.Setenv(noEnvLocalEnvVar, v)
		if !localEnvAutoLoadEnabled(&config.RootConfig{}) {
			t.Errorf("%s=%q disabled auto-loading, want enabled", noEnvLocalEnvVar, v)
		}
	}
}

func TestWithoutOverriddenSecrets(t *testing.T) {
	merged := &config.MergedConfig{
		Environment: "dev",
		Secrets: map[string]string{
			"DATABASE_URL":   "${env}/database/url",
			"OPENAI_API_KEY": "shared/openai/api_key",
		},
	}

	filtered := withoutOverriddenSecrets(merged, map[string]string{"DATABASE_URL": "x"})

	if _, ok := filtered.Secrets["DATABASE_URL"]; ok {
		t.Error("overridden secret DATABASE_URL should not be resolved from Vault")
	}
	if filtered.Secrets["OPENAI_API_KEY"] != "shared/openai/api_key" {
		t.Error("non-overridden secret OPENAI_API_KEY was dropped")
	}
	if filtered.Environment != "dev" {
		t.Errorf("Environment = %q, want dev", filtered.Environment)
	}
	if len(merged.Secrets) != 2 {
		t.Error("input MergedConfig was mutated")
	}

	if got := withoutOverriddenSecrets(merged, nil); got != merged {
		t.Error("expected the same config back when there are no overrides")
	}
}

func TestBuildEnv_precedence(t *testing.T) {
	env := buildEnv(
		map[string]string{"A": "default", "B": "default", "C": "default"},
		map[string]string{"B": "secret", "C": "secret"},
		map[string]string{"C": "override"},
	)

	want := map[string]string{"A": "default", "B": "secret", "C": "override"}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("%s = %q, want %q", k, env[k], v)
		}
	}
}

func TestPrintTable_marksOverrides(t *testing.T) {
	merged := &config.MergedConfig{
		Secrets: map[string]string{
			"DATABASE_URL":   "${env}/database/url",
			"OPENAI_API_KEY": "shared/openai/api_key",
		},
		Defaults: map[string]string{"NODE_ENV": "development"},
	}
	sources := map[string]string{
		"DATABASE_URL":        ".env.local",
		"NODE_ENV":            ".env.local",
		"DATABASE_AUTH_TOKEN": "branch.env",
	}

	var buf bytes.Buffer
	printTable(&buf, merged, "dev", "", sources)
	out := buf.String()

	for _, want := range []string{
		"dev/database/url  (overridden by .env.local)",
		"= development  (overridden by .env.local)",
		"Env file overrides (3):",
		"DATABASE_AUTH_TOKEN",
		"<- branch.env",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printTable() output missing %q:\n%s", want, out)
		}
	}

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "OPENAI_API_KEY") && strings.Contains(line, "overridden") {
			t.Errorf("OPENAI_API_KEY incorrectly marked as overridden: %q", line)
		}
	}
}

func TestPrintTable_noOverrides(t *testing.T) {
	merged := &config.MergedConfig{
		Secrets:  map[string]string{"DATABASE_URL": "${env}/database/url"},
		Defaults: map[string]string{"NODE_ENV": "development"},
	}

	var buf bytes.Buffer
	printTable(&buf, merged, "dev", "", nil)
	out := buf.String()

	if strings.Contains(out, "overridden") || strings.Contains(out, "Env file overrides") {
		t.Errorf("printTable() mentions overrides without any env file:\n%s", out)
	}
}
