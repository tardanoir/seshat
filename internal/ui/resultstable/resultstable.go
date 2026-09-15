package resultstable

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"

	"github.com/tardanoir/seshat/internal/db"
	"github.com/tardanoir/seshat/internal/ui/style"
)

type CopiedToClipboardMsg struct {
	Label string
	Err   error
}

type Model struct {
	width   int
	height  int
	focused bool
	empty   bool
	err     string
	result  *db.QueryResult

	scrollX int
	scrollY int
	cursorX int
	cursorY int

	colWidths []int
	cells     [][]string
	totalW    int
	separator string
}

func New() Model {
	return Model{empty: true}
}

func (m Model) Focused() bool      { return m.focused }
func (m *Model) SetFocused(f bool) { m.focused = f }
func (m *Model) SetSize(w, h int)  { m.width = w; m.height = h }

func (m *Model) CurrentResult() *db.QueryResult {
	return m.result
}

func (m *Model) SetError(err string) {
	m.err = err
	m.empty = true
	m.result = nil
	m.cells = nil
}

func (m *Model) SetResult(result *db.QueryResult) {
	m.err = ""
	m.result = result
	if result == nil || len(result.Columns) == 0 {
		m.empty = true
		m.cells = nil
		m.separator = ""
		return
	}
	m.empty = false
	m.scrollX = 0
	m.scrollY = 0
	m.cursorX = 0
	m.cursorY = 0

	m.cells = make([][]string, len(result.Rows))
	sampleSize := 200
	resultsSize := len(result.Rows)

	if sampleSize > resultsSize {
		sampleSize = resultsSize
	}

	for i := 0; i < sampleSize; i++ {
		m.cells[i] = formatRow(result.Rows[i])
	}
	m.computeColumnWidths()
}

func formatRow(raw []string) []string {
	row := make([]string, len(raw))
	for j, v := range raw {
		row[j] = formatCell(v)
	}
	return row
}

func (m *Model) ensureFormatted(i int) []string {
	if m.cells[i] == nil {
		m.cells[i] = formatRow(m.result.Rows[i])
	}
	return m.cells[i]
}

func (m *Model) computeColumnWidths() {
	if m.result == nil {
		return
	}
	n := len(m.result.Columns)
	m.colWidths = make([]int, n)

	for i, h := range m.result.Columns {
		m.colWidths[i] = len(h)
	}

	sample := m.cells
	if len(sample) > 200 {
		sample = sample[:200]
	}
	for _, row := range sample {
		for i, cell := range row {
			if i < n && len(cell) > m.colWidths[i] {
				m.colWidths[i] = len(cell)
			}
		}
	}

	for i := range m.colWidths {
		if m.colWidths[i] > 40 {
			m.colWidths[i] = 40
		}
		if m.colWidths[i] < 4 {
			m.colWidths[i] = 4
		}
	}

	m.totalW = 0
	for _, w := range m.colWidths {
		m.totalW += w
	}
	if n > 1 {
		m.totalW += (n - 1) * 3
	}

	sepParts := make([]string, n)
	for i, w := range m.colWidths {
		sepParts[i] = strings.Repeat(" ", w)
	}
	m.separator = strings.Join(sepParts, "   ")
}

func formatCell(v string) string {
	if v == "NULL" {
		return "∅"
	}
	v = strings.ReplaceAll(v, "\r\n", " ")
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	v = strings.ReplaceAll(v, "\t", " ")
	if len(v) > 100 {
		return v[:97] + "..."
	}
	return v
}

// Layout: header line (1) + column headers (1) + data rows + footer (1).
func (m Model) viewportHeight() int {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) viewportWidth() int {
	w := m.width - 4
	if w < 10 {
		w = 10
	}
	return w
}

func (m *Model) ensureCursorVisible() {
	vh := m.viewportHeight()
	if m.cursorY < m.scrollY {
		m.scrollY = m.cursorY
	}
	if m.cursorY >= m.scrollY+vh {
		m.scrollY = m.cursorY - vh + 1
	}
}

func (m *Model) ensureColumnVisible() {
	if len(m.colWidths) == 0 {
		return
	}
	vw := m.viewportWidth()
	colStart := m.columnStart(m.cursorX)
	colEnd := colStart + m.colWidths[m.cursorX]
	if colStart < m.scrollX {
		m.scrollX = colStart
	}
	if colEnd > m.scrollX+vw {
		m.scrollX = colEnd - vw
	}
}

