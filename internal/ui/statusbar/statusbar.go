package statusbar

import (
	"fmt"
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/lipgloss/v2"
)

type Model struct {
	width    int
	message  string
	isError  bool
	connName string
	dbName   string
	focus    string // "query", "results", "sidebar"
	stmtInfo string // "2/5" etc
}

func New() Model {
	return Model{message: "Ready"}
}

func (m *Model) SetWidth(w int)        { m.width = w }
func (m *Model) SetMessage(msg string) { m.message = msg; m.isError = false }
func (m *Model) SetError(msg string)   { m.message = msg; m.isError = true }
func (m *Model) SetFocus(f string)     { m.focus = f }
func (m *Model) SetStmtInfo(s string)  { m.stmtInfo = s }

func (m *Model) SetConnection(name, dbName string) {
	m.connName = name
	m.dbName = dbName
}

func (m *Model) SetResult(duration, rowCount string) {
	m.isError = false
	m.message = fmt.Sprintf("%s rows · %s", rowCount, duration)
}

func (m *Model) Clear() {
	m.message = "Ready"
	m.isError = false
}

func hint(key, label string) string {
	return style.StatusKey.Render(key) + " " + style.StatusKeyLabel.Render(label)
}

func (m Model) View() string {
	dim := style.StatusSep.Render

	// Left: connection badge + message
	var left string
	if m.connName != "" {
		label := m.connName
		if m.dbName != "" {
			label += "/" + m.dbName
		}
		left = style.StatusConn.Render(label)
	}

	msg := m.message
	if m.isError {
		msg = style.Error.Render(msg)
	} else {
		msg = style.StatusMsg.Render(msg)
	}
	if left != "" {
		left += dim("  ")
	}
	left += msg

	// Right: focus indicator + stmt info + key hints
	var right []string

	if m.stmtInfo != "" {
		right = append(right, dim("stmt:")+style.StatusKey.Render(m.stmtInfo))
	}

	if m.focus != "" {
		right = append(right, style.StatusFocus.Render(m.focus))
	}

	hints := []string{
		hint("^e", "edit"),
		hint("^r", "run"),
		hint("^w", "save"),
		hint("^n", "conn"),
		hint("^\\", "side"),
		hint("tab", "focus"),
	}
	right = append(right, strings.Join(hints, dim(" ")))

	rightStr := strings.Join(right, dim("  │  "))

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(rightStr)
	gap := m.width - leftW - rightW - 2 // 1 char padding each side
	if gap < 1 {
		gap = 1
	}

	bar := " " + left + strings.Repeat(" ", gap) + rightStr + " "
	return style.StatusBar.Width(m.width).Render(bar)
}
