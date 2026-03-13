package modal

import (
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ExportFormat int

const (
	FormatCSV ExportFormat = iota
	FormatJSON
)

type ExportFormatMsg struct {
	Format ExportFormat
}

type ExportModel struct {
	cursor int
	width  int
	height int
}

func NewExport() ExportModel {
	return ExportModel{}
}

func (m *ExportModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ExportModel) Update(msg tea.Msg) (ExportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < 1 {
				m.cursor++
			}
		case key.Matches(msg, style.Keys.Enter):
			cursor := m.cursor
			return m, func() tea.Msg {
				return ExportFormatMsg{Format: ExportFormat(cursor)}
			}
		}
	}
	return m, nil
}

func (m ExportModel) View() string {
	formats := []string{"CSV", "JSON"}

	var items string
	for i, f := range formats {
		if i == m.cursor {
			items += style.ListSelected.Render("▸ "+f) + "\n"
		} else {
			items += style.ListItem.Render(f) + "\n"
		}
	}

	content := style.Title.Render("Export Results") + "\n\n" +
		items + "\n" +
		style.ListItem.Render("↵ export  Esc cancel")

	modalW := 40
	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
