package modal

import (
	"testing"

	"github.com/tardanoir/seshat/internal/query"

	tea "charm.land/bubbletea/v2"
)

func historyFixture() []query.HistoryEntry {
	return []query.HistoryEntry{
		{SQL: "SELECT * FROM users", Connection: "prod"},
		{SQL: "INSERT INTO orders (id) VALUES (1)", Connection: "staging"},
		{SQL: "UPDATE users SET name = 'x'", Connection: "prod"},
	}
}

func TestHistoryPickerSearchBySQL(t *testing.T) {
	m := NewHistoryPicker(historyFixture())
	m.SetSize(100, 40)

	m, _ = m.Update(rk('/'))
	if !m.Searching() {
		t.Fatal("expected search mode after '/'")
	}
	for _, r := range "users" {
		m, _ = m.Update(rk(r))
	}

	vis := m.visible()
	if len(vis) != 2 {
		t.Fatalf("visible = %d, want 2 (the two queries mentioning users)", len(vis))
	}

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a select command")
	}
	sel, ok := cmd().(HistorySelectedMsg)
	if !ok || sel.SQL != "SELECT * FROM users" {
		t.Fatalf("selected = %#v, want first matching entry", cmd())
	}
}

func TestHistoryPickerSearchByConnection(t *testing.T) {
	m := NewHistoryPicker(historyFixture())
	m.SetSize(100, 40)

	m, _ = m.Update(rk('/'))
	for _, r := range "staging" {
		m, _ = m.Update(rk(r))
	}
	vis := m.visible()
	if len(vis) != 1 || vis[0].Connection != "staging" {
		t.Fatalf("visible = %v, want the single staging entry", vis)
	}
}

func TestHistoryPickerEscClearsSearch(t *testing.T) {
	m := NewHistoryPicker(historyFixture())
	m.SetSize(100, 40)

	m, _ = m.Update(rk('/'))
	for _, r := range "zzz" {
		m, _ = m.Update(rk(r))
	}
	if len(m.visible()) != 0 {
		t.Fatalf("expected no matches for 'zzz', got %d", len(m.visible()))
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.Searching() {
		t.Fatal("expected search cleared after Esc")
	}
	if len(m.visible()) != 3 {
		t.Fatalf("after clear visible = %d, want 3", len(m.visible()))
	}
}

func TestHistoryPickerNavigationScrolls(t *testing.T) {
	// More entries than fit; the scroll window should follow the cursor.
	entries := make([]query.HistoryEntry, 40)
	for i := range entries {
		entries[i] = query.HistoryEntry{SQL: "q", Connection: "c"}
	}
	m := NewHistoryPicker(entries)
	m.SetSize(100, 20) // visibleRows clamps to a small window

	for i := 0; i < 30; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.cursor != 30 {
		t.Fatalf("cursor = %d, want 30", m.cursor)
	}
	if m.cursor < m.scrollY || m.cursor >= m.scrollY+m.visibleRows() {
		t.Fatalf("cursor %d outside scroll window [%d,%d)", m.cursor, m.scrollY, m.scrollY+m.visibleRows())
	}
	// Render must not panic with a scrolled window.
	if m.View() == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestHistoryPickerViewBranches(t *testing.T) {
	// Empty history, no-match search, and a normal render must all be safe.
	empty := NewHistoryPicker(nil)
	empty.SetSize(100, 40)
	_ = empty.View()

	m := NewHistoryPicker(historyFixture())
	m.SetSize(100, 40)
	if m.View() == "" {
		t.Fatal("expected non-empty view")
	}
	m, _ = m.Update(rk('/'))
	for _, r := range "zzz" {
		m, _ = m.Update(rk(r))
	}
	if m.View() == "" {
		t.Fatal("expected non-empty view for no-match state")
	}
}
