package nvimbridge

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// cell is a single grid cell: the (possibly multi-rune) glyph and the id of the
// highlight group it was painted with. An empty text marks the trailing half of
// a double-width glyph and is skipped when rendering.
type cell struct {
	text string
	hl   int
}

// hlAttr is a resolved highlight definition from hl_attr_define. Color fields
// hold 24-bit RGB values, or -1 when the attribute is unset (inherit default).
type hlAttr struct {
	fg, bg, sp                       int
	bold, italic, underline, reverse bool
}

// handleRedraw is the "redraw" RPC notification handler. Neovim batches many
// events into a single notification; they are applied in order under gridMu so
// the renderer never observes a half-applied frame. A repaint is signalled only
// on "flush", which marks the end of a coherent frame.
func (b *Bridge) handleRedraw(updates ...[]any) {
	b.gridMu.Lock()
	flush := false
	for _, u := range updates {
		if len(u) == 0 {
			continue
		}
		name := asString(u[0])
		args := u[1:]
		switch name {
		case "grid_resize":
			for _, e := range args {
				b.gridResize(asSlice(e))
			}
		case "grid_clear":
			b.gridClear()
		case "grid_line":
			for _, e := range args {
				b.gridLine(asSlice(e))
			}
		case "grid_scroll":
			for _, e := range args {
				b.gridScroll(asSlice(e))
			}
		case "grid_cursor_goto":
			for _, e := range args {
				b.cursorGoto(asSlice(e))
			}
		case "default_colors_set":
			for _, e := range args {
				b.defaultColors(asSlice(e))
			}
		case "hl_attr_define":
			for _, e := range args {
				b.hlAttrDefine(asSlice(e))
			}
		case "mode_change":
			for _, e := range args {
				b.modeChange(asSlice(e))
			}
		case "flush":
			flush = true
		}
	}
	b.gridMu.Unlock()
	if flush {
		b.signalRedraw()
	}
}

func (b *Bridge) signalRedraw() {
	select {
	case b.redraw <- struct{}{}:
	default:
	}
}

func (b *Bridge) allocGrid(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	cells := make([][]cell, h)
	for r := range cells {
		cells[r] = make([]cell, w)
		for c := range cells[r] {
			cells[r][c] = cell{text: " "}
		}
	}
	for r := 0; r < h && r < b.rows; r++ {
		for c := 0; c < w && c < b.cols; c++ {
			cells[r][c] = b.cells[r][c]
		}
	}
	b.cells = cells
	b.cols = w
	b.rows = h
}

func (b *Bridge) gridResize(a []any) {
	if len(a) < 3 {
		return
	}
	b.allocGrid(asInt(a[1]), asInt(a[2]))
}

func (b *Bridge) gridClear() {
	for r := range b.cells {
		for c := range b.cells[r] {
			b.cells[r][c] = cell{text: " "}
		}
	}
}

func (b *Bridge) gridLine(a []any) {
	if len(a) < 4 {
		return
	}
	row, col := asInt(a[1]), asInt(a[2])
	if row < 0 || row >= b.rows {
		return
	}
	hl := 0
	for _, cv := range asSlice(a[3]) {
		cc := asSlice(cv)
		if len(cc) == 0 {
			continue
		}
		txt := asString(cc[0])
		if len(cc) >= 2 {
			hl = asInt(cc[1])
		}
		rep := 1
		if len(cc) >= 3 {
			rep = asInt(cc[2])
		}
		for k := 0; k < rep; k++ {
			if col >= 0 && col < b.cols {
				b.cells[row][col] = cell{text: txt, hl: hl}
			}
			col++
		}
	}
}

func (b *Bridge) gridScroll(a []any) {
	if len(a) < 7 {
		return
	}
	top, bot, left, right, rows := asInt(a[1]), asInt(a[2]), asInt(a[3]), asInt(a[4]), asInt(a[5])
	if left < 0 {
		left = 0
	}
	if right > b.cols {
		right = b.cols
	}
	if left >= right {
		return
	}
	if rows > 0 { // text moves up
		for r := top; r < bot-rows; r++ {
			if r < 0 || r+rows >= b.rows {
				continue
			}
			copy(b.cells[r][left:right], b.cells[r+rows][left:right])
		}
	} else if rows < 0 { // text moves down
		for r := bot - 1; r >= top-rows; r-- {
			if r >= b.rows || r+rows < 0 {
				continue
			}
			copy(b.cells[r][left:right], b.cells[r+rows][left:right])
		}
	}
}

func (b *Bridge) cursorGoto(a []any) {
	if len(a) < 3 {
		return
	}
	b.curRow, b.curCol = asInt(a[1]), asInt(a[2])
}

func (b *Bridge) defaultColors(a []any) {
	if len(a) < 2 {
		return
	}
	b.defFg = asInt(a[0])
	b.defBg = asInt(a[1])
	if len(a) >= 3 {
		b.defSp = asInt(a[2])
	}
}

