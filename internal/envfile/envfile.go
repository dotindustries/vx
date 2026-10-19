package envfile

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return vars, nil
}

// Parse reads dotenv-formatted content and returns the declared variables.
//
// Supported syntax:
//   - KEY=VALUE, optionally prefixed with "export "
//   - blank lines and lines starting with "#"
//   - unquoted values, with trailing " # comment" stripped
//   - single-quoted values, taken literally
//   - double-quoted values, with \n, \r, \t, \", \\ and \$ escapes
//   - quoted values spanning multiple lines
//
// Variable references such as ${OTHER} are not expanded. When a key is
// declared more than once, the last declaration wins.
func Parse(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	vars := make(map[string]string)

	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", lineNo)
		}

		key = strings.TrimSpace(key)
		if !isValidKey(key) {
			return nil, fmt.Errorf("line %d: invalid variable name %q", lineNo, key)
		}

		trimmed := strings.TrimLeft(rest, " \t")

		var value string
		switch {
		case strings.HasPrefix(trimmed, `"`), strings.HasPrefix(trimmed, `'`):
			quote := trimmed[0]
			raw, consumed, err := readQuoted(trimmed[1:], lines[i+1:], quote)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			i += consumed

			if quote == '"' {
				value = unescapeDouble(raw)
			} else {
				value = raw
			}
		default:
			value = stripInlineComment(rest)
		}

		vars[key] = value
	}

	return vars, nil
}

func readQuoted(first string, following []string, quote byte) (string, int, error) {
	var b strings.Builder
	current := first

	for consumed := 0; ; consumed++ {
		if end := findClosingQuote(current, quote); end >= 0 {
			b.WriteString(current[:end])

			trailing := strings.TrimSpace(current[end+1:])
			if trailing != "" && !strings.HasPrefix(trailing, "#") {
				return "", 0, fmt.Errorf("unexpected characters after closing quote: %q", trailing)
			}

			return b.String(), consumed, nil
		}

		if consumed >= len(following) {
			return "", 0, fmt.Errorf("unterminated %c-quoted value", quote)
		}

		b.WriteString(current)
		b.WriteByte('\n')
		current = following[consumed]
	}
}

func findClosingQuote(s string, quote byte) int {
	for i := 0; i < len(s); i++ {
		if quote == '"' && s[i] == '\\' {
			i++
			continue
		}
		if s[i] == quote {
			return i
		}
	}
	return -1
}

func unescapeDouble(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}

		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '"', '\\', '$':
			b.WriteByte(s[i])
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}

	return b.String()
}

// A "#" only starts a comment when preceded by whitespace so that values such
// as "https://host/#anchor" or "pa#ss" survive unquoted.
func stripInlineComment(s string) string {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

func isValidKey(key string) bool {
	if key == "" {
		return false
	}

	for i, c := range key {
		switch {
		case c == '_', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}

	return true
}
