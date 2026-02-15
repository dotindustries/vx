package tui

import (
	"fmt"
	"strings"
)

// renderHelpPopup returns the help overlay content.
func (m model) renderHelpPopup() string {
	helpBindings := []struct{ key, desc string }{
		{"j/k or ↑/↓", "Navigate within current pane"},
		{"Tab", "Switch focus between workspaces and secrets"},
		{"e", "Open environment picker"},
		{"/", "Enter filter mode (type to filter secrets)"},
		{"Enter", "View secret detail (resolves from Vault)"},
		{"c", "Copy resolved secret value to clipboard"},
		{"a", "Add new secret mapping"},
		{"r", "Edit selected mapping"},
		{"d", "Delete selected mapping (with confirmation)"},
		{"?", "Toggle this help"},
		{"Esc", "Close popup / exit filter mode"},
		{"q / Ctrl+C", "Quit"},
	}

	var b strings.Builder
	for _, h := range helpBindings {
		key := styleKey.Width(14).Render(h.key)
		desc := styleDesc.Render(h.desc)
		b.WriteString(key + " " + desc + "\n")
	}

	return stylePopup.
		Width(56).
		Render(
			styleTitle.Render("Keyboard Shortcuts") + "\n\n" +
				b.String(),
		)
}

// renderEnvPickerPopup returns the environment picker overlay.
func (m model) renderEnvPickerPopup() string {
	var b strings.Builder
	for i, env := range m.environments {
		prefix := "  "
		style := styleNormal
		if i == m.envPickerCursor {
			prefix = "> "
			style = styleSelected
		}
		if env == m.env {
			b.WriteString(style.Render(prefix + env + " (current)") + "\n")
		} else {
			b.WriteString(style.Render(prefix+env) + "\n")
		}
	}

	return stylePopup.
		Width(40).
		Render(
			styleTitle.Render("Select Environment") + "\n\n" +
				b.String() + "\n" +
				styleMuted.Render("j/k:nav  enter:select  esc:close"),
		)
}

// renderDetailPopup returns the secret detail overlay.
func (m model) renderDetailPopup() string {
	var content string

	if m.detailLoading {
		content = styleMuted.Render("Resolving from Vault...")
	} else if m.detailError != "" {
		content = styleErrorText.Render("Error: " + m.detailError)
	} else if m.detailValue != "" {
		content = styleNormal.Render(m.detailValue)
	} else {
		content = styleMuted.Render("No value resolved")
	}

	envVar := styleKey.Render(m.detailEnvVar)
	path := styleDim.Render(m.detailPath)

	footer := styleMuted.Render("c:copy  esc:close")

	return stylePopup.
		Width(min(m.width-10, 70)).
		Render(
			styleTitle.Render("Secret Detail") + "\n\n" +
				"Env var:  " + envVar + "\n" +
				"Path:     " + path + "\n\n" +
				"Value:\n" + content + "\n\n" +
				footer,
		)
}

// renderVaultBrowserPopup returns the Vault tree browser overlay.
func (m model) renderVaultBrowserPopup() string {
	var b strings.Builder

	if m.vaultBrowserLoading {
		b.WriteString(styleMuted.Render("Loading..."))
	} else if m.vaultBrowserError != "" {
		b.WriteString(styleErrorText.Render("Error: " + m.vaultBrowserError))
	} else if len(m.vaultBrowserEntries) == 0 {
		b.WriteString(styleMuted.Render("No entries found"))
	} else {
		maxVisible := min(15, len(m.vaultBrowserEntries))
		offset := 0
		if m.vaultBrowserCursor >= maxVisible {
			offset = m.vaultBrowserCursor - maxVisible + 1
		}

		for i := offset; i < len(m.vaultBrowserEntries) && i < offset+maxVisible; i++ {
			entry := m.vaultBrowserEntries[i]
			prefix := "  "
			style := styleNormal
			if i == m.vaultBrowserCursor {
				prefix = "> "
				style = styleSelected
			}

			icon := "  "
			if entry.IsDir {
				icon = "📁 "
			} else {
				icon = "📄 "
			}

			b.WriteString(style.Render(prefix+icon+entry.Name) + "\n")
		}
	}

	title := fmt.Sprintf("Browse Vault: %s", m.vaultBrowserPath)

	return stylePopup.
		Width(min(m.width-10, 55)).
		Render(
			styleTitle.Render(title) + "\n\n" +
				b.String() + "\n" +
				styleMuted.Render("j/k:nav  enter:open  backspace:up  esc:close"),
		)
}

// renderMappingFormPopup returns the add/edit mapping form overlay.
func (m model) renderMappingFormPopup() string {
	title := "New Secret Mapping"
	if m.mappingFormIsEdit {
		title = "Edit Secret Mapping"
	}

	targets := m.bridge.WorkspaceFiles(m.config, m.rootDir)
	targetLabel := "[none]"
	if m.mappingFormTarget >= 0 && m.mappingFormTarget < len(targets) {
		targetLabel = targets[m.mappingFormTarget].Label
	}

	fields := []struct {
		label string
		value string
	}{
		{"Vault path", m.mappingFormPath},
		{"Env var", m.mappingFormEnvVar},
		{"Target", targetLabel},
	}

	var b strings.Builder
	for i, f := range fields {
		label := styleDim.Render(fmt.Sprintf("  %-12s", f.label+":"))
		val := styleNormal.Render(f.value)
		if i == m.mappingFormField {
			label = styleKey.Render(fmt.Sprintf("> %-12s", f.label+":"))
			val = styleSelected.Render(f.value + "_")
		}
		b.WriteString(label + " " + val + "\n")
	}

	return stylePopup.
		Width(min(m.width-10, 55)).
		Render(
			styleTitle.Render(title) + "\n\n" +
				b.String() + "\n" +
				styleMuted.Render("tab:next field  enter:save  esc:cancel"),
		)
}

// renderConfirmPopup returns the delete confirmation overlay.
func (m model) renderConfirmPopup() string {
	choices := []string{"Cancel", "Delete"}
	var b strings.Builder
	for i, c := range choices {
		prefix := "  "
		style := styleNormal
		if i == m.confirmCursor {
			prefix = "> "
			style = styleSelected
		}
		b.WriteString(style.Render(prefix+c) + "\n")
	}

	return stylePopup.
		Width(min(m.width-10, 50)).
		Render(
			styleTitle.Render("Confirm Delete") + "\n\n" +
				styleNormal.Render(fmt.Sprintf("Delete %s from %s?",
					styleKey.Render(m.confirmEnvVar),
					m.confirmFile)) + "\n\n" +
				b.String() + "\n" +
				styleMuted.Render("j/k:nav  enter:confirm  esc:cancel"),
		)
}