func (b *Bridge) hlAttrDefine(a []any) {
	if len(a) < 2 {
		return
	}
	id := asInt(a[0])
	m := asMap(a[1])
	h := hlAttr{fg: -1, bg: -1, sp: -1}
	if v, ok := m["foreground"]; ok {
		h.fg = asInt(v)
	}
	if v, ok := m["background"]; ok {
		h.bg = asInt(v)
	}
	if v, ok := m["special"]; ok {
		h.sp = asInt(v)
	}
	h.reverse = asBool(m["reverse"])
	h.bold = asBool(m["bold"])
	h.italic = asBool(m["italic"])
	h.underline = asBool(m["underline"]) || asBool(m["undercurl"]) ||
		asBool(m["underdouble"]) || asBool(m["underdotted"]) || asBool(m["underdashed"])
	b.hls[id] = h
}

func (b *Bridge) modeChange(a []any) {
	if len(a) < 1 {
		return
	}
	b.mode = asString(a[0])
}

// Render produces an h-row, w-column block reflecting the latest flushed grid,
// styled with Neovim's own colors. When focused, the cell under the cursor is
// drawn with reversed video to act as a block cursor.
func (b *Bridge) Render(w, h int, focused bool) string {
	b.gridMu.Lock()
	defer b.gridMu.Unlock()

	base := b.baseStyle()
	var sb strings.Builder
	for r := 0; r < h; r++ {
		if r > 0 {
			sb.WriteByte('\n')
		}
		if r >= b.rows {
			if w > 0 {
				sb.WriteString(base.Render(strings.Repeat(" ", w)))
			}
			continue
		}
		sb.WriteString(b.renderRow(r, w, focused, base))
	}
	return sb.String()
}

func (b *Bridge) renderRow(r, w int, focused bool, base lipgloss.Style) string {
	row := b.cells[r]
	var out strings.Builder
	var run strings.Builder
	runHL, runCursor := -1, false
	width := 0

	flush := func() {
		if run.Len() == 0 {
			return
		}
		out.WriteString(b.styleFor(runHL, runCursor).Render(run.String()))
		run.Reset()
	}

	for c := 0; c < b.cols && width < w; c++ {
		cur := row[c]
		if cur.text == "" { // trailing half of a wide glyph
			continue
		}
		isCursor := focused && r == b.curRow && c == b.curCol
		if cur.hl != runHL || isCursor != runCursor {
			flush()
			runHL, runCursor = cur.hl, isCursor
		}
		run.WriteString(cur.text)
		width += lipgloss.Width(cur.text)
	}
	flush()

	if width < w {
		out.WriteString(base.Render(strings.Repeat(" ", w-width)))
	}
	return out.String()
}

func (b *Bridge) hlOf(id int) hlAttr {
	if h, ok := b.hls[id]; ok {
		return h
	}
	return hlAttr{fg: -1, bg: -1, sp: -1}
}

func (b *Bridge) styleFor(id int, cursor bool) lipgloss.Style {
	h := b.hlOf(id)
	reverse := h.reverse
	if cursor {
		reverse = !reverse
	}

	// Resolved colors, falling back to the editor defaults.
	fg := h.fg
	if fg < 0 {
		fg = b.defFg
	}
	bg := h.bg
	if bg < 0 {
		bg = b.defBg
	}

	s := lipgloss.NewStyle()
	if reverse {
		// Reversed/cursor cells must paint both sides to stay visible.
		if bg >= 0 {
			s = s.Foreground(rgbColor(bg))
		}
		if fg >= 0 {
			s = s.Background(rgbColor(fg))
		}
	} else {
		if fg >= 0 {
			s = s.Foreground(rgbColor(fg))
		}
		// Only paint an explicit background (selection, search, popup, ...).
		// Cells on the editor's default background stay transparent so the
		// panel blends with the app's own theme.
		if h.bg >= 0 {
			s = s.Background(rgbColor(h.bg))
		}
	}
	if h.bold {
		s = s.Bold(true)
	}
	if h.italic {
		s = s.Italic(true)
	}
	if h.underline {
		s = s.Underline(true)
	}
	return s
}

// baseStyle is used for padding and below-buffer rows. It is intentionally
// transparent so those areas match the app's background.
func (b *Bridge) baseStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

func rgbColor(v int) color.Color {
	return lipgloss.Color(fmt.Sprintf("#%06x", v&0xffffff))
}

// ── msgpack any helpers ──────────────────────────────
// The go-client decodes into any as: int64/uint64 (ints), float64,
// bool, string, []any, map[string]any.

func asInt(v any) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case uint64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	case uint32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func asString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if mm, ok := v.(map[any]any); ok {
		out := make(map[string]any, len(mm))
		for k, val := range mm {
			out[asString(k)] = val
		}
		return out
	}
	return nil
}
