package modal

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ai"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// AIAcceptMsg is emitted when the user accepts the generated SQL.
// The receiver should replace the comment block with msg.SQL.
type AIAcceptMsg struct {
	SQL   string
	Block ai.CommentBlock
}

// AIRejectMsg is emitted when the user rejects the generated SQL.
type AIRejectMsg struct{}

type AIReviewModel struct {
	sql      string
	block    ai.CommentBlock
	provider string
	width    int
	height   int
}

func NewAIReview(sql, provider string, block ai.CommentBlock) AIReviewModel {
	return AIReviewModel{sql: sql, provider: provider, block: block}
}

func (m *AIReviewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m AIReviewModel) Update(msg tea.Msg) (AIReviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Enter),
			key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
			sql := m.sql
			block := m.block
			return m, func() tea.Msg { return AIAcceptMsg{SQL: sql, Block: block} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			return m, func() tea.Msg { return AIRejectMsg{} }
		}
	}
	return m, nil
}

func (m AIReviewModel) View() string {
	title := "AI suggestion"
	if m.provider != "" {
		title += " — " + m.provider
	}
	body := strings.TrimRight(m.sql, "\n")

	modalW := m.width - 8
	if modalW > 100 {
		modalW = 100
	}
	if modalW < 40 {
		modalW = 40
	}

	content := style.Title.Render(title) + "\n\n" +
		body + "\n\n" +
		style.ListItem.Render("↵/y accept · Esc/n reject · replaces the comment with this SQL")

	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
