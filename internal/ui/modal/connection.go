package modal

import (
	"sort"
	"strings"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type SwitchConnectionMsg struct {
	Name       string
	Connection config.Connection
}

type OpenAddConnMsg struct{}

type DeleteConnectionMsg struct {
	Name string
}

type ConnectionModel struct {
	names   []string
	conns   map[string]config.Connection
	cursor  int
	current string
	width   int
	height  int
}

const newConnEntry = "+ New Connection"

func NewConnection(conns map[string]config.Connection, current string) ConnectionModel {
	names := make([]string, 0, len(conns))
	for k := range conns {
		names = append(names, k)
	}
	sort.Strings(names)
	// append the "new connection" entry at the end.
	names = append(names, newConnEntry)
	cursor := 0
	for i, n := range names {
		if n == current {
			cursor = i
			break
		}
	}
	return ConnectionModel{
		names:   names,
		conns:   conns,
		cursor:  cursor,
		current: current,
	}
}

func (m *ConnectionModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ConnectionModel) Update(msg tea.Msg) (ConnectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Up), key.Matches(msg, key.NewBinding(key.WithKeys("k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, style.Keys.Down), key.Matches(msg, key.NewBinding(key.WithKeys("j"))):
			if m.cursor < len(m.names)-1 {
				m.cursor++
			}
		case key.Matches(msg, style.Keys.Enter):
			name := m.names[m.cursor]
			if name == newConnEntry {
				return m, func() tea.Msg { return OpenAddConnMsg{} }
			}
			conn := m.conns[name]
			return m, func() tea.Msg {
				return SwitchConnectionMsg{Name: name, Connection: conn}
			}
		case key.Matches(msg, style.Keys.Delete):
			name := m.names[m.cursor]
			if name != newConnEntry {
				return m, func() tea.Msg { return DeleteConnectionMsg{Name: name} }
			}
		}
	}
	return m, nil
}

func connTypeBadge(driverType string) string {
	switch driverType {
	case "postgres", "postgresql":
		return style.StatusConn.Render("pg")
	case "sqlite", "sqlite3":
		return style.StatusMsg.Render("sqlite")
	case "csv":
		return style.StatusMsg.Render("csv")
	case "json":
		return style.StatusMsg.Render("json")
	default:
		return style.StatusMsg.Render(driverType)
	}
}

func (m ConnectionModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("Connections"))
	b.WriteString("\n\n")

	for i, name := range m.names {
		selected := i == m.cursor
		prefix := "  "
		if selected {
			prefix = style.ListSelected.Render("▸ ")
		}

		if name == newConnEntry {
			b.WriteString(style.StatusMsg.Render("────────────────────────"))
			b.WriteString("\n")
			if selected {
				b.WriteString(style.ListSelected.Render("▸ + New Connection"))
			} else {
				b.WriteString(style.ListItem.Render("  + New Connection"))
			}
			b.WriteString("\n")
			continue
		}

		c := m.conns[name]
		badge := connTypeBadge(c.DriverType())
		if c.HasSSH() {
			badge += " " + style.StatusKey.Render("ssh")
		}
		detail := style.StatusMsg.Render(c.DisplayLabel())

		connName := name
		if name == m.current {
			connName += style.Success.Render(" *")
		}

		if selected {
			b.WriteString(prefix + style.ListSelected.Render(connName) + "  " + badge)
		} else {
			b.WriteString(prefix + style.ListItem.Render(connName) + "  " + badge)
		}
		b.WriteString("\n")
		b.WriteString("    " + detail)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	hints := []string{
		style.StatusKey.Render("↑↓") + " " + style.StatusKeyLabel.Render("navigate"),
		style.StatusKey.Render("↵") + " " + style.StatusKeyLabel.Render("connect"),
		style.StatusKey.Render("C-d") + " " + style.StatusKeyLabel.Render("delete"),
		style.StatusKey.Render("Esc") + " " + style.StatusKeyLabel.Render("close"),
	}
	b.WriteString("  " + strings.Join(hints, "  "))

	modalW := 60
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

