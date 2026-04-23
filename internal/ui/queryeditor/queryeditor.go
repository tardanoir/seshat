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
	"github.com/charmbracelet/x/ansi"
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

	popupBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(style.ColorBorder)

	popupRow = lipgloss.NewStyle().
			Foreground(style.ColorText)

	popupSelectedRow = lipgloss.NewStyle().
				Foreground(style.ColorBg).
				Background(style.ColorPrimary).
				Bold(true)

	kindKeywordStyle = lipgloss.NewStyle().Foreground(style.ColorCyan)
	kindTableStyle   = lipgloss.NewStyle().Foreground(style.ColorPrimary)
	kindColumnStyle  = lipgloss.NewStyle().Foreground(style.ColorSubtext)
)

func kindStyle(k SuggestionKind) lipgloss.Style {
	switch k {
	case KindTable:
		return kindTableStyle
	case KindColumn:
		return kindColumnStyle
	default:
		return kindKeywordStyle
	}
}

var (
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

	completer *provider
	comp      completion
}

type completion struct {
	open      bool
	items     []Suggestion
	selected  int
	prefix    string
	wordStart int
}

type statement struct {
	sql       string
	startLine int
	endLine   int
}

func New(vimMode bool) Model {
	m := Model{vimMode: vimMode, focused: true, completer: newProvider()}

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
			m.closeCompletion()
		}
	}
}

func (m Model) Focused() bool {
	if m.vimMode {
		return m.focused
	}
	return m.ta.Focused()
}

func (m Model) CompletionOpen() bool { return m.comp.open }

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

func (m *Model) SetSchema(tables []TableRef, columns []ColumnRef) {
	if m.completer == nil {
		m.completer = newProvider()
	}
	m.completer.setSchema(tables, columns)
	m.refreshCompletion()
}

