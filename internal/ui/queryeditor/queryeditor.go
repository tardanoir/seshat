package queryeditor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SQL keyword highlighting (vim mode only)
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
	ansiKeyword = "\x1b[1;34m"  // bold blue
	ansiReset   = "\x1b[22;39m" // reset bold+fg
	ansiDim     = "\x1b[2m"
	ansiDimEnd  = "\x1b[22m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[1;36m"
	ansiFull    = "\x1b[0m"
)

func highlightSQL(s string) string {
	return sqlKeywordRe.ReplaceAllStringFunc(s, func(match string) string {
		return ansiKeyword + match + ansiReset
	})
}

// Model is the query editor, supporting both vim (read-only) and insert (textarea) modes.
type Model struct {
	vimMode bool

	// textarea mode
	ta       textarea.Model
	selRange *[2]int

	// vim mode
	sql     string
	cursor  int
	scrollY int

	// shared
	stmts   []statement
	width   int
	height  int
	focused bool
}

type statement struct {
	sql       string
	startLine int
	endLine   int
}

func New(vimMode bool) Model {
	m := Model{vimMode: vimMode, focused: true}

	if vimMode {
	} else {
		ta := textarea.New()
		ta.Placeholder = "Write a query or press Ctrl+E to open your editor"
		ta.ShowLineNumbers = true
		ta.EndOfBufferCharacter = ' '
		ta.SetHeight(4)
		ta.CharLimit = 0

		// Unbind keys that conflict with app-level keybindings.
		ta.KeyMap.LineEnd = key.NewBinding(key.WithKeys("end"))
		ta.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace"))
		ta.KeyMap.DeleteCharacterForward = key.NewBinding(key.WithKeys("delete"))
		ta.KeyMap.TransposeCharacterBackward = key.NewBinding(key.WithDisabled())
		ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
		ta.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))
		ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("enter"))
		ta.KeyMap.DeleteCharacterBackward = key.NewBinding(key.WithKeys("backspace"))
		ta.KeyMap.LineStart = key.NewBinding(key.WithKeys("home"))
		ta.KeyMap.DeleteBeforeCursor = key.NewBinding(key.WithKeys("alt+backspace"))
		ta.KeyMap.DeleteAfterCursor = key.NewBinding(key.WithDisabled())

		// Statement selection marker via prompt func.
		sel := &[2]int{-1, -1}
		ta.SetPromptFunc(2, func(info textarea.PromptInfo) string {
			if info.LineNumber >= sel[0] && info.LineNumber <= sel[1] {
				return lipgloss.NewStyle().Foreground(style.ColorPrimary).Render("▎") + " "
			}
			return "  "
		})

		// Style: no border on the textarea itself — View() wraps it in a border.
		s := ta.Styles()
		s.Focused.Base = lipgloss.NewStyle()
		s.Blurred.Base = lipgloss.NewStyle()
		s.Focused.EndOfBuffer = lipgloss.NewStyle().Foreground(style.ColorBorder)
		s.Blurred.EndOfBuffer = lipgloss.NewStyle().Foreground(style.ColorBorder)
		s.Focused.LineNumber = lipgloss.NewStyle().Foreground(style.ColorBorder)
		s.Blurred.LineNumber = lipgloss.NewStyle().Foreground(style.ColorBorder)
		s.Focused.CursorLineNumber = lipgloss.NewStyle().Foreground(style.ColorText)
		s.Blurred.CursorLineNumber = lipgloss.NewStyle().Foreground(style.ColorBorder)
		s.Focused.CursorLine = lipgloss.NewStyle()
		s.Blurred.CursorLine = lipgloss.NewStyle()
		ta.SetStyles(s)

		ta.Focus()
		m.ta = ta
		m.selRange = sel
	}

	return m
}

func (m Model) Value() string {
	if m.vimMode {
		return m.sql
	}
	return m.ta.Value()
}

