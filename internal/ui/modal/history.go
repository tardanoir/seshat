package modal

import (
	"fmt"
	"strings"

	"github.com/tardanoir/seshat/internal/query"
	"github.com/tardanoir/seshat/internal/ui/filter"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HistorySelectedMsg is emitted when the user picks a history entry to load.
type HistorySelectedMsg struct{ SQL string }

// HistoryPickerModel is a searchable, scrollable browser over the query history.
type HistoryPickerModel struct {
	entries []query.HistoryEntry
	cursor  int
	scrollY int
	search  filter.Model
	width   int
	height  int
}

func NewHistoryPicker(entries []query.HistoryEntry) HistoryPickerModel {
	return HistoryPickerModel{entries: entries}
}

func (m *HistoryPickerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Searching reports whether the picker is in incremental-search mode.
func (m HistoryPickerModel) Searching() bool { return m.search.Active() }

// visible returns the entries matching the search (by SQL or connection), or
// all of them when not searching.
func (m HistoryPickerModel) visible() []query.HistoryEntry {
	if !m.search.Active() {
		return m.entries
	}
	out := make([]query.HistoryEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if m.search.Matches(e.SQL) || m.search.Matches(e.Connection) {
			out = append(out, e)
		}
	}
	return out
}

// visibleRows is how many entry rows fit in the modal for the current height.
func (m HistoryPickerModel) visibleRows() int {
	r := m.height - 9
	if r < 5 {
		r = 5
	}
	if r > 18 {
		r = 18
	}
	return r
}

func (m HistoryPickerModel) Update(msg tea.Msg) (HistoryPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		vis := m.visible()
		if m.search.Active() {
			switch msg.String() {
			case "esc":
				m.search.Clear()
				m.cursor = 0
				m.scrollY = 0
			case "up":
				m.moveUp()
			case "down":
				m.moveDown(len(vis))
			case "enter":
				return m.selectAt(vis)
			default:
				if m.search.HandleKey(msg) {
					m.cursor = 0
					m.scrollY = 0
				}
			}
			return m, nil
		}
		switch {
		case msg.String() == "/":
			m.search.Activate()
			m.cursor = 0
			m.scrollY = 0
		case key.Matches(msg, style.Keys.Up), key.Matches(msg, key.NewBinding(key.WithKeys("k"))):
			m.moveUp()
		case key.Matches(msg, style.Keys.Down), key.Matches(msg, key.NewBinding(key.WithKeys("j"))):
			m.moveDown(len(vis))
		case key.Matches(msg, style.Keys.Enter):
			return m.selectAt(vis)
		}
	}
	return m, nil
}

func (m *HistoryPickerModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	}
}

func (m *HistoryPickerModel) moveDown(n int) {
	if m.cursor < n-1 {
		m.cursor++
	}
	if rows := m.visibleRows(); m.cursor >= m.scrollY+rows {
		m.scrollY = m.cursor - rows + 1
	}
}

func (m HistoryPickerModel) selectAt(vis []query.HistoryEntry) (HistoryPickerModel, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(vis) {
		return m, nil
	}
	sql := vis[m.cursor].SQL
	return m, func() tea.Msg { return HistorySelectedMsg{SQL: sql} }
}

const historyModalW = 80

func (m HistoryPickerModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("Query History"))
	if m.search.Active() {
		b.WriteString("   " + m.search.View())
	}
	b.WriteString("\n\n")

	vis := m.visible()
	switch {
	case len(m.entries) == 0:
		b.WriteString(style.ListItem.Render("No history yet."))
		b.WriteString("\n")
	case len(vis) == 0:
		b.WriteString(style.ListItem.Render("No matches."))
		b.WriteString("\n")
	default:
		rows := m.visibleRows()
		end := m.scrollY + rows
		if end > len(vis) {
			end = len(vis)
		}
		lineW := historyModalW - 6
		for i := m.scrollY; i < end; i++ {
			e := vis[i]
			ts := style.StatusMsg.Render(e.Timestamp.Local().Format("01-02 15:04"))
			sql := histOneLine(e.SQL)
			if i == m.cursor {
				b.WriteString(style.ListSelected.Render("▸ " + ts + "  " + histTrunc(sql, lineW-14)))
			} else {
				b.WriteString("  " + ts + "  " + style.ListItem.Render(histTrunc(sql, lineW-14)))
			}
			b.WriteString("\n")
		}
		if len(vis) > rows {
			b.WriteString(style.StatusMsg.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(vis))))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↑↓ navigate  / search  ↵ load  Esc close"))

	content := style.ModalOverlay.Width(historyModalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// histOneLine collapses a SQL statement to a single line for list display.
func histOneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.Join(strings.Fields(s), " ")
}

func histTrunc(s string, maxW int) string {
	r := []rune(s)
	if maxW <= 0 || len(r) <= maxW {
		return s
	}
	if maxW < 2 {
		return string(r[:maxW])
	}
	return string(r[:maxW-1]) + "…"
}
