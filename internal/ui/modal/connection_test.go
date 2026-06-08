package modal

import (
	"testing"

	"github.com/tardanoir/seshat/internal/config"

	tea "charm.land/bubbletea/v2"
)

func rk(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func connFixture() map[string]config.Connection {
	return map[string]config.Connection{
		"prod":    {Host: "prod.db", Port: 5432, Database: "app", User: "u"},
		"staging": {Host: "staging.db", Port: 5432, Database: "app", User: "u"},
		"local":   {Host: "localhost", Port: 5432, Database: "dev", User: "u"},
	}
}

func TestConnectionSearchFilters(t *testing.T) {
	m := NewConnection(connFixture(), "local")
	m, _ = m.Update(rk('/'))
	if !m.Searching() {
		t.Fatal("expected search mode after '/'")
	}
	for _, r := range "prod" {
		m, _ = m.Update(rk(r))
	}

	names := m.displayNames()
	// Only "prod" matches; the new-connection entry is always pinned last.
	if len(names) != 2 || names[0] != "prod" || names[1] != newConnEntry {
		t.Fatalf("displayNames = %v, want [prod %q]", names, newConnEntry)
	}

	// Enter on the single match switches to it.
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a switch command")
	}
	sw, ok := cmd().(SwitchConnectionMsg)
	if !ok || sw.Name != "prod" {
		t.Fatalf("selected = %#v, want prod", cmd())
	}
}

func TestConnectionSearchMatchesLabel(t *testing.T) {
	m := NewConnection(connFixture(), "local")
	m, _ = m.Update(rk('/'))
	for _, r := range "localhost" { // matches local by its DisplayLabel host
		m, _ = m.Update(rk(r))
	}
	names := m.displayNames()
	if len(names) != 2 || names[0] != "local" {
		t.Fatalf("displayNames = %v, want [local %q]", names, newConnEntry)
	}
}

func TestConnectionSearchNoMatchStillHasNewEntry(t *testing.T) {
	m := NewConnection(connFixture(), "local")
	m, _ = m.Update(rk('/'))
	for _, r := range "zzzzz" {
		m, _ = m.Update(rk(r))
	}
	names := m.displayNames()
	if len(names) != 1 || names[0] != newConnEntry {
		t.Fatalf("displayNames = %v, want [%q]", names, newConnEntry)
	}

	// Esc exits search.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.Searching() {
		t.Fatal("expected search cleared after Esc")
	}
}
