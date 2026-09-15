package sidebar

import (
	"fmt"
	"strings"

	"github.com/tardanoir/seshat/internal/query"
	"github.com/tardanoir/seshat/internal/ui/filter"
	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

type SelectQueryMsg struct{ Content string }
type SelectTemplateMsg struct{ Template query.Template }
type SelectHistoryMsg struct{ SQL string }
type DeleteQueryMsg struct{ Name string }
type RequestColumnsMsg struct{ Schema, TableName string }

// SelectTableMsg is emitted when a table row is clicked, so the app can drop a
// starter statement for it into the editor.
type SelectTableMsg struct{ Schema, Name string }

type ColumnDef struct {
	Name     string
	DataType string
	Nullable bool
}

type TableEntry struct {
	Schema   string
	Name     string
	Columns  []ColumnDef
	Expanded bool
}

func (t TableEntry) DisplayName() string {
	if t.Schema != "" && t.Schema != "public" {
		return t.Schema + "." + t.Name
	}
	return "public." + t.Name
}

type Section int

const (
	SectionQueries Section = iota
	SectionTemplates
	SectionTables
	SectionHistory
	sectionCount = 4
)

type Model struct {
	width   int
	height  int
	focused bool

	connName string
	connDB   string

	queries   []query.SavedQuery
	templates []query.Template
	tables    []TableEntry
	history   []query.HistoryEntry

	activeSection Section
	cursor        int
	scrollY       int
	search        filter.Model
}

func New() Model {
	return Model{activeSection: SectionTables}
}

func (m Model) Focused() bool      { return m.focused }
func (m *Model) SetFocused(f bool) { m.focused = f }
func (m *Model) SetSize(w, h int)  { m.width = w; m.height = h }

func (m *Model) SetConnection(name, db string) {
	m.connName = name
	m.connDB = db
}

func (m *Model) SetQueries(q []query.SavedQuery)   { m.queries = q }
func (m *Model) SetTemplates(t []query.Template)   { m.templates = t }
func (m *Model) SetTables(tables []TableEntry)     { m.tables = tables }
func (m *Model) SetHistory(h []query.HistoryEntry) { m.history = h }

func (m *Model) SetTableColumns(schema, tableName string, cols []ColumnDef) {
	for i := range m.tables {
		if m.tables[i].Schema == schema && m.tables[i].Name == tableName {
			m.tables[i].Columns = cols
			m.tables[i].Expanded = true
			return
		}
	}
}

// CacheTableColumns populates a table's columns without changing its expansion
// state — used for background prefetch so the sidebar doesn't auto-open.
func (m *Model) CacheTableColumns(schema, tableName string, cols []ColumnDef) {
	for i := range m.tables {
		if m.tables[i].Schema == schema && m.tables[i].Name == tableName {
			m.tables[i].Columns = cols
			return
		}
	}
}

func (m Model) sectionItemCount(sec Section) int {
	switch sec {
	case SectionQueries:
		return len(m.visibleQueries())
	case SectionTemplates:
		return len(m.visibleTemplates())
	case SectionTables:
		n := 0
		for _, ti := range m.visibleTableIdx() {
			n++
			if m.tables[ti].Expanded {
				n += len(m.tables[ti].Columns)
			}
		}
		return n
	case SectionHistory:
		return len(m.visibleHistory())
	}
	return 0
}

// visibleQueries / visibleTemplates / visibleHistory return the entries of the
// active section matching the current search (all of them when not searching).
// The search only ever applies to the active section (it is cleared on switch).

func (m Model) visibleQueries() []query.SavedQuery {
	if !m.search.Active() {
		return m.queries
	}
	out := make([]query.SavedQuery, 0, len(m.queries))
	for _, q := range m.queries {
		if m.search.Matches(q.Name) {
			out = append(out, q)
		}
	}
	return out
}

func (m Model) visibleTemplates() []query.Template {
	if !m.search.Active() {
		return m.templates
	}
	out := make([]query.Template, 0, len(m.templates))
	for _, t := range m.templates {
		if m.search.Matches(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

func (m Model) visibleHistory() []query.HistoryEntry {
	if !m.search.Active() {
		return m.history
	}
	out := make([]query.HistoryEntry, 0, len(m.history))
	for _, h := range m.history {
		if m.search.Matches(h.SQL) {
			out = append(out, h)
		}
	}
	return out
}

// visibleTableIdx returns indices into m.tables for the tables matching the
// search, preserving order. Indices (not copies) so callers can mutate the
// underlying entries (e.g. toggling expansion).
func (m Model) visibleTableIdx() []int {
	out := make([]int, 0, len(m.tables))
	for i, t := range m.tables {
		if !m.search.Active() || m.search.Matches(t.DisplayName()) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) emptyLabel() string {
	if m.search.Active() && m.search.Query() != "" {
		return "(no matches)"
	}
	return "(none)"
}

func (m *Model) switchSection(sec Section) {
	m.activeSection = sec
	m.cursor = 0
	m.scrollY = 0
	m.search.Clear()
}

func (m *Model) moveDown() {
	mx := m.sectionItemCount(m.activeSection) - 1
	if mx < 0 {
		mx = 0
	}
	if m.cursor < mx {
		m.cursor++
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if m.search.Active() {
			// Search mode: arrows navigate the filtered list, enter/ctrl+d act on
			// it, esc exits search; every other printable key edits the query
			// (so digits and j/k are typed, not treated as commands).
			switch k {
			case "esc":
				m.search.Clear()
				m.cursor = 0
				m.scrollY = 0
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				m.moveDown()
			case "enter":
				return m.handleEnter()
			case "ctrl+d":
				return m.handleDelete()
			default:
				if m.search.HandleKey(msg) {
					m.cursor = 0
					m.scrollY = 0
				}
			}
			return m, nil
		}
		switch k {
		case "/":
			m.search.Activate()
			m.cursor = 0
			m.scrollY = 0
		case "1":
			m.switchSection(SectionQueries)
		case "2":
			m.switchSection(SectionTemplates)
		case "3":
			m.switchSection(SectionTables)
		case "4":
			m.switchSection(SectionHistory)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.moveDown()
		case "enter":
			return m.handleEnter()
		case "ctrl+d":
			return m.handleDelete()
		}
	}
	return m, nil
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	switch m.activeSection {
	case SectionQueries:
		vq := m.visibleQueries()
		if m.cursor < len(vq) {
			content := vq[m.cursor].Content
			return m, func() tea.Msg { return SelectQueryMsg{Content: content} }
		}
	case SectionTemplates:
		vt := m.visibleTemplates()
		if m.cursor < len(vt) {
			t := vt[m.cursor]
			return m, func() tea.Msg { return SelectTemplateMsg{Template: t} }
		}
	case SectionTables:
		idx := 0
		for _, ti := range m.visibleTableIdx() {
			if idx == m.cursor {
				if m.tables[ti].Expanded {
					m.tables[ti].Expanded = false
				} else if len(m.tables[ti].Columns) > 0 {
					m.tables[ti].Expanded = true
				} else {
					schema := m.tables[ti].Schema
					name := m.tables[ti].Name
					return m, func() tea.Msg {
						return RequestColumnsMsg{Schema: schema, TableName: name}
					}
				}
				return m, nil
			}
			idx++
			if m.tables[ti].Expanded {
				for range m.tables[ti].Columns {
					if idx == m.cursor {
						return m, nil
					}
					idx++
				}
			}
		}
	case SectionHistory:
		vh := m.visibleHistory()
		if m.cursor < len(vh) {
			sql := vh[m.cursor].SQL
			return m, func() tea.Msg { return SelectHistoryMsg{SQL: sql} }
		}
	}
	return m, nil
}

func (m Model) handleDelete() (Model, tea.Cmd) {
	if m.activeSection == SectionQueries {
		vq := m.visibleQueries()
		if m.cursor < len(vq) {
			name := vq[m.cursor].Name
			return m, func() tea.Msg { return DeleteQueryMsg{Name: name} }
		}
	}
	return m, nil
}

// ── Mouse ──────────────────────────────────────────────────
//
// The helpers below map a click on a rendered line back to an item. They mirror
// View's layout math (header lines, one title line per section, the active
// section's scrolled body); the two must change together.

// headerLines is the CONNECTION block above the section list.
const headerLines = 3

// bodyGeometry returns the first content line of the active section's body, how
// many lines it spans, and the scroll offset View would render it with.
func (m Model) bodyGeometry() (start, bodyH, scrollY int) {
	innerH := m.height
	if innerH < 5 {
		innerH = 5
	}
	bodyH = innerH - (headerLines + int(sectionCount))
	if bodyH < 1 {
		bodyH = 1
	}
	start = headerLines + int(m.activeSection) + 1

	bodyLines := m.buildSectionLines(m.activeSection, m.innerWidth())
	curVL := m.cursorToVisualLine(m.activeSection)
	scrollY = m.scrollY
	if curVL < scrollY {
		scrollY = curVL
	}
	if curVL >= scrollY+bodyH {
		scrollY = curVL - bodyH + 1
	}
	if scrollY < 0 {
		scrollY = 0
	}
	maxScroll := len(bodyLines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	return start, bodyH, scrollY
}

// sectionTitleLine returns the content line the given section's title sits on.
func (m Model) sectionTitleLine(sec Section) int {
	_, bodyH, _ := m.bodyGeometry()
	line := headerLines + int(sec)
	if sec > m.activeSection {
		line += bodyH
	}
	return line
}

// visualLineToCursor is the inverse of cursorToVisualLine.
func (m Model) visualLineToCursor(sec Section, vl int) (int, bool) {
	if vl < 0 {
		return 0, false
	}
	if sec != SectionTables {
		if vl >= m.sectionItemCount(sec) {
			return 0, false
		}
		return vl, true
	}
	visualLine, idx := 0, 0
	for _, ti := range m.visibleTableIdx() {
		if visualLine == vl {
			return idx, true
		}
		visualLine++
		idx++
		if m.tables[ti].Expanded {
			for range m.tables[ti].Columns {
				if visualLine == vl {
					return idx, true
				}
				visualLine++
				idx++
			}
		}
	}
	return 0, false
}

// TableAt reports the table at the given cursor index, or ok=false when that
// index addresses one of a table's expanded columns instead.
func (m Model) TableAt(cursor int) (schema, name string, ok bool) {
	idx := 0
	for _, ti := range m.visibleTableIdx() {
		if idx == cursor {
			return m.tables[ti].Schema, m.tables[ti].Name, true
		}
		idx++
		if m.tables[ti].Expanded {
			for range m.tables[ti].Columns {
				if idx == cursor {
					return "", "", false
				}
				idx++
			}
		}
	}
	return "", "", false
}

// MouseClick handles a click on the given content line. Clicking a section
// title activates that section; clicking a row selects it, and clicking a table
// row sends a starter statement to the editor. Expanding a table stays on Enter
// so a click has one unambiguous meaning.
func (m Model) MouseClick(line int) (Model, tea.Cmd) {
	for sec := Section(0); sec < sectionCount; sec++ {
		if line == m.sectionTitleLine(sec) {
			if sec != m.activeSection {
				m.switchSection(sec)
			}
			return m, nil
		}
	}

	start, bodyH, scrollY := m.bodyGeometry()
	if line < start || line >= start+bodyH {
		return m, nil
	}
	cursor, ok := m.visualLineToCursor(m.activeSection, line-start+scrollY)
	if !ok {
		return m, nil
	}
	m.cursor = cursor

	if m.activeSection == SectionTables {
		if schema, name, isTable := m.TableAt(cursor); isTable {
			return m, func() tea.Msg { return SelectTableMsg{Schema: schema, Name: name} }
		}
		return m, nil
	}
	return m.handleEnter()
}

// MouseWheel moves the selection by delta rows (positive scrolls down).
func (m Model) MouseWheel(delta int) Model {
	for ; delta > 0; delta-- {
		m.moveDown()
	}
	for ; delta < 0; delta++ {
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m
}

const hPadTotal = 3

func (m Model) innerWidth() int {
	w := m.width - hPadTotal - 1
	if w < 10 {
		w = 10
	}
	return w
}

func (m Model) View() string {
	innerW := m.innerWidth()
	innerH := m.height
	if innerH < 5 {
		innerH = 5
	}

	var allLines []string

	allLines = append(allLines, style.PanelTitle.Render("CONNECTION"))

	if m.connName != "" {
		dot := style.ConnDot.Render("●")
		name := style.ConnName.Render(m.connName)
		allLines = append(allLines, dot+" "+name)
	} else {
		allLines = append(allLines, style.PanelTitle.Render("● (not connected)"))
	}

	if m.connDB != "" {
		allLines = append(allLines, style.StatusMsg.Render(trunc(m.connDB, innerW)))
	} else {
		allLines = append(allLines, "")
	}

	fixedLines := len(allLines) + int(sectionCount)
	bodyH := innerH - fixedLines
	if bodyH < 1 {
		bodyH = 1
	}

	for sec := Section(0); sec < sectionCount; sec++ {
		title := m.sectionTitle(sec)
		if sec == m.activeSection {
			titleStyle := style.PanelTitleActive
			if !m.focused {
				titleStyle = style.PanelTitle.Bold(true)
			}
			if m.search.Active() {
				// Show the live query in place of the count, truncated to fit.
				raw := trunc(m.sectionName(sec)+" /"+m.search.Query(), innerW-1) + "▏"
				allLines = append(allLines, titleStyle.Render(raw))
			} else {
				allLines = append(allLines, titleStyle.Render(title))
			}

			bodyLines := m.buildSectionLines(sec, innerW)

			curVL := m.cursorToVisualLine(sec)
			scrollY := m.scrollY
			if curVL < scrollY {
				scrollY = curVL
			}
			if curVL >= scrollY+bodyH {
				scrollY = curVL - bodyH + 1
			}
			if scrollY < 0 {
				scrollY = 0
			}
			maxScroll := len(bodyLines) - bodyH
			if maxScroll < 0 {
				maxScroll = 0
			}
			if scrollY > maxScroll {
				scrollY = maxScroll
			}

			end := scrollY + bodyH
			if end > len(bodyLines) {
				end = len(bodyLines)
			}
			visible := bodyLines[scrollY:end]
			for len(visible) < bodyH {
				visible = append(visible, "")
			}
			allLines = append(allLines, visible...)
		} else {
			allLines = append(allLines, style.PanelTitle.Render(title))
		}
	}

	if len(allLines) > innerH {
		allLines = allLines[:innerH]
	}
	for len(allLines) < innerH {
		allLines = append(allLines, "")
	}

	content := strings.Join(allLines, "\n")
	content = style.PrefixFocusBar(content, m.focused)
	return style.Sidebar.
		Width(m.width).
		Height(innerH).
		MaxHeight(innerH).
		Render(content)
}

func (m Model) sectionTitle(sec Section) string {
	switch sec {
	case SectionQueries:
		return fmt.Sprintf("1 QUERIES (%d)", len(m.queries))
	case SectionTemplates:
		return fmt.Sprintf("2 TEMPLATES (%d)", len(m.templates))
	case SectionTables:
		return fmt.Sprintf("3 TABLES (%d)", len(m.tables))
	case SectionHistory:
		return fmt.Sprintf("4 HISTORY (%d)", len(m.history))
	}
	return ""
}

func (m Model) sectionName(sec Section) string {
	switch sec {
	case SectionQueries:
		return "QUERIES"
	case SectionTemplates:
		return "TEMPLATES"
	case SectionTables:
		return "TABLES"
	case SectionHistory:
		return "HISTORY"
	}
	return ""
}

func (m Model) cursorToVisualLine(sec Section) int {
	if sec != SectionTables {
		return m.cursor
	}
	visualLine := 0
	idx := 0
	for _, ti := range m.visibleTableIdx() {
		if idx == m.cursor {
			return visualLine
		}
		visualLine++
		idx++
		if m.tables[ti].Expanded {
			for range m.tables[ti].Columns {
				if idx == m.cursor {
					return visualLine
				}
				visualLine++
				idx++
			}
		}
	}
	return visualLine
}

func (m Model) buildSectionLines(sec Section, maxW int) []string {
	switch sec {
	case SectionQueries:
		return m.buildQueryLines(maxW)
	case SectionTemplates:
		return m.buildTemplateLines(maxW)
	case SectionTables:
		return m.buildTableLines(maxW)
	case SectionHistory:
		return m.buildHistoryLines(maxW)
	}
	return nil
}

func (m Model) buildQueryLines(maxW int) []string {
	vq := m.visibleQueries()
	if len(vq) == 0 {
		return []string{style.StatusMsg.Render(m.emptyLabel())}
	}
	var lines []string
	for i, q := range vq {
		name := trunc(q.Name, maxW-3)
		if m.focused && m.activeSection == SectionQueries && i == m.cursor {
			lines = append(lines, style.ListSelected.Render("▸ "+name))
		} else {
			lines = append(lines, style.ListItem.Render(name))
		}
	}
	return lines
}

func (m Model) buildTemplateLines(maxW int) []string {
	vt := m.visibleTemplates()
	if len(vt) == 0 {
		return []string{style.StatusMsg.Render(m.emptyLabel())}
	}
	var lines []string
	for i, t := range vt {
		name := t.Name
		if name == "" {
			name = fmt.Sprintf("template-%d", i)
		}
		name = trunc(name, maxW-3)
		if m.focused && m.activeSection == SectionTemplates && i == m.cursor {
			lines = append(lines, style.ListSelected.Render("▸ "+name))
		} else {
			lines = append(lines, style.ListItem.Render(name))
		}
	}
	return lines
}

func (m Model) buildTableLines(maxW int) []string {
	vis := m.visibleTableIdx()
	if len(vis) == 0 {
		return []string{style.StatusMsg.Render(m.emptyLabel())}
	}
	var lines []string
	idx := 0

	for _, ti := range vis {
		t := m.tables[ti]
		selected := m.focused && m.activeSection == SectionTables && idx == m.cursor
		arrow := "▶"
		if t.Expanded {
			arrow = "▼"
		}
		if selected {
			arrow = "▸"
		}

		label := trunc(t.DisplayName(), maxW-4)
		if selected {
			lines = append(lines, style.TableNameSelected.Render(arrow+" "+label))
		} else {
			lines = append(lines, style.TableName.Render(arrow+" "+label))
		}
		idx++

		if t.Expanded {
			colNameW := 0
			for _, col := range t.Columns {
				if len(col.Name) > colNameW {
					colNameW = len(col.Name)
				}
			}
			if colNameW > maxW/2 {
				colNameW = maxW / 2
			}

			for _, col := range t.Columns {
				colSel := m.focused && m.activeSection == SectionTables && idx == m.cursor

				leftChrome := 7
				if colSel {
					leftChrome = 6
				}
				typeAvail := maxW - leftChrome - colNameW
				if typeAvail < 4 {
					typeAvail = 4
				}
				dataType := trunc(col.DataType, typeAvail)
				name := trunc(col.Name, colNameW)

				if colSel {
					row := "▸ " + padRight(name, colNameW) + " " + dataType
					lines = append(lines, style.ColumnItemSelected.Render(row))
				} else {
					styledName := style.TableName.Render(padRight(name, colNameW))
					styledType := style.ColumnType.Render(dataType)
					lines = append(lines, "    "+styledName+" "+styledType)
				}
				idx++
			}
		}
	}
	return lines
}

func (m Model) buildHistoryLines(maxW int) []string {
	vh := m.visibleHistory()
	if len(vh) == 0 {
		return []string{style.StatusMsg.Render(m.emptyLabel())}
	}
	var lines []string
	for i, h := range vh {
		sql := strings.ReplaceAll(h.SQL, "\n", " ")
		sql = strings.ReplaceAll(sql, "\r", "")
		sql = strings.ReplaceAll(sql, "\t", " ")
		ts := h.Timestamp.Local().Format("15:04")
		name := trunc(sql+"  "+ts, maxW-3)
		if m.focused && m.activeSection == SectionHistory && i == m.cursor {
			lines = append(lines, style.ListSelected.Render("▸ "+name))
		} else {
			lines = append(lines, style.ListItem.Render(name))
		}
	}
	return lines
}

func trunc(s string, maxW int) string {
	r := []rune(s)
	if len(r) <= maxW || maxW <= 0 {
		return s
	}
	if maxW < 2 {
		return string(r[:maxW])
	}
	return string(r[:maxW-1]) + "…"
}

func padRight(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(r))
}
