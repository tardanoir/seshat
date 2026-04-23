package style

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func FocusBar(focused bool) string {
	if focused {
		return FocusBarActive.Render("▎")
	}
	return FocusBarInactive.Render(" ")
}

func PrefixFocusBar(content string, focused bool) string {
	bar := FocusBar(focused)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = bar + line
	}
	return strings.Join(lines, "\n")
}

var (
	ColorBg      = lipgloss.ANSIColor(0)
	ColorSurface = ColorBg

	ColorText    = lipgloss.ANSIColor(7)
	ColorSubtext = lipgloss.ANSIColor(8)
	ColorMuted   = lipgloss.ANSIColor(8)

	ColorBorder = lipgloss.ANSIColor(8)

	ColorPrimary   = lipgloss.ANSIColor(3)
	ColorSecondary = lipgloss.ANSIColor(11)
	ColorAccent    = lipgloss.ANSIColor(3)

	ColorError   = lipgloss.ANSIColor(1)
	ColorSuccess = lipgloss.ANSIColor(2)
	ColorWarning = lipgloss.ANSIColor(3)

	ColorCyan = lipgloss.ANSIColor(4)

	ColorHighlight = lipgloss.Color("236")
)

var (
	Sidebar = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 2, 0, 1)

	Editor = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 2, 0, 1)

	Results = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 2, 0, 1)

	FrameStyle = lipgloss.NewStyle().Foreground(ColorBorder)

	ConnLabel = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ConnDot = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	ConnName = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	FocusBarActive = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	FocusBarInactive = lipgloss.NewStyle()

	PanelTitle = lipgloss.NewStyle().
			Foreground(ColorSubtext)

	PanelTitleFocused = lipgloss.NewStyle().
				Foreground(ColorText).
				Bold(true)

	PanelTitleActive = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	PanelBorder       = PanelTitle
	PanelBorderActive = PanelTitleActive

	TitleSep = lipgloss.NewStyle().Foreground(ColorMuted)

	StatusBar = lipgloss.NewStyle().Foreground(ColorText)

	StatusModePill = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)

	StatusConnPill = lipgloss.NewStyle().
			Background(ColorCyan).
			Foreground(ColorBg).
			Padding(0, 1)

	StatusUpdatePill = lipgloss.NewStyle().
				Background(ColorWarning).
				Foreground(ColorBg).
				Padding(0, 1)

	StatusMsg = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Italic(true)

	StatusStmt = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	StatusHints = lipgloss.NewStyle().Foreground(ColorSubtext)

	StatusSep = lipgloss.NewStyle().Foreground(ColorBorder)

	StatusKey = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	StatusKeyLabel = lipgloss.NewStyle().Foreground(ColorSubtext)

	StatusConn   = StatusModePill
	StatusFocus  = StatusModePill
	StatusUpdate = lipgloss.NewStyle().Foreground(ColorWarning).Italic(true)

		Foreground(ColorPrimary).
		Bold(true)

	ModalOverlay = lipgloss.NewStyle().
			Background(ColorBg).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			BorderBackground(ColorBg).
			Padding(1, 2)

	Label = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	Error = lipgloss.NewStyle().Foreground(ColorError)

	Success = lipgloss.NewStyle().Foreground(ColorSuccess)

	ListItem = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	ListSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			PaddingLeft(1)

	TableHeader = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Padding(0, 1)

	TableCell = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	TableSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	TableNull = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Padding(0, 1)

	TableName = lipgloss.NewStyle().
			Foreground(ColorCyan).
			PaddingLeft(2)

	TableNameSelected = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(1)

	ColumnItem = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(4)

	ColumnItemSelected = lipgloss.NewStyle().
				Foreground(ColorText).
				Bold(true).
				PaddingLeft(3)

	ColumnType = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Italic(true)
)
