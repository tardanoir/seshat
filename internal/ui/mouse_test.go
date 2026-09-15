package ui

import (
	"strings"
	"testing"

	"github.com/tardanoir/seshat/internal/config"

	tea "charm.land/bubbletea/v2"
)

func newSizedApp(t *testing.T, w, h int) App {
	t.Helper()
	cfg := &config.Config{Connections: map[string]config.Connection{}}
	a := NewApp(cfg, "test", nil, "")
	m, _ := a.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m.(App)
}

// paneAt derives the editor/results split from previewH; composeFrame derives it
// from the number of lines the editor actually renders. If these ever disagree,
// every click below the editor lands on the wrong pane.
func TestEditorRowsMatchesRenderedEditor(t *testing.T) {
	for _, focus := range []Focus{FocusSidebar, FocusPreview, FocusResults} {
		a := newSizedApp(t, 120, 40)
		a.setFocus(focus)

		rendered := len(strings.Split(a.preview.View(), "\n"))
		if got := a.editorRows(); got != rendered {
			t.Errorf("focus %d: editorRows() = %d, but the editor renders %d lines",
				focus, got, rendered)
		}
	}
}

func TestPaneAtMapsPanels(t *testing.T) {
	a := newSizedApp(t, 120, 40)
	if !a.sidebarVisible {
		t.Fatal("expected the sidebar to be visible")
	}

	// A sidebar row below the horizontal rule shifts up by one content line.
	hit := a.paneAt(3, 1+5)
	if !hit.ok || hit.focus != FocusSidebar {
		t.Fatalf("sidebar hit = %+v, want a sidebar hit", hit)
	}
	if hit.y != 4 {
		t.Errorf("sidebar content line = %d, want 4 (rule at pane row 3 shifts it)", hit.y)
	}

	// The sidebar's rule row itself is chrome, not content.
	if h := a.paneAt(3, 1+sidebarRuleRow); h.ok {
		t.Errorf("pane row %d is the sidebar rule, want no hit, got %+v", sidebarRuleRow, h)
	}

	mainX := a.sidebarW + 2
	if h := a.paneAt(mainX, 1); !h.ok || h.focus != FocusPreview {
		t.Errorf("top of the main column = %+v, want the editor", h)
	}
	// The divider between editor and results is chrome.
	if h := a.paneAt(mainX, 1+a.editorRows()); h.ok {
		t.Errorf("editor/results divider returned a hit: %+v", h)
	}
	// First results row.
	h := a.paneAt(mainX, 1+a.editorRows()+1)
	if !h.ok || h.focus != FocusResults || h.y != 0 {
		t.Errorf("first results row = %+v, want results y=0", h)
	}

	// Borders and chrome must not register.
	for _, tc := range []struct {
		name string
		x, y int
	}{
		{"top border", 5, 0},
		{"left border", 0, 5},
		{"status bar", 5, a.mainH + 2},
		{"past right edge", a.width + 5, 5},
	} {
		if h := a.paneAt(tc.x, tc.y); h.ok {
			t.Errorf("%s registered a hit: %+v", tc.name, h)
		}
	}
}

// Content coordinates must skip the padding column and the focus bar.
func TestPaneAtContentInset(t *testing.T) {
	a := newSizedApp(t, 120, 40)
	mainX := a.sidebarW + 2
	h := a.paneAt(mainX, 1+a.editorRows()+1)
	if !h.ok {
		t.Fatal("expected a results hit")
	}
	if h.x != -panelContentInset {
		t.Errorf("x at the panel's left edge = %d, want %d", h.x, -panelContentInset)
	}
	h = a.paneAt(mainX+panelContentInset, 1+a.editorRows()+1)
	if h.x != 0 {
		t.Errorf("x at the first content column = %d, want 0", h.x)
	}
}

// A click must not overwrite SQL already in the buffer.
func TestInsertTableIntoEditorPreservesBuffer(t *testing.T) {
	a := newSizedApp(t, 120, 40)
	a.preview.SetValue("select 1;")
	a.insertTableIntoEditor("public", "users")

	got := a.preview.Value()
	if !strings.Contains(got, "select 1;") {
		t.Errorf("existing SQL was lost: %q", got)
	}
	if !strings.Contains(got, "select * from users;") {
		t.Errorf("table statement missing: %q", got)
	}
}

func TestInsertTableIntoEditorQualifiesSchema(t *testing.T) {
	cases := []struct {
		schema, name, want string
	}{
		{"public", "users", "select * from users;"},
		{"", "people", "select * from people;"}, // file-backed drivers report no schema
		{"analytics", "events", "select * from analytics.events;"},
	}
	for _, tc := range cases {
		a := newSizedApp(t, 120, 40)
		a.insertTableIntoEditor(tc.schema, tc.name)
		if got := a.preview.Value(); got != tc.want {
			t.Errorf("schema=%q name=%q -> %q, want %q", tc.schema, tc.name, got, tc.want)
		}
	}
}

// Clicking a pane focuses it.
func TestMouseClickFocusesPane(t *testing.T) {
	a := newSizedApp(t, 120, 40)
	a.setFocus(FocusPreview)

	m, _ := a.Update(tea.MouseClickMsg{X: a.sidebarW + 3, Y: 1 + a.editorRows() + 1, Button: tea.MouseLeft})
	if got := m.(App).focus; got != FocusResults {
		t.Errorf("focus after clicking the results pane = %d, want FocusResults", got)
	}
}

// Mouse events must not leak through while a modal is open.
func TestMouseIgnoredWhileModalOpen(t *testing.T) {
	a := newSizedApp(t, 120, 40)
	a.setFocus(FocusPreview)
	a.modalState = ModalHelp

	m, _ := a.Update(tea.MouseClickMsg{X: 3, Y: 5, Button: tea.MouseLeft})
	if got := m.(App).focus; got != FocusPreview {
		t.Errorf("a click changed focus to %d while a modal was open", got)
	}
}