func (m *Model) columnStart(col int) int {
	pos := 0
	for i := 0; i < col && i < len(m.colWidths); i++ {
		pos += m.colWidths[i] + 3
	}
	return pos
}

func (m *Model) clampScroll() {
	maxScrollY := len(m.cells) - m.viewportHeight()
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	if m.scrollY > maxScrollY {
		m.scrollY = maxScrollY
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}

	maxScrollX := m.totalW - m.viewportWidth()
	if maxScrollX < 0 {
		maxScrollX = 0
	}
	if m.scrollX > maxScrollX {
		m.scrollX = maxScrollX
	}
	if m.scrollX < 0 {
		m.scrollX = 0
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused || m.empty {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		vh := m.viewportHeight()

		switch k {
		case "up", "k":
			if m.cursorY > 0 {
				m.cursorY--
				m.ensureCursorVisible()
			}
		case "down", "j":
			if m.cursorY < len(m.cells)-1 {
				m.cursorY++
				m.ensureCursorVisible()
			}
		case "left", "h":
			if m.cursorX > 0 {
				m.cursorX--
				m.ensureColumnVisible()
			}
		case "right", "l":
			if m.cursorX < len(m.colWidths)-1 {
				m.cursorX++
				m.ensureColumnVisible()
			}
		case "pgup":
			m.cursorY -= vh
			if m.cursorY < 0 {
				m.cursorY = 0
			}
			m.ensureCursorVisible()
		case "pgdown":
			m.cursorY += vh
			if m.cursorY >= len(m.cells) {
				m.cursorY = len(m.cells) - 1
			}
			m.ensureCursorVisible()
		case "home":
			m.cursorX = 0
			m.ensureColumnVisible()
		case "end":
			if len(m.colWidths) > 0 {
				m.cursorX = len(m.colWidths) - 1
				m.ensureColumnVisible()
			}
		case "g":
			m.cursorY = 0
			m.ensureCursorVisible()
		case "G":
			m.cursorY = len(m.cells) - 1
			m.ensureCursorVisible()
		case "y":
			if m.cursorY < len(m.result.Rows) && m.cursorX < len(m.result.Rows[m.cursorY]) {
				val := m.result.Rows[m.cursorY][m.cursorX]
				return m, func() tea.Msg {
					err := clipboard.WriteAll(val)
					return CopiedToClipboardMsg{Label: "cell", Err: err}
				}
			}
		case "Y":
			if m.cursorY < len(m.result.Rows) {
				row := strings.Join(m.result.Rows[m.cursorY], "\t")
				return m, func() tea.Msg {
					err := clipboard.WriteAll(row)
					return CopiedToClipboardMsg{Label: "row", Err: err}
				}
			}
		case "c":
			if m.cursorX < len(m.result.Columns) {
				col := m.columnValues(m.cursorX)
				return m, func() tea.Msg {
					err := clipboard.WriteAll(col)
					return CopiedToClipboardMsg{Label: "column", Err: err}
				}
			}
		}
		m.clampScroll()
	}

	return m, nil
}

// ── Mouse ──────────────────────────────────────────────────

// MouseClick moves the cell cursor to the clicked position. x and y are
// relative to the panel's content area; the layout is one header line, one
// column-header line, then the data rows.
func (m Model) MouseClick(x, y int) Model {
	if m.empty || m.result == nil {
		return m
	}
	const headerRows = 2 // panel header + column headers
	row := y - headerRows + m.scrollY
	if row >= 0 && row < len(m.cells) {
		m.cursorY = row
	}
	if col, ok := m.columnAtX(x + m.scrollX); ok {
		m.cursorX = col
	}
	m.ensureCursorVisible()
	m.ensureColumnVisible()
	m.clampScroll()
	return m
}

// columnAtX maps a horizontal offset in the full (unscrolled) row to a column.
func (m Model) columnAtX(x int) (int, bool) {
	if x < 0 {
		return 0, false
	}
	pos := 0
	for i, w := range m.colWidths {
		if x < pos+w {
			return i, true
		}
		pos += w + 3
		if x < pos {
			// Inside the gutter between two columns — treat it as the left one.
			return i, true
		}
	}
	return 0, false
}

// MouseWheel scrolls the rows by delta (positive scrolls down), carrying the
// cursor along so the selection stays on screen.
func (m Model) MouseWheel(delta int) Model {
	if m.empty || m.result == nil {
		return m
	}
	m.cursorY += delta
	if m.cursorY < 0 {
		m.cursorY = 0
	}
	if m.cursorY > len(m.cells)-1 {
		m.cursorY = len(m.cells) - 1
	}
	m.ensureCursorVisible()
	m.clampScroll()
	return m
}

// columnValues joins every raw value in the given column, one per line, so the
// result pastes straight into a spreadsheet or an IN (...) list.
func (m Model) columnValues(col int) string {
	var sb strings.Builder
	for i, row := range m.result.Rows {
		if i > 0 {
			sb.WriteByte('\n')
		}
		// A short row still emits its line, so line N of the clipboard always
		// lines up with row N on screen.
		if col < len(row) {
			sb.WriteString(row[col])
		}
	}
	return sb.String()
}

func (m Model) buildRow(cells []string) string {
	var sb strings.Builder
	sb.Grow(m.totalW)
	for i, w := range m.colWidths {
		if i > 0 {
			sb.WriteString("   ")
		}
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		writePadded(&sb, cell, w)
	}
	return sb.String()
}

// Styles for inline rendering inside the results view. No backgrounds
// on plain text — let the terminal's bg show through. Only selection
// highlights paint a subtle bg for contrast.
var (
	rTextDim = lipgloss.NewStyle().Foreground(style.ColorSubtext)

	rTextMuted = lipgloss.NewStyle().Foreground(style.ColorMuted)

	rText = lipgloss.NewStyle().Foreground(style.ColorText)

	rAccent = lipgloss.NewStyle().
		Foreground(style.ColorPrimary).
		Bold(true)

	rAccentIdle = lipgloss.NewStyle().Foreground(style.ColorSubtext)

	rHeaderCol = lipgloss.NewStyle().Foreground(style.ColorSubtext)

	rCellSelected = lipgloss.NewStyle().
			Foreground(style.ColorPrimary).
			Bold(true).
			Background(style.ColorHighlight)

	rRowSelected = lipgloss.NewStyle().
			Foreground(style.ColorText).
			Background(style.ColorHighlight)

	rColHeaderSelected = lipgloss.NewStyle().
				Foreground(style.ColorPrimary).
				Bold(true).
				Underline(true)
)

func (m Model) renderHeader() string {
	status := "READY"
	styAccent := rAccentIdle
	if m.focused && !m.empty {
		styAccent = rAccent
	}
	if m.err != "" {
		status = "ERROR"
		styAccent = lipgloss.NewStyle().
			Foreground(style.ColorError).
			Bold(true)
	} else if !m.empty && m.result != nil {
		if m.result.IsSelect {
			status = fmt.Sprintf("%d ROWS · %s", len(m.result.Rows), strings.ToUpper(m.result.Duration.String()))
		} else {
			status = fmt.Sprintf("%d AFFECTED · %s", m.result.RowsAffected, strings.ToUpper(m.result.Duration.String()))
		}
	}
	return rTextDim.Render("RESULTS") + rTextMuted.Render(" · ") + styAccent.Render(status)
}

func (m Model) View() string {
	innerH := m.height - 1 // minus 1 for header
	if innerH < 1 {
		innerH = 1
	}

	var lines []string
	lines = append(lines, m.renderHeader())

	if m.err != "" {
		lines = append(lines, style.Error.Render("Error: "+m.err))
	} else if m.empty {
		lines = append(lines, rTextDim.Render("No results. Execute a query with Ctrl+R."))
	} else {
		vw := m.viewportWidth()
		vh := m.viewportHeight()

		// Column header row
		if m.focused {
			lines = append(lines, m.buildHeaderRow(vw))
		} else {
			headerRow := m.buildRow(m.result.Columns)
			visibleHeader := sliceVisible(headerRow, m.scrollX, vw)
			lines = append(lines, rHeaderCol.Render(visibleHeader))
		}

		// Data rows
		endRow := m.scrollY + vh
		if endRow > len(m.cells) {
			endRow = len(m.cells)
		}
		for i := m.scrollY; i < endRow; i++ {
			if i == m.cursorY && m.focused {
				lines = append(lines, m.buildCursorRow(vw))
			} else {
				fullRow := m.buildRow(m.ensureFormatted(i))
				visibleRow := sliceVisible(fullRow, m.scrollX, vw)
				lines = append(lines, rText.Render(visibleRow))
			}
		}

		// Footer
		if m.result != nil {
			colName := ""
			if m.cursorX < len(m.result.Columns) {
				colName = m.result.Columns[m.cursorX]
			}
			rowInfo := fmt.Sprintf("%d rows", len(m.result.Rows))
			if !m.result.IsSelect {
				rowInfo = fmt.Sprintf("%d rows affected", m.result.RowsAffected)
			}
			rowPos := fmt.Sprintf("  [%d/%d]", m.cursorY+1, len(m.cells))
			colInfo := fmt.Sprintf("  col:%s", colName)
			scrollHint := ""
			if m.totalW > vw {
				pct := 0
				maxSX := m.totalW - vw
				if maxSX > 0 {
					pct = m.scrollX * 100 / maxSX
				}
				scrollHint = fmt.Sprintf("  ←→%d%%", pct)
			}
			lines = append(lines, rTextDim.Render(rowInfo+rowPos+colInfo+scrollHint))
		}
	}

	for len(lines) < innerH+1 {
		lines = append(lines, "")
	}
	lines = lines[:innerH+1]

	content := strings.Join(lines, "\n")
	content = style.PrefixFocusBar(content, m.focused)
	return style.Results.Width(m.width).Height(m.height).MaxHeight(m.height).Render(content)
}

func (m *Model) buildCursorRow(vw int) string {
	fullRow := m.buildRow(m.ensureFormatted(m.cursorY))
	visibleRow := sliceVisible(fullRow, m.scrollX, vw)
	runes := []rune(visibleRow)

	colStart := m.columnStart(m.cursorX) - m.scrollX
	colEnd := colStart + m.colWidths[m.cursorX]

	if colStart < 0 {
		colStart = 0
	}
	if colEnd > vw {
		colEnd = vw
	}
	if colEnd > len(runes) {
		colEnd = len(runes)
	}

	if colStart < colEnd && colStart < len(runes) {
		before := string(runes[:colStart])
		cell := string(runes[colStart:colEnd])
		after := string(runes[colEnd:])
		return rRowSelected.Render(before) +
			rCellSelected.Render(cell) +
			rRowSelected.Render(after)
	}

	return rRowSelected.Render(visibleRow)
}

func (m Model) buildHeaderRow(vw int) string {
	fullRow := m.buildRow(m.result.Columns)
	visibleRow := sliceVisible(fullRow, m.scrollX, vw)
	runes := []rune(visibleRow)

	colStart := m.columnStart(m.cursorX) - m.scrollX
	colEnd := colStart + m.colWidths[m.cursorX]

	if colStart < 0 {
		colStart = 0
	}
	if colEnd > vw {
		colEnd = vw
	}
	if colEnd > len(runes) {
		colEnd = len(runes)
	}

	if colStart < colEnd && colStart < len(runes) {
		before := string(runes[:colStart])
		cell := string(runes[colStart:colEnd])
		after := string(runes[colEnd:])
		return rHeaderCol.Render(before) +
			rColHeaderSelected.Render(cell) +
			rHeaderCol.Render(after)
	}

	return rHeaderCol.Render(visibleRow)
}

func (m Model) ResultSummary() string {
	if m.result == nil {
		return ""
	}
	if m.result.IsSelect {
		return fmt.Sprintf("%d rows in %s", len(m.result.Rows), m.result.Duration)
	}
	return fmt.Sprintf("%d affected in %s", m.result.RowsAffected, m.result.Duration)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func sliceVisible(s string, offset, width int) string {
	if isASCII(s) {
		if offset >= len(s) {
			return strings.Repeat(" ", width)
		}
		end := offset + width
		if end > len(s) {
			end = len(s)
		}
		result := s[offset:end]
		if pad := width - (end - offset); pad > 0 {
			return result + strings.Repeat(" ", pad)
		}
		return result
	}
	runes := []rune(s)
	if offset >= len(runes) {
		return strings.Repeat(" ", width)
	}
	end := offset + width
	if end > len(runes) {
		end = len(runes)
	}
	result := string(runes[offset:end])
	if pad := width - (end - offset); pad > 0 {
		result += strings.Repeat(" ", pad)
	}
	return result
}

func writePadded(sb *strings.Builder, s string, width int) {
	if isASCII(s) {
		if len(s) >= width {
			sb.WriteString(s[:width])
		} else {
			sb.WriteString(s)
			for i := len(s); i < width; i++ {
				sb.WriteByte(' ')
			}
		}
		return
	}
	r := []rune(s)
	if len(r) >= width {
		sb.WriteString(string(r[:width]))
	} else {
		sb.WriteString(s)
		for i := len(r); i < width; i++ {
			sb.WriteByte(' ')
		}
	}
}
