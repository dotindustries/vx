package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keyboard shortcuts for the TUI.
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Tab        key.Binding
	Enter      key.Binding
	Filter     key.Binding
	Env        key.Binding
	Help       key.Binding
	Copy       key.Binding
	Add        key.Binding
	Edit       key.Binding
	Delete     key.Binding
	Escape     key.Binding
	Quit       key.Binding
	ForceQuit  key.Binding
	Backspace  key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("j/k", "navigate"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("", ""),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view secret"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Env: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "environment"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy value"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add mapping"),
	),
	Edit: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "edit mapping"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete mapping"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close/cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	ForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "force quit"),
	),
	Backspace: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("backspace", "go up"),
	),
}
