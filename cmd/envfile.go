package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/envfile"
)

const (
	localEnvFileName = ".env.local"
	noEnvLocalEnvVar = "VX_NO_ENV_LOCAL"
)

var (
	flagEnvFiles   []string
	flagNoEnvLocal bool
)

func addEnvFileFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&flagEnvFiles, "env-file", nil,
		"dotenv file whose values override Vault secrets and defaults (repeatable; later files win)")
	cmd.Flags().BoolVar(&flagNoEnvLocal, "no-env-local", false,
		"do not auto-load "+localEnvFileName+" next to the root vx.toml (also "+noEnvLocalEnvVar+"=1)")
}

// envFileLayer is the parsed content of one env file, kept per file so that
// logs and `vx list` can attribute each override to its source.
type envFileLayer struct {
	Source string
	Vars   map[string]string
}

type envFileSource struct {
	path     string
	display  string
	optional bool
}

// loadEnvFileLayers reads the auto-loaded .env.local (when present and
// enabled) followed by every --env-file, in that order, so explicit files win.
func loadEnvFileLayers(cfg *config.RootConfig, rootDir string) ([]envFileLayer, error) {
	sources := make([]envFileSource, 0, len(flagEnvFiles)+1)

	if localEnvAutoLoadEnabled(cfg) {
		path := filepath.Join(rootDir, localEnvFileName)
		sources = append(sources, envFileSource{path: path, display: displayPath(path), optional: true})
	} else {
		log.Debug().Msg("auto-loading of " + localEnvFileName + " is disabled")
	}

	for _, path := range flagEnvFiles {
		sources = append(sources, envFileSource{path: path, display: path})
	}

	layers := make([]envFileLayer, 0, len(sources))
	for _, src := range sources {
		vars, err := envfile.Load(src.path)
		if err != nil {
			if src.optional && errors.Is(err, fs.ErrNotExist) {
				log.Debug().Str("path", src.path).Msg("no local env file found")
				continue
			}
			return nil, fmt.Errorf("loading env file: %w", err)
		}

		layers = append(layers, envFileLayer{Source: src.display, Vars: vars})
	}

	return layers, nil
}

func localEnvAutoLoadEnabled(cfg *config.RootConfig) bool {
	if flagNoEnvLocal || !cfg.EnvFile.AutoLoadEnabled() {
		return false
	}

	disabled, err := strconv.ParseBool(os.Getenv(noEnvLocalEnvVar))
	return err != nil || !disabled
}

// flattenEnvFileLayers maps each overridden key to the source of the layer
// that set it last.
func flattenEnvFileLayers(layers []envFileLayer) (values map[string]string, sources map[string]string) {
	values = make(map[string]string)
	sources = make(map[string]string)

	for _, layer := range layers {
		for k, v := range layer.Vars {
			values[k] = v
			sources[k] = layer.Source
		}
	}

	return values, sources
}

// logEnvFileLayers reports overridden key names only; values may be
// credentials and must never reach the logs.
func logEnvFileLayers(layers []envFileLayer) {
	for _, layer := range layers {
		if len(layer.Vars) == 0 {
			continue
		}

		keys := sortedKeys(layer.Vars)
		log.Info().Msgf("overriding %d %s from %s: %s",
			len(keys), pluralize(len(keys), "key", "keys"), layer.Source, strings.Join(keys, ", "))
	}
}

// withoutOverriddenSecrets drops secrets that an env file replaces so they are
// not fetched from Vault, which also lets local overrides cover paths the
// developer has no Vault access to.
func withoutOverriddenSecrets(merged *config.MergedConfig, overrides map[string]string) *config.MergedConfig {
	if len(overrides) == 0 {
		return merged
	}

	filtered := *merged
	filtered.Secrets = make(map[string]string, len(merged.Secrets))
	for k, v := range merged.Secrets {
		if _, ok := overrides[k]; !ok {
			filtered.Secrets[k] = v
		}
	}

	return &filtered
}

// buildEnv layers defaults, then Vault secrets, then env file overrides.
func buildEnv(defaults, secrets, overrides map[string]string) map[string]string {
	env := make(map[string]string, len(defaults)+len(secrets)+len(overrides))
	for _, layer := range []map[string]string{defaults, secrets, overrides} {
		for k, v := range layer {
			env[k] = v
		}
	}
	return env
}

func displayPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return rel
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
