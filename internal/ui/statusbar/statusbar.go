package statusbar

import (
	"strings"

	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/lipgloss/v2"
)

type Model struct {
	width      int
	message    string
	isError    bool
	connName   string
	dbName     string
	focus      string // "query", "results", "sidebar"
	stmtInfo   string // "2/5" etc
	updateHint string // e.g. "0.3.0"
}

func New() Model {
	return Model{message: "Ready"}
}

func (m *Model) SetWidth(w int)           { m.width = w }
func (m *Model) SetMessage(msg string)    { m.message = msg; m.isError = false }
func (m *Model) SetError(msg string)      { m.message = msg; m.isError = true }
func (m *Model) SetFocus(f string)        { m.focus = f }
func (m *Model) SetStmtInfo(s string)     { m.stmtInfo = s }
func (m *Model) SetUpdateHint(ver string) { m.updateHint = ver }

func (m *Model) SetConnection(name, dbName string) {
	m.connName = name
	m.dbName = dbName
}

func (m *Model) SetResult(duration, rowCount string) {
	m.isError = false
	m.message = rowCount + " rows · " + duration
}

func (m *Model) Clear() {
	m.message = "Ready"
	m.isError = false
}

func hint(key, label string) string {
	return style.StatusKey.Render(key) + " " + style.StatusKeyLabel.Render(label)
}

func (m Model) View() string {
	sep := style.StatusSep.Render(" │ ")

	modeLabel := m.focus
	if modeLabel == "" {
		modeLabel = "READY"
	}
	modePill := style.StatusModePill.Render(modeLabel)

	var leftChunks []string
	leftChunks = append(leftChunks, modePill)

	if m.connName != "" {
		connLabel := m.connName
		if m.dbName != "" {
			connLabel += "/" + m.dbName
		}
		leftChunks = append(leftChunks, style.StatusConnPill.Render(connLabel))
	}

	msg := m.message
	if msg != "" {
		if m.isError {
			leftChunks = append(leftChunks, style.Error.Render(msg))
		} else {
			leftChunks = append(leftChunks, style.StatusMsg.Render(msg))
		}
	}

	left := joinChunks(leftChunks, " ")

	var rightChunks []string

	if m.stmtInfo != "" {
		rightChunks = append(rightChunks, style.StatusStmt.Render("stmt "+m.stmtInfo))
	}

	if m.updateHint != "" {
		rightChunks = append(rightChunks, style.StatusUpdatePill.Render("v"+m.updateHint+" available"))
	}

	var hints []string
	switch m.focus {
	case "RESULTS":
		hints = []string{
			hint("y", "cell"),
			hint("Y", "row"),
			hint("c", "col"),
			hint("^x", "export"),
			hint("^r", "run"),
			hint("?", "help"),
		}
	default:
		hints = []string{
			hint("^e", "edit"),
			hint("^r", "run"),
			hint("^w", "save"),
			hint("^x", "export"),
			hint("?", "help"),
		}
	}
	rightChunks = append(rightChunks, strings.Join(hints, "  "))

	right := joinChunks(rightChunks, sep)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := m.width - leftW - rightW - 2
	if gap < 1 {
		gap = 1
	}

	bar := " " + left + strings.Repeat(" ", gap) + right + " "
	return style.StatusBar.Width(m.width).Render(bar)
}

func joinChunks(chunks []string, sep string) string {
	out := make([]string, 0, len(chunks))
	for _, c := range chunks {
		if c != "" {
			out = append(out, c)
		}
	}
	return strings.Join(out, sep)
}
