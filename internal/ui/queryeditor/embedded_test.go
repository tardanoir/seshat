package queryeditor

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func newEmbeddedModel(t *testing.T) Model {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed; skipping embedded model test")
	}
	m := New(true)
	if !m.NvimActive() {
		t.Skip("embedded nvim did not start")
	}
	m.SetSize(48, 8)
	return m
}

func waitForView(t *testing.T, m Model, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(ansi.Strip(m.View()), want) {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for view to contain %q; got:\n%s", want, ansi.Strip(m.View()))
}

// TestEmbeddedModelRendersBuffer drives the full editor model: a value set
// through the model must show up in the rendered nvim grid and read back out.
func TestEmbeddedModelRendersBuffer(t *testing.T) {
	m := newEmbeddedModel(t)
	defer m.Close()

	m.SetValue("select 42;")

	waitForView(t, m, "select 42;")

	if got := m.Value(); got != "select 42;" {
		t.Errorf("Value() = %q, want %q", got, "select 42;")
	}
	// parseStatements trims the trailing ";" for execution (same as the other
	// editor modes).
	if got := m.SelectedStatement(); got != "select 42" {
		t.Errorf("SelectedStatement() = %q, want %q", got, "select 42")
	}
	if lines := strings.Count(m.View(), "\n") + 1; lines != 8 {
		t.Errorf("View rendered %d rows, want 8", lines)
	}
}

// TestEmbeddedModelHandlesKeys feeds key presses through Update and verifies
// they reach the embedded editor and change the buffer.
func TestEmbeddedModelHandlesKeys(t *testing.T) {
	m := newEmbeddedModel(t)
	defer m.Close()

	m.SetValue("")

	// "o" opens a line below in insert mode; type "xyz"; <Esc> back to normal.
	keys := []tea.KeyPressMsg{
		{Code: 'o', Text: "o"},
		{Code: 'x', Text: "x"},
		{Code: 'y', Text: "y"},
		{Code: 'z', Text: "z"},
		{Code: tea.KeyEscape},
	}
	for _, k := range keys {
		m, _ = m.Update(k)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(m.Value(), "xyz") {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("typed text never reached buffer; Value()=%q", m.Value())
}

// TestEmbeddedModelCompletion drives the schema-aware popup: typing a FROM
// clause prefix in insert mode must surface matching table names.
func TestEmbeddedModelCompletion(t *testing.T) {
	m := newEmbeddedModel(t)
	defer m.Close()

	m.SetSchema([]TableRef{{Name: "users"}, {Name: "orders"}}, []ColumnRef{{Name: "email"}})
	m.SetValue("")

	typed := []tea.KeyPressMsg{
		{Code: 'i', Text: "i"},
		{Code: 's', Text: "s"}, {Code: 'e', Text: "e"}, {Code: 'l', Text: "l"},
		{Code: 'e', Text: "e"}, {Code: 'c', Text: "c"}, {Code: 't', Text: "t"},
		{Code: tea.KeySpace}, {Code: '*', Text: "*"}, {Code: tea.KeySpace},
		{Code: 'f', Text: "f"}, {Code: 'r', Text: "r"}, {Code: 'o', Text: "o"}, {Code: 'm', Text: "m"},
		{Code: tea.KeySpace},
		{Code: 'u', Text: "u"}, {Code: 's', Text: "s"},
	}
	for _, k := range typed {
		m, _ = m.Update(k)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m, _ = m.Update(NvimRedrawMsg{})
		if m.CompletionOpen() {
			for _, it := range m.comp.items {
				if it.Text == "users" && it.Kind == KindTable {
					return // popup opened with the expected schema suggestion
				}
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("completion popup never offered 'users'; sql=%q open=%v items=%+v",
		m.sql, m.CompletionOpen(), m.comp.items)
}

// TestEmbeddedModelStatementSelection verifies cursor-aware statement selection
// against the live buffer.
func TestEmbeddedModelStatementSelection(t *testing.T) {
	m := newEmbeddedModel(t)
	defer m.Close()

	m.SetValue("select 1;\nselect 2;")
	waitForView(t, m, "select 2;")

	// Move to the last line; the selected statement should follow the cursor.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.SelectedStatement() == "select 2" {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("SelectedStatement() = %q, want %q", m.SelectedStatement(), "select 2")
}
