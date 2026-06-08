package filter

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func rk(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestMatches(t *testing.T) {
	var m Model
	// Inactive / empty query matches everything.
	if !m.Matches("anything") {
		t.Error("empty query should match everything")
	}
	m.Activate()
	for _, r := range "Us" {
		m.HandleKey(rk(r))
	}
	if m.Query() != "Us" {
		t.Fatalf("query = %q, want %q", m.Query(), "Us")
	}
	// Case-insensitive substring.
	if !m.Matches("users") || !m.Matches("BUSINESS") {
		t.Error("expected case-insensitive substring match")
	}
	if m.Matches("orders") {
		t.Error("did not expect match for 'orders'")
	}
}

func TestHandleKey(t *testing.T) {
	var m Model
	if m.HandleKey(rk('a')) {
		t.Error("inactive filter should not consume keys")
	}
	m.Activate()
	for _, r := range "abc" {
		if !m.HandleKey(rk(r)) {
			t.Errorf("expected %q to be consumed", r)
		}
	}
	m.HandleKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	for _, r := range "de" {
		m.HandleKey(rk(r))
	}
	if m.Query() != "abc de" {
		t.Fatalf("query = %q, want %q", m.Query(), "abc de")
	}
	// Backspace trims the last rune.
	m.HandleKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.Query() != "abc d" {
		t.Fatalf("after backspace query = %q, want %q", m.Query(), "abc d")
	}
	// Non-printable keys are not consumed (caller handles them).
	if m.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter}) {
		t.Error("enter should not be consumed by the filter")
	}
}

func TestActivateClear(t *testing.T) {
	var m Model
	m.Activate()
	m.HandleKey(rk('x'))
	if !m.Active() || m.Query() != "x" {
		t.Fatal("expected active with query 'x'")
	}
	m.Clear()
	if m.Active() || m.Query() != "" {
		t.Fatal("expected inactive and empty after Clear")
	}
	if m.View() != "" {
		t.Error("inactive View should be empty")
	}
}