const completionLimit = 8

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
	if k, ok := msg.(tea.KeyMsg); ok && m.comp.open {
		switch k.String() {
		case "tab":
			m.expandOrAccept()
			return m, nil
		case "enter":
			m.acceptSelected()
			return m, nil
		case "esc":
			m.closeCompletion()
			return m, nil
		case "up", "ctrl+p":
			if m.comp.selected > 0 {
				m.comp.selected--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.comp.selected < len(m.comp.items)-1 {
				m.comp.selected++
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	m.stmts = parseStatements(m.ta.Value())
	m.updateSelRange()
	m.refreshCompletion()
	return m, cmd
}

func (m *Model) closeCompletion() {
	m.comp = completion{}
}

func (m *Model) expandOrAccept() {
	if !m.comp.open || len(m.comp.items) == 0 {
		return
	}
	lcp := longestCommonPrefix(m.comp.items)
	if len(lcp) > len(m.comp.prefix) {
		suffix := lcp[len(m.comp.prefix):]
		m.ta.InsertString(suffix)
		m.stmts = parseStatements(m.ta.Value())
		m.updateSelRange()
		m.refreshCompletion()
		return
	}
	m.acceptSelected()
}

func (m *Model) acceptSelected() {
	if !m.comp.open || len(m.comp.items) == 0 {
		return
	}
	pick := m.comp.items[m.comp.selected].Text
	suffix := pick
	if strings.HasPrefix(strings.ToLower(pick), strings.ToLower(m.comp.prefix)) {
		suffix = pick[len(m.comp.prefix):]
	}
	m.ta.InsertString(suffix)
	m.closeCompletion()
	m.stmts = parseStatements(m.ta.Value())
	m.updateSelRange()
}

func longestCommonPrefix(items []Suggestion) string {
	if len(items) == 0 {
		return ""
	}
	first := items[0].Text
	firstLo := strings.ToLower(first)
	maxLen := len(firstLo)
	for _, it := range items[1:] {
		lo := strings.ToLower(it.Text)
		i := 0
		for i < maxLen && i < len(lo) && firstLo[i] == lo[i] {
			i++
		}
		maxLen = i
		if maxLen == 0 {
			return ""
		}
	}
	return first[:maxLen]
}

func (m *Model) refreshCompletion() {
	if m.vimMode || m.completer == nil {
		m.closeCompletion()
		return
	}
	row := m.ta.Line()
	lines := strings.Split(m.ta.Value(), "\n")
	if row < 0 || row >= len(lines) {
		m.closeCompletion()
		return
	}
	col := m.ta.LineInfo().ColumnOffset + m.ta.LineInfo().StartColumn
	line := lines[row]
	if !cursorAtWordEnd(line, col) {
		m.closeCompletion()
		return
	}
	word, wordStart := wordBeforeCursor(line, col)
	if word == "" {
		m.closeCompletion()
		return
	}

	var sb strings.Builder
	for i := 0; i < row; i++ {
		sb.WriteString(lines[i])
		sb.WriteByte('\n')
	}
	curLineRunes := []rune(line)
	if wordStart > len(curLineRunes) {
		wordStart = len(curLineRunes)
	}
	sb.WriteString(string(curLineRunes[:wordStart]))

	ctx := sqlContextAt(sb.String())
	items := m.completer.suggest(word, ctx, completionLimit)
	if len(items) == 0 {
		m.closeCompletion()
		return
	}
	selected := 0
	if m.comp.open && m.comp.prefix == word {
		if m.comp.selected < len(items) {
			selected = m.comp.selected
		}
	}
	m.comp = completion{
		open:      true,
		items:     items,
		selected:  selected,
		prefix:    word,
		wordStart: wordStart,
	}
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
	if m.comp.open {
		body = m.overlayPopup(body)
	}
	content := header + "\n" + body
	content = style.PrefixFocusBar(content, m.ta.Focused())
	return style.Editor.Width(m.width).Height(m.height).MaxHeight(m.height).Render(content)
}

func (m Model) cursorVisualPos() (int, int) {
	li := m.ta.LineInfo()
	digits := 2
	if lc := m.ta.LineCount(); lc >= 100 {
		digits = 3
	}
	const promptW = 2
	lineNumW := digits + 2 // " N " format
	x := promptW + lineNumW + li.CharOffset
	y := m.ta.Line()
	return x, y
}

func (m Model) overlayPopup(body string) string {
	popup := m.renderPopup()
	if popup == "" {
		return body
	}
	popupLines := strings.Split(popup, "\n")
	popupH := len(popupLines)
	popupW := 0
	for _, pl := range popupLines {
		if w := lipgloss.Width(pl); w > popupW {
			popupW = w
		}
	}

	lines := strings.Split(body, "\n")
	bodyH := len(lines)
	contentW := m.contentWidth()
	cx, cy := m.cursorVisualPos()

	anchorX := cx - 1
	if anchorX < 0 {
		anchorX = 0
	}
	if anchorX+popupW > contentW {
		anchorX = contentW - popupW
	}
	if anchorX < 0 {
		anchorX = 0
	}

	anchorY := cy + 1
	if anchorY+popupH > bodyH {
		anchorY = cy - popupH
	}
	if anchorY < 0 {
		anchorY = 0
	}
	if anchorY >= bodyH {
		return body
	}

	for i, pl := range popupLines {
		y := anchorY + i
		if y >= bodyH {
			break
		}
		orig := lines[y]
		prefix := ansi.Truncate(orig, anchorX, "")
		prefixW := lipgloss.Width(prefix)
		if prefixW < anchorX {
			prefix += strings.Repeat(" ", anchorX-prefixW)
		}
		lines[y] = prefix + pl
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderPopup() string {
	if !m.comp.open || len(m.comp.items) == 0 {
		return ""
	}
	maxName := 0
	for _, it := range m.comp.items {
		if w := lipgloss.Width(it.Text); w > maxName {
			maxName = w
		}
	}
	const maxRowW = 38
	const kindW = 3
	rowW := maxName + 2 + kindW + 2
	if rowW > maxRowW {
		rowW = maxRowW
		maxName = rowW - 2 - kindW - 2
	}

	var rows []string
	for i, it := range m.comp.items {
		name := it.Text
		if len([]rune(name)) > maxName {
			name = string([]rune(name)[:maxName])
		}
		pad := maxName - lipgloss.Width(name)
		if pad < 0 {
			pad = 0
		}
		label := " " + name + strings.Repeat(" ", pad) + " "
		kind := it.Kind.Label()
		kindStr := kindStyle(it.Kind).Render(kind)
		right := " " + kindStr
		line := label + right
		if i == m.comp.selected {
			line = popupSelectedRow.Render(line)
		} else {
			line = popupRow.Render(line)
		}
		rows = append(rows, line)
	}
	return popupBox.Render(strings.Join(rows, "\n"))
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
