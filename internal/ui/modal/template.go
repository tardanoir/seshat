package modal

import (
	"strings"

	"github.com/tardanoir/seshat/internal/query"
	"github.com/tardanoir/seshat/internal/ui/filter"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TemplateResultMsg struct {
	SQL string
}

// TemplatePickerModel lets the user pick a template from the list.
type TemplatePickerModel struct {
	templates []query.Template
	cursor    int
	search    filter.Model
	width     int
	height    int
}

func NewTemplatePicker(templates []query.Template) TemplatePickerModel {
	return TemplatePickerModel{templates: templates}
}

func (m *TemplatePickerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Searching reports whether the picker is in incremental-search mode.
func (m TemplatePickerModel) Searching() bool { return m.search.Active() }

// visible returns the templates matching the current search (by name or
// description), or all of them when not searching.
func (m TemplatePickerModel) visible() []query.Template {
	if !m.search.Active() {
		return m.templates
	}
	out := make([]query.Template, 0, len(m.templates))
	for _, t := range m.templates {
		if m.search.Matches(t.Name) || m.search.Matches(t.Description) {
			out = append(out, t)
		}
	}
	return out
}

func (m TemplatePickerModel) Update(msg tea.Msg) (TemplatePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		vis := m.visible()
		if m.search.Active() {
			switch msg.String() {
			case "esc":
				m.search.Clear()
				m.cursor = 0
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(vis)-1 {
					m.cursor++
				}
			case "enter":
				if m.cursor < len(vis) {
					t := vis[m.cursor]
					return m, func() tea.Msg { return OpenTemplateVarsMsg{Template: t} }
				}
			default:
				if m.search.HandleKey(msg) {
					m.cursor = 0
				}
			}
			return m, nil
		}
		switch {
		case msg.String() == "/":
			m.search.Activate()
			m.cursor = 0
		case key.Matches(msg, style.Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, style.Keys.Down):
			if m.cursor < len(vis)-1 {
				m.cursor++
			}
		case key.Matches(msg, style.Keys.Enter):
			if m.cursor < len(vis) {
				t := vis[m.cursor]
				return m, func() tea.Msg {
					return OpenTemplateVarsMsg{Template: t}
				}
			}
		}
	}
	return m, nil
}

func (m TemplatePickerModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("Select Template"))
	if m.search.Active() {
		b.WriteString("   " + m.search.View())
	}
	b.WriteString("\n\n")

	vis := m.visible()
	switch {
	case len(m.templates) == 0:
		b.WriteString(style.ListItem.Render("No templates found."))
		b.WriteString("\n")
		b.WriteString(style.ListItem.Render("Add .sql files to ~/.config/seshat/templates/"))
	case len(vis) == 0:
		b.WriteString(style.ListItem.Render("No matches."))
	default:
		for i, t := range vis {
			name := t.Name
			if name == "" {
				name = "(unnamed)"
			}
			if t.Description != "" {
				name += " - " + t.Description
			}
			if i == m.cursor {
				b.WriteString(style.ListSelected.Render("▸ " + name))
			} else {
				b.WriteString(style.ListItem.Render("  " + name))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↑↓ navigate  / search  ↵ select  Esc close"))

	modalW := 60
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// OpenTemplateVarsMsg transitions from picker to vars input.
type OpenTemplateVarsMsg struct {
	Template query.Template
}

// TemplateVarsModel collects variable values for a template.
type TemplateVarsModel struct {
	template query.Template
	varNames []string
	inputs   []textinput.Model
	cursor   int
	width    int
	height   int
}

func NewTemplateVars(t query.Template) TemplateVarsModel {
	varNames := query.ExtractVariables(t.SQL)
	inputs := make([]textinput.Model, len(varNames))
	for i, name := range varNames {
		ti := textinput.New()
		if v, ok := t.Variables[name]; ok {
			ti.Placeholder = v.Description
			if v.Default != "" {
				ti.SetValue(v.Default)
			}
		} else {
			ti.Placeholder = name
		}
		ti.SetWidth(40)
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return TemplateVarsModel{
		template: t,
		varNames: varNames,
		inputs:   inputs,
	}
}

func (m *TemplateVarsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m TemplateVarsModel) Update(msg tea.Msg) (TemplateVarsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Enter):
			if m.cursor < len(m.inputs)-1 {
				m.inputs[m.cursor].Blur()
				m.cursor++
				m.inputs[m.cursor].Focus()
				return m, nil
			}
			// Last input — submit
			values := make(map[string]string)
			for i, name := range m.varNames {
				values[name] = m.inputs[i].Value()
			}
			sql := query.Substitute(m.template.SQL, values)
			return m, func() tea.Msg { return TemplateResultMsg{SQL: sql} }
		case key.Matches(msg, style.Keys.Up):
			if m.cursor > 0 {
				m.inputs[m.cursor].Blur()
				m.cursor--
				m.inputs[m.cursor].Focus()
			}
			return m, nil
		case key.Matches(msg, style.Keys.Down):
			if m.cursor < len(m.inputs)-1 {
				m.inputs[m.cursor].Blur()
				m.cursor++
				m.inputs[m.cursor].Focus()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
	return m, cmd
}

func (m TemplateVarsModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("Template: " + m.template.Name))
	b.WriteString("\n\n")

	for i, name := range m.varNames {
		label := name
		if v, ok := m.template.Variables[name]; ok && v.Description != "" {
			label += " (" + v.Description + ")"
		}
		b.WriteString(style.Label.Render(label + ": "))
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↵ next/submit  ↑↓ navigate  Esc cancel"))

	modalW := 60
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
