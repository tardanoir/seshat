package modal

import (
	"seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ConfirmMsg struct {
	Confirmed bool
	Tag       string
}

type ConfirmModel struct {
	message string
	tag     string
	yes     bool
	width   int
	height  int
}

func NewConfirm(message, tag string) ConfirmModel {
	return ConfirmModel{message: message, tag: tag}
}

func (m *ConfirmModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "right", "tab"))):
			m.yes = !m.yes
		case key.Matches(msg, style.Keys.Enter):
			confirmed := m.yes
			tag := m.tag
			return m, func() tea.Msg { return ConfirmMsg{Confirmed: confirmed, Tag: tag} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
			tag := m.tag
			return m, func() tea.Msg { return ConfirmMsg{Confirmed: true, Tag: tag} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			tag := m.tag
			return m, func() tea.Msg { return ConfirmMsg{Confirmed: false, Tag: tag} }
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	yesStyle := style.ListItem
	noStyle := style.ListItem
	if m.yes {
		yesStyle = style.ListSelected
	} else {
		noStyle = style.ListSelected
	}

	content := style.Title.Render("Confirm") + "\n\n" +
		m.message + "\n\n" +
		yesStyle.Render("[ Yes ]") + "  " + noStyle.Render("[ No ]") + "\n\n" +
		style.ListItem.Render("y/n or ←→ + ↵")

	modalW := 50
	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
