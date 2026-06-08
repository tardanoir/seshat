package queryeditor

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

// NvimRedrawMsg signals that the embedded Neovim flushed a new frame.
type NvimRedrawMsg struct{}

// NvimExitedMsg signals that the embedded Neovim process is gone.
type NvimExitedMsg struct{}

// NvimActive reports whether the editor is backed by an embedded Neovim.
func (m Model) NvimActive() bool { return m.embedded() }

// NvimInsertMode reports whether Neovim is in an insert-like mode where keys
// such as Tab should be forwarded to the editor rather than handled by the app.
func (m Model) NvimInsertMode() bool {
	if !m.embedded() {
		return false
	}
	mode := m.bridge.ModeName()
	return strings.HasPrefix(mode, "insert") ||
		strings.HasPrefix(mode, "replace") ||
		strings.HasPrefix(mode, "cmdline")
}

// NvimSubscribe returns a command that waits for the next redraw or for the
// process to exit. Re-issue it after every NvimRedrawMsg to keep listening.
func (m Model) NvimSubscribe() tea.Cmd {
	if m.bridge == nil {
		return nil
	}
	redraw := m.bridge.RedrawCh()
	done := m.bridge.Done()
	return func() tea.Msg {
		select {
		case <-redraw:
			return NvimRedrawMsg{}
		case <-done:
			return NvimExitedMsg{}
		}
	}
}

// Close terminates the embedded Neovim, if any.
func (m *Model) Close() {
	if m.bridge != nil {
		_ = m.bridge.Close()
		m.bridge = nil
	}
}

// DisableNvim tears down the embedded editor while preserving its last contents
// so vim mode can degrade to the j/k navigator.
func (m *Model) DisableNvim() {
	if m.bridge == nil {
		return
	}
	m.sql = m.bridge.Text()
	m.stmts = parseStatements(m.sql)
	_ = m.bridge.Close()
	m.bridge = nil
}

func (m Model) updateEmbedded(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case NvimRedrawMsg:
		// Refresh the cached buffer mirror used for the status bar, statement
		// parsing, and schema completion. The visible grid is rendered straight
		// from the bridge.
		m.sql = m.bridge.Text()
		m.stmts = parseStatements(m.sql)
		m.cursorLine = m.bridge.CursorLine()
		m.cursorCol = m.bridge.CursorCol()
		m.refreshCompletionEmbedded()
		return m, nil
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		// When seshat's completion popup is open, it owns the navigation/accept
		// keys before they reach nvim (mirrors the textarea editor).
		if m.comp.open {
			switch msg.String() {
			case "tab":
				m.expandOrAcceptEmbedded()
				return m, nil
			case "enter":
				m.acceptSelectedEmbedded()
				return m, nil
			case "up", "ctrl+p":
				if m.comp.selected > 0 {
					m.comp.selected--
				}
				return m, nil
			case "down":
				if m.comp.selected < len(m.comp.items)-1 {
					m.comp.selected++
				}
				return m, nil
			case "esc":
				m.closeCompletion()
				m.bridge.Input("<Esc>")
				return m, nil
			}
		}
		m.bridge.Input(nvimKeys(msg))
		return m, nil
	}
	return m, nil
}

// refreshCompletionEmbedded recomputes the schema completion popup from the
// mirrored buffer + cursor. It only shows while inserting text.
func (m *Model) refreshCompletionEmbedded() {
	mode := m.bridge.ModeName()
	if !strings.HasPrefix(mode, "insert") && !strings.HasPrefix(mode, "replace") {
		m.closeCompletion()
		return
	}
	lines := strings.Split(m.sql, "\n")
	row := m.cursorLine
	if row < 0 || row >= len(lines) {
		m.closeCompletion()
		return
	}
	// nvim reports a byte column; buildCompletion wants a rune index.
	byteCol := m.cursorCol
	if byteCol > len(lines[row]) {
		byteCol = len(lines[row])
	}
	runeCol := len([]rune(lines[row][:byteCol]))
	m.comp = m.buildCompletion(lines, row, runeCol)
}

// expandOrAcceptEmbedded inserts the longest common prefix of the suggestions,
// or accepts the selected one if there's nothing left to expand.
func (m *Model) expandOrAcceptEmbedded() {
	if !m.comp.open || len(m.comp.items) == 0 {
		return
	}
	lcp := longestCommonPrefix(m.comp.items)
	if len(lcp) > len(m.comp.prefix) {
		m.bridge.Input(escapeLt(lcp[len(m.comp.prefix):]))
		m.closeCompletion()
		return
	}
	m.acceptSelectedEmbedded()
}

func (m *Model) acceptSelectedEmbedded() {
	if !m.comp.open || len(m.comp.items) == 0 {
		return
	}
	pick := m.comp.items[m.comp.selected].Text
	suffix := pick
	if strings.HasPrefix(strings.ToLower(pick), strings.ToLower(m.comp.prefix)) {
		suffix = pick[len(m.comp.prefix):]
	}
	m.bridge.Input(escapeLt(suffix))
	m.closeCompletion()
}

func (m Model) viewEmbedded() string {
	body := m.bridge.Render(m.contentWidth(), m.contentHeight(), m.focused)
	if m.comp.open {
		row, col := m.bridge.CursorScreen()
		body = m.overlayPopupAt(body, col, row)
	}
	content := m.renderHeader() + "\n" + body
	content = style.PrefixFocusBar(content, m.focused)
	return style.Editor.Width(m.width).Height(m.height).MaxHeight(m.height).Render(content)
}
