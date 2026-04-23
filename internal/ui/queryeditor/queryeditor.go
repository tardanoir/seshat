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

var (
	keywordStyle = lipgloss.NewStyle().
			Foreground(style.ColorCyan).
			Bold(true)

	lineNumStyle = lipgloss.NewStyle().
			Foreground(style.ColorMuted)

	markerStyle = lipgloss.NewStyle().
			Foreground(style.ColorPrimary)

	tildeStyle = lipgloss.NewStyle().
			Foreground(style.ColorMuted)

	textStyle = lipgloss.NewStyle().
			Foreground(style.ColorText)

	headerLabel = lipgloss.NewStyle().
			Foreground(style.ColorSubtext)

	headerLabelFocused = lipgloss.NewStyle().
				Foreground(style.ColorText).
				Bold(true)
)

func highlightSQL(s string) string {
	return sqlKeywordRe.ReplaceAllStringFunc(s, func(match string) string {
		return keywordStyle.Render(match)
	})
}

type Model struct {
	vimMode bool

	ta       textarea.Model
	selRange *[2]int

	sql     string
	cursor  int
	scrollY int

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

	if !vimMode {
		ta := textarea.New()
		ta.Placeholder = "Write a query or press Ctrl+E to open your editor"
		ta.ShowLineNumbers = true
		ta.EndOfBufferCharacter = ' '
		ta.SetHeight(4)
		ta.CharLimit = 0

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

		sel := &[2]int{-1, -1}
		ta.SetPromptFunc(2, func(info textarea.PromptInfo) string {
			if info.LineNumber >= sel[0] && info.LineNumber <= sel[1] {
				return markerStyle.Render("▎") + " "
			}
			return "  "
		})

		s := ta.Styles()
		empty := lipgloss.NewStyle()
		text := lipgloss.NewStyle().Foreground(style.ColorText)
		placeholder := lipgloss.NewStyle().Foreground(style.ColorMuted)

		s.Focused.Base = empty
		s.Blurred.Base = empty
		s.Focused.Text = text
		s.Blurred.Text = text
		s.Focused.Placeholder = placeholder
		s.Blurred.Placeholder = placeholder
		s.Focused.Prompt = empty
		s.Blurred.Prompt = empty
		s.Focused.EndOfBuffer = tildeStyle
		s.Blurred.EndOfBuffer = tildeStyle
		s.Focused.LineNumber = lineNumStyle
		s.Blurred.LineNumber = lineNumStyle
		s.Focused.CursorLineNumber = lipgloss.NewStyle().Foreground(style.ColorText)
		s.Blurred.CursorLineNumber = lineNumStyle
		s.Focused.CursorLine = empty
		s.Blurred.CursorLine = empty
		s.Cursor.Color = style.ColorPrimary
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
		// inner width = w - hPad ; inner height = h - 1 (header row)
		m.ta.SetWidth(w - 4)
		m.ta.SetHeight(h - 1)
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.vimMode {
		return m.updateVim(msg)
	}
	return m.updateTextarea(msg)
}

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
	vh := m.contentHeight()
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

func (m Model) contentHeight() int {
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) contentWidth() int {
	w := m.width - 4
	if w < 10 {
		w = 10
	}
	return w
}

func (m Model) renderHeader() string {
	if m.Focused() {
		return headerLabelFocused.Render("EDITOR")
	}
	return headerLabel.Render("EDITOR")
}

func (m Model) viewVim() string {
	innerH := m.contentHeight()
	innerW := m.contentWidth()

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
		numLabel := padRight(fmt.Sprintf("%d", i+1), 3)
		selected := i >= selStart && i <= selEnd

		var prefix string
		if selected {
			prefix = markerStyle.Render("▎") + lineNumStyle.Render(numLabel)
		} else {
			prefix = " " + lineNumStyle.Render(numLabel)
		}

		hl := highlightSQL(truncate(line, innerW-6))
		rendered = append(rendered, prefix+" "+textStyle.Render(hl))
	}

	for len(rendered) < innerH {
		rendered = append(rendered, " "+tildeStyle.Render("~"))
	}
	rendered = rendered[:innerH]

	header := m.renderHeader()
	body := strings.Join(rendered, "\n")
	content := header + "\n" + body
	content = style.PrefixFocusBar(content, m.focused)

	return style.Editor.Width(m.width).Height(m.height).MaxHeight(m.height).Render(content)
}

func (m Model) viewTextarea() string {
	header := m.renderHeader()
	body := m.ta.View()
	content := header + "\n" + body
	content = style.PrefixFocusBar(content, m.ta.Focused())
	return style.Editor.Width(m.width).Height(m.height).MaxHeight(m.height).Render(content)
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