func (m Model) StmtCount() int { return len(m.stmts) }

func (m Model) StmtIndex() int {
	if len(m.stmts) == 0 {
		return 0
	}
	if m.vimMode {
		return m.cursor
	}
	row := m.ta.Line()
	for i, s := range m.stmts {
		if row >= s.startLine && row <= s.endLine {
			return i
		}
	}
	return len(m.stmts) - 1
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
	if !m.vimMode {
		if f {
			m.ta.Focus()
		} else {
			m.ta.Blur()
		}
	}
}

func (m Model) Focused() bool {
	if m.vimMode {
		return m.focused
	}
	return m.ta.Focused()
}

func (m Model) SelectedStatement() string {
	if len(m.stmts) == 0 {
		return m.Value()
	}
	return m.stmts[m.StmtIndex()].sql
}

func (m *Model) SetValue(s string) {
	if m.vimMode {
		m.sql = s
		m.cursor = 0
		m.scrollY = 0
	} else {
		m.ta.SetValue(s)
	}
	m.stmts = parseStatements(s)
	if !m.vimMode {
		m.updateSelRange()
	}
}

func (m *Model) updateSelRange() {
	if m.selRange == nil {
		return
	}
	idx := m.StmtIndex()
	if idx < len(m.stmts) {
		m.selRange[0] = m.stmts[idx].startLine
		m.selRange[1] = m.stmts[idx].endLine
	} else {
		m.selRange[0] = -1
		m.selRange[1] = -1
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	if !m.vimMode {
		m.ta.SetWidth(w - 2)
		m.ta.SetHeight(h - 2)
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.vimMode {
		return m.updateVim(msg)
	}
	return m.updateTextarea(msg)
}

// ── Textarea mode ──────────────────────────────────────────

func (m Model) updateTextarea(msg tea.Msg) (Model, tea.Cmd) {
	if !m.ta.Focused() {
		return m, nil
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	m.stmts = parseStatements(m.ta.Value())
	m.updateSelRange()
	return m, cmd
}

func (m Model) viewTextarea() string {
	s := style.Editor
	if m.ta.Focused() {
		s = style.Focused
	}
	return s.Width(m.width - 2).Render(m.ta.View())
}

// ── Vim mode ───────────────────────────────────────────────

func (m Model) updateVim(msg tea.Msg) (Model, tea.Cmd) {
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
	h := m.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) viewVim() string {
	s := style.Editor
	if m.focused {
		s = style.Focused
	}

	innerH := m.viewHeight()
	innerW := m.width - 4
	if innerW < 10 {
		innerW = 10
	}

	allLines := strings.Split(m.sql, "\n")

	selStart, selEnd := -1, -1
	if len(m.stmts) > 0 && m.cursor < len(m.stmts) {
		selStart = m.stmts[m.cursor].startLine
		selEnd = m.stmts[m.cursor].endLine
	}

	endLine := m.scrollY + innerH
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	var rendered []string
	for i := m.scrollY; i < endLine; i++ {
		line := allLines[i]
		lineNum := ansiDim + padRight(fmt.Sprintf("%d", i+1), 3) + ansiDimEnd

		selected := i >= selStart && i <= selEnd
		if selected {
			lineNum = ansiMagenta + "▎" + ansiFull + lineNum
		} else {
			lineNum = " " + lineNum
		}

		hl := highlightSQL(truncate(line, innerW-6))
		rendered = append(rendered, lineNum+" "+hl)
	}

	for len(rendered) < innerH {
		rendered = append(rendered, " "+ansiDim+"~"+ansiDimEnd)
	}
	rendered = rendered[:innerH]

	content := strings.Join(rendered, "\n")
	return s.Width(m.width - 2).Render(content)
}

func (m Model) View() string {
	if m.vimMode {
		return m.viewVim()
	}
	return m.viewTextarea()
}

// ── Shared ─────────────────────────────────────────────────

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
