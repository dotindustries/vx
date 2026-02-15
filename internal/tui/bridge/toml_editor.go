package bridge

import (
	"bytes"
	"fmt"
	"os"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/creachadair/tomledit/transform"
)

// AddMapping adds a new KEY = "value" line under the [secrets] section of a
// vx.toml file. It preserves all existing comments, formatting, and ordering.
// If the [secrets] section does not exist, it is created.
func (b *Bridge) AddMapping(filePath, envVar, vaultPath string) error {
	doc, err := readTOMLDoc(filePath)
	if err != nil {
		return err
	}

	secretsSection := findSecretsSection(doc)
	if secretsSection == nil {
		secretsSection = createSecretsSection(doc)
	}

	kv := &parser.KeyValue{
		Name:  parser.Key{envVar},
		Value: parser.MustValue(fmt.Sprintf("%q", vaultPath)),
	}

	transform.InsertMapping(secretsSection, kv, false)

	return writeTOMLDoc(filePath, doc)
}

// EditMapping updates an existing mapping in a vx.toml file. If oldEnvVar
// differs from newEnvVar, the key is renamed and the value is updated.
func (b *Bridge) EditMapping(filePath, oldEnvVar, newEnvVar, newPath string) error {
	doc, err := readTOMLDoc(filePath)
	if err != nil {
		return err
	}

	// Find the existing entry
	entry := doc.First("secrets", oldEnvVar)
	if entry == nil {
		return fmt.Errorf("secret %q not found in [secrets] of %s", oldEnvVar, filePath)
	}

	if oldEnvVar == newEnvVar {
		// Same key name — just update the value
		entry.KeyValue.Value = parser.MustValue(fmt.Sprintf("%q", newPath))
	} else {
		// Key name changed — remove old, add new
		section := findSecretsSection(doc)
		if section == nil {
			return fmt.Errorf("no [secrets] section found in %s", filePath)
		}

		entry.Remove()

		kv := &parser.KeyValue{
			Name:  parser.Key{newEnvVar},
			Value: parser.MustValue(fmt.Sprintf("%q", newPath)),
		}
		transform.InsertMapping(section, kv, false)
	}

	return writeTOMLDoc(filePath, doc)
}

// DeleteMapping removes a mapping from the [secrets] section of a vx.toml file.
func (b *Bridge) DeleteMapping(filePath, envVar string) error {
	doc, err := readTOMLDoc(filePath)
	if err != nil {
		return err
	}

	entry := doc.First("secrets", envVar)
	if entry == nil {
		return fmt.Errorf("secret %q not found in [secrets] of %s", envVar, filePath)
	}

	if !entry.Remove() {
		return fmt.Errorf("failed to remove secret %q from %s", envVar, filePath)
	}

	return writeTOMLDoc(filePath, doc)
}

// readTOMLDoc reads and parses a TOML file into a document tree.
func readTOMLDoc(filePath string) (*tomledit.Document, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer f.Close()

	doc, err := tomledit.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing TOML in %s: %w", filePath, err)
	}

	return doc, nil
}

// writeTOMLDoc writes the TOML document tree back to disk, preserving comments
// and formatting as much as possible.
func writeTOMLDoc(filePath string, doc *tomledit.Document) error {
	var buf bytes.Buffer
	var fmtr tomledit.Formatter
	if err := fmtr.Format(&buf, doc); err != nil {
		return fmt.Errorf("formatting TOML: %w", err)
	}

	// Preserve original file permissions
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", filePath, err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("writing %s: %w", filePath, err)
	}

	return nil
}

// findSecretsSection returns the [secrets] section from the document, or nil.
func findSecretsSection(doc *tomledit.Document) *tomledit.Section {
	entries := doc.Find("secrets")
	for _, e := range entries {
		if e.IsSection() {
			return e.Section
		}
	}
	return nil
}

// createSecretsSection adds a new [secrets] section to the document and
// returns it.
func createSecretsSection(doc *tomledit.Document) *tomledit.Section {
	heading := &parser.Heading{
		Name: parser.Key{"secrets"},
	}

	section := &tomledit.Section{
		Heading: heading,
	}

	doc.Sections = append(doc.Sections, section)
	return section
}

