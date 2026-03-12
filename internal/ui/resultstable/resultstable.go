package resultstable

import (
	"fmt"
	"strings"

	"seshat/internal/db"
	"seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	width   int
	height  int
	focused bool
	empty   bool
	err     string
	result  *db.QueryResult

	scrollX int // horizontal character offset
	scrollY int // first visible row index
	cursorY int // selected row (absolute index)

	colWidths []int
	cells     [][]string
	totalW    int // total width of all columns + separators
}

func New() Model {
	return Model{empty: true}
}

func (m Model) Focused() bool { return m.focused }

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
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
		m.totalW += (n - 1) * 3 // " | " separators
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
	// height - border(2) - header(1) - separator(1) - footer(1) - extra(1)
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) viewportWidth() int {
	// width - border(2) - padding(2) - safety margin(2)
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
			m.scrollX -= 8
		case "right", "l":
			m.scrollX += 8
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
			m.scrollX = 0
		case "end":
			m.scrollX = m.totalW
		case "g":
			m.cursorY = 0
			m.ensureCursorVisible()
		case "G":
			m.cursorY = len(m.cells) - 1
			m.ensureCursorVisible()
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

// ANSI escape helpers — using raw codes avoids lipgloss double-wrapping issues
const (
	ansiBoldCyan     = "\x1b[1;36m"
	ansiDim          = "\x1b[2m"
	ansiBoldMagBg    = "\x1b[1;35;100m" // bold magenta fg, bright-black bg
	ansiReset        = "\x1b[0m"
	ansiSubtext      = "\x1b[97m" // bright white
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

		// Header — raw ANSI to avoid lipgloss width interference
		headerRow := m.buildRow(m.result.Columns)
		visibleHeader := sliceVisible(headerRow, m.scrollX, vw)

		// Separator
		sepParts := make([]string, len(m.colWidths))
		for i, w := range m.colWidths {
			sepParts[i] = strings.Repeat("-", w)
		}
		fullSep := strings.Join(sepParts, "-+-")
		visibleSep := sliceVisible(fullSep, m.scrollX, vw)

		lines = append(lines, ansiBoldCyan+visibleHeader+ansiReset)
		lines = append(lines, ansiDim+visibleSep+ansiReset)

		// Data rows
		endRow := m.scrollY + vh
		if endRow > len(m.cells) {
			endRow = len(m.cells)
		}
		for i := m.scrollY; i < endRow; i++ {
			fullRow := m.buildRow(m.cells[i])
			visibleRow := sliceVisible(fullRow, m.scrollX, vw)

			if i == m.cursorY && m.focused {
				lines = append(lines, ansiBoldMagBg+visibleRow+ansiReset)
			} else {
				lines = append(lines, visibleRow)
			}
		}

		// Footer
		if m.result != nil {
			rowInfo := fmt.Sprintf(" %d rows", len(m.result.Rows))
			if !m.result.IsSelect {
				rowInfo = fmt.Sprintf(" %d rows affected", m.result.RowsAffected)
			}
			scrollHint := ""
			if m.totalW > vw {
				pct := 0
				maxSX := m.totalW - vw
				if maxSX > 0 {
					pct = m.scrollX * 100 / maxSX
				}
				scrollHint = fmt.Sprintf("  ←→ %d%%", pct)
			}
			rowPos := fmt.Sprintf("  [%d/%d]", m.cursorY+1, len(m.cells))
			lines = append(lines, ansiSubtext+rowInfo+rowPos+scrollHint+ansiReset)
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
