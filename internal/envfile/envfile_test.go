package envfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "empty input",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "simple assignments",
			input: "DATABASE_URL=libsql://branch.turso.io\nDATABASE_AUTH_TOKEN=abc123\n",
			want: map[string]string{
				"DATABASE_URL":        "libsql://branch.turso.io",
				"DATABASE_AUTH_TOKEN": "abc123",
			},
		},
		{
			name:  "comments and blank lines",
			input: "# local overrides\n\n  # indented comment\nFOO=bar\n\n",
			want:  map[string]string{"FOO": "bar"},
		},
		{
			name:  "export prefix",
			input: "export FOO=bar",
			want:  map[string]string{"FOO": "bar"},
		},
		{
			name:  "whitespace around separator",
			input: "FOO = bar  ",
			want:  map[string]string{"FOO": "bar"},
		},
		{
			name:  "empty value",
			input: "FOO=\nBAR=",
			want:  map[string]string{"FOO": "", "BAR": ""},
		},
		{
			name:  "empty value with comment",
			input: "FOO= # unset",
			want:  map[string]string{"FOO": ""},
		},
		{
			name:  "inline comment on unquoted value",
			input: "FOO=bar # comment",
			want:  map[string]string{"FOO": "bar"},
		},
		{
			name:  "hash without preceding whitespace is kept",
			input: "URL=https://example.com/#anchor\nPASS=pa#ss",
			want: map[string]string{
				"URL":  "https://example.com/#anchor",
				"PASS": "pa#ss",
			},
		},
		{
			name:  "value containing equals sign",
			input: "DSN=postgres://u:p@h/db?sslmode=disable",
			want:  map[string]string{"DSN": "postgres://u:p@h/db?sslmode=disable"},
		},
		{
			name:  "single quoted value is literal",
			input: `FOO='bar\n # not a comment $HOME'`,
			want:  map[string]string{"FOO": `bar\n # not a comment $HOME`},
		},
		{
			name:  "double quoted value with escapes",
			input: `FOO="line1\nline2\t\"quoted\" \\ \$HOME"`,
			want:  map[string]string{"FOO": "line1\nline2\t\"quoted\" \\ $HOME"},
		},
		{
			name:  "double quoted value with trailing comment",
			input: `FOO="bar # kept" # dropped`,
			want:  map[string]string{"FOO": "bar # kept"},
		},
		{
			name:  "multi-line double quoted value",
			input: "KEY=\"-----BEGIN-----\nabc\n-----END-----\"\nNEXT=1",
			want: map[string]string{
				"KEY":  "-----BEGIN-----\nabc\n-----END-----",
				"NEXT": "1",
			},
		},
		{
			name:  "multi-line single quoted value",
			input: "KEY='a\nb'",
			want:  map[string]string{"KEY": "a\nb"},
		},
		{
			name:  "variable references are not expanded",
			input: "FOO=${BAR}",
			want:  map[string]string{"FOO": "${BAR}"},
		},
		{
			name:  "last declaration wins",
			input: "FOO=first\nFOO=second",
			want:  map[string]string{"FOO": "second"},
		},
		{
			name:  "CRLF line endings",
			input: "FOO=bar\r\nBAZ=qux\r\n",
			want:  map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			assertVars(t, got, tt.want)
		})
	}
}

func TestParse_errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "missing separator", input: "FOO", wantErr: "line 1: expected KEY=VALUE"},
		{name: "invalid key", input: "\n1FOO=bar", wantErr: `line 2: invalid variable name "1FOO"`},
		{name: "key with dash", input: "FOO-BAR=baz", wantErr: "invalid variable name"},
		{name: "empty key", input: "=bar", wantErr: "invalid variable name"},
		{name: "unterminated double quote", input: "FOO=\"bar", wantErr: "unterminated"},
		{name: "unterminated single quote", input: "FOO='bar\nBAZ=1", wantErr: "unterminated"},
		{name: "garbage after quote", input: `FOO="bar" baz`, wantErr: "after closing quote"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("Parse() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(path, []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	assertVars(t, got, map[string]string{"FOO": "bar"})
}

func TestLoad_missingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.env"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() error = %v, want fs.ErrNotExist", err)
	}
}

func TestLoad_parseErrorIncludesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(path, []byte("not valid"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("Load() error = %v, want it to mention %s", err, path)
	}
}

func assertVars(t *testing.T, got, want map[string]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("got %d vars %v, want %d vars %v", len(got), got, len(want), want)
	}
	for k, w := range want {
		g, ok := got[k]
		if !ok {
			t.Errorf("missing key %q", k)
			continue
		}
		if g != w {
			t.Errorf("%s = %q, want %q", k, g, w)
		}
	}
}
