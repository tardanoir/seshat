package sidebar

import (
	"testing"

	"github.com/tardanoir/seshat/internal/query"

	tea "charm.land/bubbletea/v2"
)

func rk(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func typeRunes(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(rk(r))
	}
	return m
}

func TestSidebarSearchFiltersQueries(t *testing.T) {
	m := New()
	m.SetFocused(true)
	m.SetQueries([]query.SavedQuery{
		{Name: "alpha", Content: "SELECT 1"},
		{Name: "beta", Content: "SELECT 2"},
		{Name: "gamma", Content: "SELECT 3"},
	})
	m.switchSection(SectionQueries)

	m, _ = m.Update(rk('/'))
	if !m.search.Active() {
		t.Fatal("search should be active after '/'")
	}
	m = typeRunes(m, "lph") // matches only "alpha"

	vis := m.visibleQueries()
	if len(vis) != 1 || vis[0].Name != "alpha" {
		t.Fatalf("visibleQueries = %v, want [alpha]", vis)
	}

	// Enter selects the filtered entry.
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a select command")
	}
	sel, ok := cmd().(SelectQueryMsg)
	if !ok || sel.Content != "SELECT 1" {
		t.Fatalf("selected = %#v, want alpha's content", cmd())
	}

	// Esc clears the search and restores the full list.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.search.Active() {
		t.Fatal("search should be cleared after Esc")
	}
	if len(m.visibleQueries()) != 3 {
		t.Fatalf("after clear visibleQueries = %d, want 3", len(m.visibleQueries()))
	}
}

func TestSidebarSearchDigitsAreTyped(t *testing.T) {
	m := New()
	m.SetFocused(true)
	m.SetQueries([]query.SavedQuery{{Name: "report_2024"}, {Name: "report_2025"}})
	m.switchSection(SectionQueries)

	m, _ = m.Update(rk('/'))
	m = typeRunes(m, "2025") // digits must edit the query, not switch sections

	if m.activeSection != SectionQueries {
		t.Fatalf("active section changed to %v; digits should be typed while searching", m.activeSection)
	}
	if m.search.Query() != "2025" {
		t.Fatalf("query = %q, want %q", m.search.Query(), "2025")
	}
	vis := m.visibleQueries()
	if len(vis) != 1 || vis[0].Name != "report_2025" {
		t.Fatalf("visibleQueries = %v, want [report_2025]", vis)
	}
}

// TestSidebarSearchTablesIndexMapping checks the cursor→real-table mapping when
// the table list is filtered (the expansion is on the underlying entry).
func TestSidebarSearchTablesIndexMapping(t *testing.T) {
	m := New()
	m.SetFocused(true)
	m.SetTables([]TableEntry{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "orders"},
		{Schema: "public", Name: "user_roles"},
	})
	// Tables is the default active section.
	m, _ = m.Update(rk('/'))
	m = typeRunes(m, "user") // matches users + user_roles, not orders

	vis := m.visibleTableIdx()
	if len(vis) != 2 || vis[0] != 0 || vis[1] != 2 {
		t.Fatalf("visibleTableIdx = %v, want [0 2]", vis)
	}

	// Move to the second match (user_roles, real index 2) and select it.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected RequestColumns command")
	}
	rc, ok := cmd().(RequestColumnsMsg)
	if !ok || rc.TableName != "user_roles" {
		t.Fatalf("requested = %#v, want user_roles", cmd())
	}
}
