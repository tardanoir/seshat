package sidebar

import (
	"fmt"
	"strings"

	"github.com/tardanoir/seshat/internal/query"
	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

type SelectQueryMsg struct{ Content string }
type SelectTemplateMsg struct{ Template query.Template }
type SelectHistoryMsg struct{ SQL string }
type DeleteQueryMsg struct{ Name string }
type RequestColumnsMsg struct{ Schema, TableName string }

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
		return len(m.queries)
	case SectionTemplates:
		return len(m.templates)
	case SectionTables:
		n := 0
		for _, t := range m.tables {
			n++
			if t.Expanded {
				n += len(t.Columns)
			}
		}
		return n
	case SectionHistory:
		return len(m.history)
	}
	return 0
}

func (m *Model) switchSection(sec Section) {
	m.activeSection = sec
	m.cursor = 0
	m.scrollY = 0
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
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
			mx := m.sectionItemCount(m.activeSection) - 1
			if mx < 0 {
				mx = 0
			}
			if m.cursor < mx {
				m.cursor++
			}
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
		if m.cursor < len(m.queries) {
			content := m.queries[m.cursor].Content
			return m, func() tea.Msg { return SelectQueryMsg{Content: content} }
		}
	case SectionTemplates:
		if m.cursor < len(m.templates) {
			t := m.templates[m.cursor]
			return m, func() tea.Msg { return SelectTemplateMsg{Template: t} }
		}
	case SectionTables:
		idx := 0
		for i := range m.tables {
			if idx == m.cursor {
				if m.tables[i].Expanded {
					m.tables[i].Expanded = false
				} else if len(m.tables[i].Columns) > 0 {
					m.tables[i].Expanded = true
				} else {
					schema := m.tables[i].Schema
					name := m.tables[i].Name
					return m, func() tea.Msg {
						return RequestColumnsMsg{Schema: schema, TableName: name}
					}
				}
				return m, nil
			}
			idx++
			if m.tables[i].Expanded {
				for range m.tables[i].Columns {
					if idx == m.cursor {
						return m, nil
					}
					idx++
				}
			}
		}
	case SectionHistory:
		if m.cursor < len(m.history) {
			sql := m.history[m.cursor].SQL
			return m, func() tea.Msg { return SelectHistoryMsg{SQL: sql} }
		}
	}
	return m, nil
}

func (m Model) handleDelete() (Model, tea.Cmd) {
	if m.activeSection == SectionQueries && m.cursor < len(m.queries) {
		name := m.queries[m.cursor].Name
		return m, func() tea.Msg { return DeleteQueryMsg{Name: name} }
	}
	return m, nil
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
			allLines = append(allLines, titleStyle.Render(title))

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

func (m Model) cursorToVisualLine(sec Section) int {
	if sec != SectionTables {
		return m.cursor
	}
	visualLine := 0
	idx := 0
	for _, t := range m.tables {
		if idx == m.cursor {
			return visualLine
		}
		visualLine++
		idx++
		if t.Expanded {
			for range t.Columns {
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
	if len(m.queries) == 0 {
		return []string{style.StatusMsg.Render("(none)")}
	}
	var lines []string
	for i, q := range m.queries {
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
	if len(m.templates) == 0 {
		return []string{style.StatusMsg.Render("(none)")}
	}
	var lines []string
	for i, t := range m.templates {
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
	if len(m.tables) == 0 {
		return []string{style.StatusMsg.Render("(none)")}
	}
	var lines []string
	idx := 0

	for _, t := range m.tables {
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
	if len(m.history) == 0 {
		return []string{style.StatusMsg.Render("(none)")}
	}
	var lines []string
	for i, h := range m.history {
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
