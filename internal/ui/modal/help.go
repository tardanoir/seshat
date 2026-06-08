package modal

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type CloseHelpMsg struct{}

type HelpModel struct {
	width  int
	height int
}

func NewHelp() HelpModel {
	return HelpModel{}
}

func (m *HelpModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "?" || k == "esc" || k == "enter" || k == "q" {
			return m, func() tea.Msg { return CloseHelpMsg{} }
		}
	}
	return m, nil
}

func (m HelpModel) View() string {
	keyStyle := style.StatusKey
	descStyle := style.StatusKeyLabel

	row := func(key, desc string) string {
		return "  " + keyStyle.Render(key) + "  " + descStyle.Render(desc)
	}

	sections := []string{
		style.Title.Render("Keybindings"),
		"",
		style.Label.Render("Global"),
		row("Ctrl+R", "Execute query"),
		row("Ctrl+E", "Open external editor"),
		row("Ctrl+W", "Save query"),
		row("Ctrl+T", "Load template"),
		row("Ctrl+N", "Switch connection"),
		row("Ctrl+H", "Query history"),
		row("Ctrl+X", "Export results"),
		row("Ctrl+\\", "Toggle sidebar"),
		row("Tab", "Switch focus"),
		row("Ctrl+Z", "Suspend"),
		row("Ctrl+C", "Quit"),
		row("?", "This help"),
		"",
		style.Label.Render("Results"),
		row("j/k", "Move row up/down"),
		row("h/l", "Move column left/right"),
		row("g/G", "First/last row"),
		row("Home/End", "First/last column"),
		row("PgUp/PgDn", "Page up/down"),
		row("y", "Copy cell to clipboard"),
		row("Y", "Copy row to clipboard"),
		"",
		style.Label.Render("Sidebar"),
		row("1-4", "Switch section"),
		row("/", "Search entries"),
		row("Enter", "Select item"),
		row("Ctrl+D", "Delete query"),
	}

	content := strings.Join(sections, "\n")

	modalW := 44
	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
