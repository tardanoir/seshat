package sidebar

import (
	"fmt"
	"strings"

	"seshat/internal/query"
	"seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

type SelectQueryMsg struct{ Content string }
type SelectTemplateMsg struct{ Template query.Template }
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
	return t.Name
}

type Section int

const (
	SectionQueries Section = iota
	SectionTemplates
	SectionTables
	sectionCount = 3
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

	activeSection Section // which panel is expanded
	cursor        int     // item index within the active section
	scrollY       int     // scroll offset within the active panel
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
func (m *Model) SetTemplates(t []query.Template)    { m.templates = t }
func (m *Model) SetTables(tables []TableEntry)      { m.tables = tables }

func (m *Model) SetTableColumns(schema, tableName string, cols []ColumnDef) {
	for i := range m.tables {
		if m.tables[i].Schema == schema && m.tables[i].Name == tableName {
			m.tables[i].Columns = cols
			m.tables[i].Expanded = true
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
		// Map cursor to table/column
		idx := 0
		for i := range m.tables {
			if idx == m.cursor {
				// Toggle table expansion
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
						// Column selected — no action for now
						return m, nil
					}
					idx++
				}
			}
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

func (m Model) View() string {
	innerW := m.width - 4 // outer border(2) + padding(2)
	if innerW < 4 {
		innerW = 4
	}
	innerH := m.height - 2 // outer border
	if innerH < 3 {
		innerH = 3
	}

	// Build all lines manually to guarantee exact height.
	// Line budget: connHeader(1) + per-section(title line) + expanded body lines
	var allLines []string

	// Connection header
	connLabel := m.connName
	if m.connDB != "" {
		connLabel += "/" + m.connDB
	}
	allLines = append(allLines, style.StatusConn.Render(trunc(connLabel, innerW)))

	// Fixed lines: 1 (conn) + 3 (section titles)
	fixedLines := 1 + int(sectionCount)
	bodyH := innerH - fixedLines
	if bodyH < 1 {
		bodyH = 1
	}

	for sec := Section(0); sec < sectionCount; sec++ {
		title := m.sectionTitle(sec)
		if sec == m.activeSection {
			titleStyle := style.PanelTitleActive
			if !m.focused {
				titleStyle = style.PanelTitle
			}
			allLines = append(allLines, titleStyle.Render(title))
			// Expanded body
			bodyLines := m.buildSectionLines(sec, innerW)

			// Scroll
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

	// Ensure exact height
	for len(allLines) < innerH {
		allLines = append(allLines, "")
	}
	allLines = allLines[:innerH]

	content := strings.Join(allLines, "\n")

	s := style.Sidebar
	if m.focused {
		s = style.Focused
	}
	return s.Width(m.width - 2).Render(content)
}

func (m Model) sectionTitle(sec Section) string {
	switch sec {
	case SectionQueries:
		return fmt.Sprintf("1 Queries (%d)", len(m.queries))
	case SectionTemplates:
		return fmt.Sprintf("2 Templates (%d)", len(m.templates))
	case SectionTables:
		return fmt.Sprintf("3 Tables (%d)", len(m.tables))
	}
	return ""
}

func (m Model) cursorToVisualLine(sec Section) int {
	if sec != SectionTables {
		return m.cursor // 1:1 for queries and templates
	}
	// Tables: each table = 1 line, each column = 2 lines (name + type)
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
	}
	return nil
}

func (m Model) buildQueryLines(maxW int) []string {
	if len(m.queries) == 0 {
		return []string{"\x1b[2m(none)\x1b[22m"}
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
		return []string{"\x1b[2m(none)\x1b[22m"}
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
		return []string{"\x1b[2m(none)\x1b[22m"}
	}
	var lines []string
	idx := 0
	colNameMax := maxW - 6
	if colNameMax < 5 {
		colNameMax = 5
	}
	for _, t := range m.tables {
		arrow := "▶"
		if t.Expanded {
			arrow = "▼"
		}
		label := trunc(arrow+" "+t.DisplayName(), maxW-3)
		if m.focused && m.activeSection == SectionTables && idx == m.cursor {
			lines = append(lines, style.TableNameSelected.Render("▸ "+label))
		} else {
			lines = append(lines, style.TableName.Render(label))
		}
		idx++

		if t.Expanded {
			for _, col := range t.Columns {
				colText := col.Name + " " + style.ColumnType.Render(col.DataType)
				colText = trunc(colText, colNameMax)
				if m.focused && m.activeSection == SectionTables && idx == m.cursor {
					lines = append(lines, style.ColumnItemSelected.Render("▸ "+colText))
				} else {
					lines = append(lines, style.ColumnItem.Render(colText))
				}
				idx++
			}
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
