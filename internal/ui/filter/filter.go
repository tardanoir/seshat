// Package filter provides a small reusable incremental-search state used by the
// app's list menus (sidebar sections and the connection/template pickers). It
// owns the "/" search query and matching; callers apply Matches to their items
// and render View where it fits.
package filter

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	active bool
	query  string
}

func (m Model) Active() bool  { return m.active }
func (m Model) Query() string { return m.query }

// Activate enters search mode with an empty query.
func (m *Model) Activate() { m.active = true; m.query = "" }

// Clear leaves search mode and drops the query.
func (m *Model) Clear() { m.active = false; m.query = "" }

// HandleKey edits the query for printable keys and backspace while active. It
// reports whether the key was consumed (so the caller can ignore it otherwise).
func (m *Model) HandleKey(msg tea.KeyMsg) bool {
	if !m.active {
		return false
	}
	switch msg.String() {
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
		return true
	case "space":
		m.query += " "
		return true
	}
	if t := msg.Key().Text; t != "" {
		m.query += t
		return true
	}
	return false
}

// Matches reports whether s contains the query, case-insensitively. An empty
// query matches everything.
func (m Model) Matches(s string) bool {
	if m.query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(m.query))
}

var (
	promptStyle = lipgloss.NewStyle().Foreground(style.ColorPrimary).Bold(true)
	queryStyle  = lipgloss.NewStyle().Foreground(style.ColorText)
)

// View renders the search prompt, e.g. "/users▏". Empty when inactive.
func (m Model) View() string {
	if !m.active {
		return ""
	}
	return promptStyle.Render("/") + queryStyle.Render(m.query) + promptStyle.Render("▏")
}
