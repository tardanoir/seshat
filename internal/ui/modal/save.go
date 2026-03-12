package modal

import (
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type SaveQueryMsg struct {
	Name string
}

type SaveModel struct {
	input  textinput.Model
	width  int
	height int
}

func NewSave() SaveModel {
	ti := textinput.New()
	ti.Placeholder = "query name"
	ti.Focus()
	ti.CharLimit = 64
	ti.SetWidth(40)
	return SaveModel{input: ti}
}

func (m *SaveModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m SaveModel) Update(msg tea.Msg) (SaveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Enter):
			name := m.input.Value()
			if name != "" {
				return m, func() tea.Msg { return SaveQueryMsg{Name: name} }
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SaveModel) View() string {
	content := style.Title.Render("Save Query") + "\n\n" +
		style.Label.Render("Name: ") + m.input.View() + "\n\n" +
		style.ListItem.Render("↵ save  Esc cancel")

	modalW := 50
	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
