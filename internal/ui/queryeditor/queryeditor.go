package queryeditor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

// SQL keyword highlighting
var sqlKeywordRe = regexp.MustCompile(
	`(?i)\b(SELECT|FROM|WHERE|INSERT|INTO|UPDATE|DELETE|CREATE|DROP|ALTER|` +
		`TABLE|INDEX|VIEW|JOIN|LEFT|RIGHT|INNER|OUTER|FULL|CROSS|ON|AND|OR|` +
		`NOT|IN|IS|NULL|AS|ORDER|BY|GROUP|HAVING|LIMIT|OFFSET|SET|VALUES|` +
		`DISTINCT|UNION|ALL|EXISTS|BETWEEN|LIKE|ILIKE|CASE|WHEN|THEN|ELSE|` +
		`END|ASC|DESC|WITH|RETURNING|BEGIN|COMMIT|ROLLBACK|TRUE|FALSE|` +
		`DEFAULT|PRIMARY|KEY|FOREIGN|REFERENCES|CONSTRAINT|CHECK|UNIQUE|` +
		`CASCADE|RESTRICT|SCHEMA|GRANT|REVOKE|COUNT|SUM|AVG|MIN|MAX|` +
		`COALESCE|CAST|TEXT|INTEGER|BOOLEAN|VARCHAR|SERIAL|BIGINT|SMALLINT|` +
		`FLOAT|NUMERIC|TIMESTAMP|DATE|TIME|INTERVAL)\b`)

const (
	ansiKeyword  = "\x1b[1;34m"  // bold blue
	ansiReset    = "\x1b[22;39m" // reset bold+fg
	ansiDim      = "\x1b[2m"
	ansiDimEnd   = "\x1b[22m"
	ansiMagenta  = "\x1b[35m"    // magenta for selection marker
	ansiCyan     = "\x1b[1;36m"
	ansiFull     = "\x1b[0m"
)

func highlightSQL(s string) string {
	return sqlKeywordRe.ReplaceAllStringFunc(s, func(match string) string {
		return ansiKeyword + match + ansiReset
	})
}

// Model is a read-only SQL preview pane with statement selection.
type Model struct {
	sql     string
	stmts   []statement
	cursor  int
	scrollY int
	width   int
	height  int
	focused bool
}

type statement struct {
	sql       string
	startLine int
	endLine   int
}

func New() Model {
	m := Model{focused: true}
	m.SetValue("-- Press Ctrl+E to open editor")
	return m
}

func (m Model) Value() string            { return m.sql }
func (m Model) StmtCount() int           { return len(m.stmts) }
func (m Model) StmtIndex() int           { return m.cursor }
func (m *Model) SetFocused(f bool)       { m.focused = f }
func (m Model) Focused() bool            { return m.focused }

func (m Model) SelectedStatement() string {
	if len(m.stmts) == 0 {
		return m.sql
	}
	return m.stmts[m.cursor].sql
}

func (m *Model) SetValue(s string) {
	m.sql = s
	m.stmts = parseStatements(s)
	m.cursor = 0
	m.scrollY = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.stmts)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		}
	}
	return m, nil
}

func (m *Model) ensureVisible() {
	if len(m.stmts) == 0 {
		return
	}
	vh := m.viewHeight()
	stmt := m.stmts[m.cursor]
	if stmt.startLine < m.scrollY {
		m.scrollY = stmt.startLine
	}
	if stmt.endLine >= m.scrollY+vh {
		m.scrollY = stmt.endLine - vh + 1
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}
}

func (m Model) viewHeight() int {
	h := m.height - 2 // border
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) View() string {
	s := style.Editor
	if m.focused {
		s = style.Focused
	}

	innerH := m.viewHeight()
	innerW := m.width - 4 // border(2) + padding(2)
	if innerW < 10 {
		innerW = 10
	}

	allLines := strings.Split(m.sql, "\n")

	// Selection range
	selStart, selEnd := -1, -1
	if len(m.stmts) > 0 && m.cursor < len(m.stmts) {
		selStart = m.stmts[m.cursor].startLine
		selEnd = m.stmts[m.cursor].endLine
	}

	// Render visible lines
	endLine := m.scrollY + innerH
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	var rendered []string
	for i := m.scrollY; i < endLine; i++ {
		line := allLines[i]
		lineNum := ansiDim + padRight(fmt.Sprintf("%d", i+1), 3) + ansiDimEnd

		// Selection: colored gutter marker (▎) on the left
		selected := i >= selStart && i <= selEnd
		if selected {
			lineNum = ansiMagenta + "▎" + ansiFull + lineNum
		} else {
			lineNum = " " + lineNum
		}

		hl := highlightSQL(truncate(line, innerW-6))
		rendered = append(rendered, lineNum+" "+hl)
	}

	// Pad to exact inner height
	for len(rendered) < innerH {
		rendered = append(rendered, " "+ansiDim+"~"+ansiDimEnd)
	}
	rendered = rendered[:innerH]

	// Footer: statement indicator (replaces last line)
	if len(m.stmts) > 1 {
		indicator := fmt.Sprintf(" %d/%d", m.cursor+1, len(m.stmts))
		hint := ""
		if m.focused {
			hint = "  j/k"
		}
		footer := " " + ansiCyan + "stmt" + ansiFull + ansiDim + indicator + hint + ansiDimEnd
		rendered[len(rendered)-1] = footer
	}

	content := strings.Join(rendered, "\n")
	return s.Width(m.width - 2).Render(content)
}

func parseStatements(sql string) []statement {
	allLines := strings.Split(sql, "\n")

	charToLine := make([]int, len(sql)+1)
	lineIdx := 0
	for i, ch := range sql {
		charToLine[i] = lineIdx
		if ch == '\n' {
			lineIdx++
		}
	}
	charToLine[len(sql)] = lineIdx

	var stmts []statement
	start := 0
	inSingle := false
	inDouble := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ';' && !inSingle && !inDouble:
			text := strings.TrimSpace(sql[start : i+1])
			if text != "" && text != ";" {
				stmts = append(stmts, statement{
					sql:       strings.TrimRight(text, "; \t\n"),
					startLine: charToLine[skipWhitespace(sql, start)],
					endLine:   charToLine[i],
				})
			}
			start = i + 1
		}
	}

	if start < len(sql) {
		text := strings.TrimSpace(sql[start:])
		if text != "" {
			stmts = append(stmts, statement{
				sql:       text,
				startLine: charToLine[skipWhitespace(sql, start)],
				endLine:   len(allLines) - 1,
			})
		}
	}

	if len(stmts) == 0 && strings.TrimSpace(sql) != "" {
		stmts = append(stmts, statement{
			sql:       strings.TrimSpace(sql),
			startLine: 0,
			endLine:   len(allLines) - 1,
		})
	}

	return stmts
}

func skipWhitespace(s string, from int) int {
	for from < len(s) && (s[from] == ' ' || s[from] == '\t' || s[from] == '\n' || s[from] == '\r') {
		from++
	}
	if from >= len(s) {
		return len(s) - 1
	}
	return from
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, maxW int) string {
	r := []rune(s)
	if len(r) <= maxW || maxW <= 0 {
		return s
	}
	if maxW < 4 {
		return string(r[:maxW])
	}
	return string(r[:maxW-3]) + "..."
}
