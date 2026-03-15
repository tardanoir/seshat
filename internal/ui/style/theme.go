package style

import (
	"charm.land/lipgloss/v2"
)

// ANSI colors — inherits from terminal colorscheme
var (
	ColorPrimary   = lipgloss.ANSIColor(5)  // magenta
	ColorSecondary = lipgloss.ANSIColor(13) // bright magenta
	ColorBg        = lipgloss.ANSIColor(0)  // black
	ColorSurface   = lipgloss.ANSIColor(8)  // bright black
	ColorText      = lipgloss.ANSIColor(7)  // white
	ColorSubtext   = lipgloss.ANSIColor(15) // bright white
	ColorBorder    = lipgloss.ANSIColor(8)  // bright black
	ColorSuccess   = lipgloss.ANSIColor(2)  // green
	ColorError     = lipgloss.ANSIColor(1)  // red
	ColorWarning   = lipgloss.ANSIColor(3)  // yellow
	ColorAccent    = lipgloss.ANSIColor(4)  // blue
	ColorCyan      = lipgloss.ANSIColor(6)  // cyan

	// Main pane borders
	Sidebar = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	Editor = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	Results = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	Focused = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	// Sidebar sub-panels (lazygit style)
	PanelBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	PanelBorderActive = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	PanelTitle = lipgloss.NewStyle().
			Foreground(ColorBorder).
			Bold(true)

	PanelTitleActive = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(ColorText)

	StatusConn = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

	StatusMsg = lipgloss.NewStyle().
			Foreground(ColorBorder).
			Italic(true)

	StatusHints = lipgloss.NewStyle().
			Foreground(ColorBorder)

	StatusSep = lipgloss.NewStyle().
			Foreground(ColorBorder)

	StatusFocus = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StatusKey = lipgloss.NewStyle().
			Foreground(ColorText)

	StatusKeyLabel = lipgloss.NewStyle().
				Foreground(ColorBorder)

	StatusUpdate = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Italic(true)

	// Modals
	Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	ModalOverlay = lipgloss.NewStyle().
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	Label = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(ColorError)

	Success = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	// List items
	ListItem = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	ListSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			PaddingLeft(1)

	// Results table
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder)

	TableCell = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	TableSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorSurface).
			Padding(0, 1)

	TableNull = lipgloss.NewStyle().
			Foreground(ColorBorder).
			Italic(true).
			Padding(0, 1)

	// Sidebar table browser
	TableName = lipgloss.NewStyle().
			Foreground(ColorCyan).
			PaddingLeft(2)

	TableNameSelected = lipgloss.NewStyle().
				Foreground(ColorCyan).
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
			Foreground(ColorBorder).
			Italic(true)
)
