package modal

import (
	"fmt"
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

type ConnectionModel struct {
	names   []string
	conns   map[string]config.Connection
	cursor  int
	current string
	width   int
	height  int
}

func NewConnection(conns map[string]config.Connection, current string) ConnectionModel {
	names := make([]string, 0, len(conns))
	for k := range conns {
		names = append(names, k)
	}
	sort.Strings(names)
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
			conn := m.conns[name]
			return m, func() tea.Msg {
				return SwitchConnectionMsg{Name: name, Connection: conn}
			}
		}
	}
	return m, nil
}

func (m ConnectionModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("Switch Connection"))
	b.WriteString("\n\n")

	for i, name := range m.names {
		c := m.conns[name]
		label := fmt.Sprintf("%s (%s:%d/%s)", name, c.Host, c.Port, c.Database)
		if name == m.current {
			label += " *"
		}
		if i == m.cursor {
			b.WriteString(style.ListSelected.Render("▸ " + label))
		} else {
			b.WriteString(style.ListItem.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↑↓ navigate  ↵ select  Esc close"))

	modalW := 50
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

