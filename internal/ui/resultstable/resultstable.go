package resultstable

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/tardanoir/seshat/internal/db"
	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
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
		return
	}
	m.empty = false
	m.scrollX = 0
	m.scrollY = 0
	m.cursorX = 0
	m.cursorY = 0

	m.cells = make([][]string, len(result.Rows))
	for i, r := range result.Rows {
		row := make([]string, len(r))
		for j, v := range r {
			row[j] = formatCell(v)
		}
		m.cells[i] = row
	}
	m.computeColumnWidths()
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

func (m Model) viewportHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) viewportWidth() int {
	w := m.width - 6
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
		}
		m.clampScroll()
	}

	return m, nil
}

func (m Model) buildRow(cells []string) string {
	parts := make([]string, len(m.colWidths))
	for i, w := range m.colWidths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = padRight(cell, w)
	}
	return strings.Join(parts, " | ")
}

// ANSI escape helpers
const (
	ansiBoldCyan     = "\x1b[1;36m"
	ansiDim          = "\x1b[2m"
	ansiBoldMagBg    = "\x1b[1;35;100m"
	ansiSelectedCell = "\x1b[1;36;100m"
	ansiReset        = "\x1b[0m"
	ansiSubtext      = "\x1b[97m"
)

func (m Model) View() string {
	borderStyle := style.Results
	if m.focused {
		borderStyle = style.Focused
	}

	innerH := m.height - 2
	if innerH < 1 {
		innerH = 1
	}

	var lines []string
	if m.err != "" {
		lines = append(lines, style.Error.Render("Error: "+m.err))
	} else if m.empty {
		lines = append(lines, style.ListItem.Render("No results. Execute a query with Ctrl+R."))
	} else {
		vw := m.viewportWidth()
		vh := m.viewportHeight()

		// Header with selected column highlight
		headerRow := m.buildRow(m.result.Columns)
		visibleHeader := sliceVisible(headerRow, m.scrollX, vw)

		if m.focused {
			lines = append(lines, m.buildHighlightedRow(m.result.Columns, vw, ansiBoldCyan))
		} else {
			lines = append(lines, ansiBoldCyan+visibleHeader+ansiReset)
		}

		// Separator
		sepParts := make([]string, len(m.colWidths))
		for i, w := range m.colWidths {
			sepParts[i] = strings.Repeat("-", w)
		}
		fullSep := strings.Join(sepParts, "-+-")
		visibleSep := sliceVisible(fullSep, m.scrollX, vw)
		lines = append(lines, ansiDim+visibleSep+ansiReset)

		// Data rows
		endRow := m.scrollY + vh
		if endRow > len(m.cells) {
			endRow = len(m.cells)
		}
		for i := m.scrollY; i < endRow; i++ {
			if i == m.cursorY && m.focused {
				lines = append(lines, m.buildCursorRow(vw))
			} else {
				fullRow := m.buildRow(m.cells[i])
				visibleRow := sliceVisible(fullRow, m.scrollX, vw)
				lines = append(lines, visibleRow)
			}
		}

		// Footer
		if m.result != nil {
			colName := ""
			if m.cursorX < len(m.result.Columns) {
				colName = m.result.Columns[m.cursorX]
			}
			rowInfo := fmt.Sprintf(" %d rows", len(m.result.Rows))
			if !m.result.IsSelect {
				rowInfo = fmt.Sprintf(" %d rows affected", m.result.RowsAffected)
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
			lines = append(lines, ansiSubtext+rowInfo+rowPos+colInfo+scrollHint+ansiReset)
		}
	}

	// Pad to exact inner height
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	lines = lines[:innerH]

	content := strings.Join(lines, "\n")
	return borderStyle.Width(m.width - 2).Render(content)
}

func (m Model) buildCursorRow(vw int) string {
	fullRow := m.buildRow(m.cells[m.cursorY])
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
		return ansiBoldMagBg + before + ansiReset +
			ansiSelectedCell + cell + ansiReset +
			ansiBoldMagBg + after + ansiReset
	}

	return ansiBoldMagBg + visibleRow + ansiReset
}

func (m Model) buildHighlightedRow(cells []string, vw int, baseAnsi string) string {
	fullRow := m.buildRow(cells)
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
		return baseAnsi + before + ansiReset +
			"\x1b[1;36;4m" + cell + ansiReset +
			baseAnsi + after + ansiReset
	}

	return baseAnsi + visibleRow + ansiReset
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

func sliceVisible(s string, offset, width int) string {
	runes := []rune(s)
	if offset >= len(runes) {
		return strings.Repeat(" ", width)
	}
	end := offset + width
	if end > len(runes) {
		end = len(runes)
	}
	result := string(runes[offset:end])
	runeLen := end - offset
	if runeLen < width {
		result += strings.Repeat(" ", width-runeLen)
	}
	return result
}

func padRight(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}
