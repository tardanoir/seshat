package nvimbridge

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func newTestBridge(w, h int) *Bridge {
	b := &Bridge{hls: map[int]hlAttr{}, defFg: -1, defBg: -1, defSp: -1}
	b.allocGrid(w, h)
	return b
}

// rowText returns the visible (ansi-stripped) text of a rendered row.
func rowText(rendered string, row int) string {
	lines := strings.Split(rendered, "\n")
	if row >= len(lines) {
		return ""
	}
	return strings.TrimRight(ansi.Strip(lines[row]), " ")
}

func TestGridLineRender(t *testing.T) {
	b := newTestBridge(12, 2)
	b.handleRedraw(
		[]any{"default_colors_set", []any{int64(0xffffff), int64(0x000000), int64(0)}},
		[]any{"grid_line", []any{int64(1), int64(0), int64(0), []any{
			[]any{"S"}, []any{"E"}, []any{"L"}, []any{"E"}, []any{"C"}, []any{"T"},
		}}},
		[]any{"flush"},
	)

	out := b.Render(12, 2, false)
	if got := rowText(out, 0); got != "SELECT" {
		t.Errorf("row 0 = %q, want %q", got, "SELECT")
	}
	if lines := strings.Count(out, "\n") + 1; lines != 2 {
		t.Errorf("rendered %d rows, want 2", lines)
	}
}

func TestGridLineRepeat(t *testing.T) {
	b := newTestBridge(8, 1)
	// A single cell with a repeat count fills the row (used for clears).
	b.handleRedraw([]any{"grid_line", []any{int64(1), int64(0), int64(0), []any{
		[]any{"x", int64(0), int64(5)},
	}}})
	if got := rowText(b.Render(8, 1, false), 0); got != "xxxxx" {
		t.Errorf("row 0 = %q, want %q", got, "xxxxx")
	}
}

func TestHighlightPersistsAcrossCells(t *testing.T) {
	b := newTestBridge(8, 1)
	b.handleRedraw(
		[]any{"hl_attr_define", []any{int64(1), map[string]any{"foreground": int64(0xff0000), "bold": true}, map[string]any{}, []any{}}},
		// hl id is given once and persists to the following cell.
		[]any{"grid_line", []any{int64(1), int64(0), int64(0), []any{
			[]any{"a", int64(1)}, []any{"b"},
		}}},
	)
	if got := b.hlOf(1); got.fg != 0xff0000 || !got.bold {
		t.Errorf("hl 1 = %+v, want fg=0xff0000 bold", got)
	}
	if b.cells[0][1].hl != 1 {
		t.Errorf("second cell hl = %d, want 1 (persisted)", b.cells[0][1].hl)
	}
	if got := rowText(b.Render(8, 1, false), 0); got != "ab" {
		t.Errorf("row 0 = %q, want %q", got, "ab")
	}
}

func TestGridScrollUp(t *testing.T) {
	b := newTestBridge(4, 3)
	put := func(row int, s string) {
		cells := make([]any, len(s))
		for i, r := range s {
			cells[i] = []any{string(r)}
		}
		b.handleRedraw([]any{"grid_line", []any{int64(1), int64(row), int64(0), cells}})
	}
	put(0, "aaaa")
	put(1, "bbbb")
	put(2, "cccc")

	// Scroll the whole grid up by one row: row 1 -> row 0, row 2 -> row 1.
	b.handleRedraw([]any{"grid_scroll", []any{int64(1), int64(0), int64(3), int64(0), int64(4), int64(1), int64(0)}})

	out := b.Render(4, 3, false)
	if got := rowText(out, 0); got != "bbbb" {
		t.Errorf("after scroll row 0 = %q, want %q", got, "bbbb")
	}
	if got := rowText(out, 1); got != "cccc" {
		t.Errorf("after scroll row 1 = %q, want %q", got, "cccc")
	}
}

func TestCursorReversesCell(t *testing.T) {
	b := newTestBridge(4, 1)
	b.handleRedraw(
		[]any{"default_colors_set", []any{int64(0xffffff), int64(0x000000), int64(0)}},
		[]any{"grid_line", []any{int64(1), int64(0), int64(0), []any{[]any{"h"}, []any{"i"}}}},
		[]any{"grid_cursor_goto", []any{int64(1), int64(0), int64(1)}},
	)
	// Focused render must place a styled (reverse) run at the cursor; the visible
	// text is unchanged but the cursor cell carries extra escape sequences.
	focused := b.Render(4, 1, true)
	unfocused := b.Render(4, 1, false)
	if rowText(focused, 0) != "hi" {
		t.Errorf("focused text = %q, want %q", rowText(focused, 0), "hi")
	}
	if len(focused) <= len(unfocused) {
		t.Errorf("focused render (%d) should carry more styling than unfocused (%d)", len(focused), len(unfocused))
	}
}

func TestConversionHelpers(t *testing.T) {
	if asInt(int64(5)) != 5 || asInt(uint64(7)) != 7 || asInt(float64(9)) != 9 {
		t.Error("asInt numeric handling")
	}
	if asString("x") != "x" || asString([]byte("y")) != "y" {
		t.Error("asString handling")
	}
	if !asBool(true) || asBool(nil) {
		t.Error("asBool handling")
	}
	m := asMap(map[string]any{"k": int64(1)})
	if asInt(m["k"]) != 1 {
		t.Error("asMap handling")
	}
}
