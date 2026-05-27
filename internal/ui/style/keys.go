package style

import (
	"charm.land/bubbles/v2/key"
)

type KeyMap struct {
	Execute       key.Binding
	Editor        key.Binding
	Save          key.Binding
	Template      key.Binding
	ConnPick      key.Binding
	ToggleSidebar key.Binding
	Export        key.Binding
	Tab           key.Binding
	ShiftTab      key.Binding
	Escape        key.Binding
	Quit          key.Binding
	Delete        key.Binding
	Suspend       key.Binding
	Enter         key.Binding
	Up            key.Binding
	Down          key.Binding
	AIGenerate    key.Binding
}

var Keys = KeyMap{
	Execute: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("C-r", "run"),
	),
	Editor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("C-e", "editor"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("C-w", "save"),
	),
	Template: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("C-t", "tmpl"),
	),
	ConnPick: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("C-n", "conn"),
	),
	ToggleSidebar: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("C-\\", "sidebar"),
	),
	Export: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("C-x", "export"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "focus"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("S-Tab", "back"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("C-c", "quit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("C-d", "delete"),
	),
	Suspend: key.NewBinding(
		key.WithKeys("ctrl+z"),
		key.WithHelp("C-z", "suspend"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "select"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("Up", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("Down", "down"),
	),
	AIGenerate: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("C-a", "ai"),
	),
}

// Keybindings mirrors config.Keybindings to avoid import cycle.
type Keybindings struct {
	Execute       string
	Editor        string
	Save          string
	Template      string
	ConnPick      string
	ToggleSidebar string
	Export        string
	Tab           string
	ShiftTab      string
	Quit          string
	Delete        string
	Suspend       string
	AIGenerate    string
}

func ApplyKeybindings(kb Keybindings) {
	apply := func(b *key.Binding, k string) {
		if k != "" {
			*b = key.NewBinding(key.WithKeys(k))
		}
	}
	apply(&Keys.Execute, kb.Execute)
	apply(&Keys.Editor, kb.Editor)
	apply(&Keys.Save, kb.Save)
	apply(&Keys.Template, kb.Template)
	apply(&Keys.ConnPick, kb.ConnPick)
	apply(&Keys.ToggleSidebar, kb.ToggleSidebar)
	apply(&Keys.Export, kb.Export)
	apply(&Keys.Tab, kb.Tab)
	apply(&Keys.ShiftTab, kb.ShiftTab)
	apply(&Keys.Quit, kb.Quit)
	apply(&Keys.Delete, kb.Delete)
	apply(&Keys.Suspend, kb.Suspend)
	apply(&Keys.AIGenerate, kb.AIGenerate)
}
