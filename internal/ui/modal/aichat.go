package modal

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ai"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// AIChatSubmitMsg is emitted when the user sends a message. The App reads the
// modal's History() to build the request, so the message carries no payload.
type AIChatSubmitMsg struct{}

// AIChatInsertMsg carries SQL from the latest assistant reply to drop into the editor.
type AIChatInsertMsg struct{ SQL string }

// AIChatRunMsg carries SQL from the latest assistant reply to insert and run.
type AIChatRunMsg struct{ SQL string }

// OpenAIProvidersMsg opens the provider manager (reachable from chat via C-p).
type OpenAIProvidersMsg struct{}

// AIChatModel is a full-screen multi-turn chat with live streaming replies.
type AIChatModel struct {
	messages  []ai.ChatMessage // finalized turns (user/assistant)
	partial   string           // in-progress assistant text while streaming
	streaming bool
	input     textinput.Model
	provider  string
	width     int
	height    int
}

func NewAIChat(provider string) AIChatModel {
	ti := textinput.New()
	ti.Placeholder = "Ask about your data — e.g. top 5 customers by revenue"
	ti.Focus()
	ti.CharLimit = 0
	return AIChatModel{input: ti, provider: provider}
}

func (m *AIChatModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	iw := m.modalWidth() - 8
	if iw < 10 {
		iw = 10
	}
	m.input.SetWidth(iw)
}

func (m *AIChatModel) SetProvider(name string) { m.provider = name }

// History returns the finalized conversation (ending with the latest user
// message), used by the App to build the chat request.
func (m AIChatModel) History() []ai.ChatMessage { return m.messages }

// Streaming reports whether a reply is currently being received.
func (m AIChatModel) Streaming() bool { return m.streaming }

// AppendDelta appends a streamed token to the in-progress assistant reply.
func (m *AIChatModel) AppendDelta(s string) { m.partial += s }

// EndTurn finalizes the in-progress assistant reply (called on Done or error).
func (m *AIChatModel) EndTurn() {
	if m.streaming {
		m.messages = append(m.messages, ai.ChatMessage{Role: "assistant", Content: m.partial})
	}
	m.partial = ""
	m.streaming = false
}

// LastSQL extracts SQL from the most recent assistant message, if any.
func (m AIChatModel) LastSQL() string {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "assistant" {
			return ai.ExtractSQL(m.messages[i].Content)
		}
	}
	return ""
}

func (m AIChatModel) Update(msg tea.Msg) (AIChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+y"))):
			if sql := m.LastSQL(); sql != "" {
				return m, func() tea.Msg { return AIChatInsertMsg{SQL: sql} }
			}
			return m, nil
		case key.Matches(msg, style.Keys.Execute): // C-r: run latest SQL
			if sql := m.LastSQL(); sql != "" {
				return m, func() tea.Msg { return AIChatRunMsg{SQL: sql} }
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
			return m, func() tea.Msg { return OpenAIProvidersMsg{} }
		case key.Matches(msg, style.Keys.Enter):
			if m.streaming {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.messages = append(m.messages, ai.ChatMessage{Role: "user", Content: text})
			m.input.SetValue("")
			m.streaming = true
			m.partial = ""
			return m, func() tea.Msg { return AIChatSubmitMsg{} }
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m AIChatModel) modalWidth() int {
	w := m.width - 8
	if w > 100 {
		w = 100
	}
	if w < 40 {
		w = 40
	}
	return w
}

func (m AIChatModel) assistantName() string {
	if m.provider == "" {
		return "ai"
	}
	return m.provider
}

func (m AIChatModel) View() string {
	modalW := m.modalWidth()
	innerW := modalW - 4
	if innerW < 10 {
		innerW = 10
	}
	wrap := lipgloss.NewStyle().Width(innerW)

	label := func(role string) string {
		if role == "user" {
			return style.StatusKey.Render("you")
		}
		return style.Success.Render(m.assistantName())
	}

	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(label(msg.Role) + "\n")
		b.WriteString(wrap.Render(strings.TrimRight(msg.Content, "\n")))
		b.WriteString("\n\n")
	}
	if m.streaming {
		b.WriteString(label("assistant") + "\n")
		b.WriteString(wrap.Render(m.partial + "▌"))
		b.WriteString("\n\n")
	}
	transcript := strings.TrimRight(b.String(), "\n")

	// Auto-follow: clamp to the available height and keep the tail.
	maxLines := m.height - 12
	if maxLines < 4 {
		maxLines = 4
	}
	if lines := strings.Split(transcript, "\n"); len(lines) > maxLines {
		transcript = strings.Join(lines[len(lines)-maxLines:], "\n")
	}
	if strings.TrimSpace(transcript) == "" {
		transcript = style.StatusMsg.Render("Ask a question to get started. Schema-aware; replies stream live.")
	}

	hints := []string{
		style.StatusKey.Render("↵") + " " + style.StatusKeyLabel.Render("send"),
		style.StatusKey.Render("C-y") + " " + style.StatusKeyLabel.Render("insert sql"),
		style.StatusKey.Render("C-r") + " " + style.StatusKeyLabel.Render("run sql"),
		style.StatusKey.Render("C-p") + " " + style.StatusKeyLabel.Render("providers"),
		style.StatusKey.Render("Esc") + " " + style.StatusKeyLabel.Render("close"),
	}

	title := "AI chat"
	if m.provider != "" {
		title += " — " + m.provider
	}

	content := style.Title.Render(title) + "\n\n" +
		transcript + "\n\n" +
		style.StatusMsg.Render(strings.Repeat("─", innerW)) + "\n" +
		"> " + m.input.View() + "\n\n" +
		"  " + strings.Join(hints, "  ")

	rendered := style.ModalOverlay.Width(modalW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}
