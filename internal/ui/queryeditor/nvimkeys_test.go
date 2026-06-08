package queryeditor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNvimKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		want string
	}{
		{"lowercase", tea.KeyPressMsg{Code: 'a', Text: "a"}, "a"},
		{"uppercase", tea.KeyPressMsg{Code: 'a', Text: "A", Mod: tea.ModShift}, "A"},
		{"digit", tea.KeyPressMsg{Code: '1', Text: "1"}, "1"},
		{"shifted symbol", tea.KeyPressMsg{Code: '!', Text: "!"}, "!"},
		{"literal lt is escaped", tea.KeyPressMsg{Code: '<', Text: "<"}, "<lt>"},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, "<Space>"},
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}, "<CR>"},
		{"escape", tea.KeyPressMsg{Code: tea.KeyEscape}, "<Esc>"},
		{"tab", tea.KeyPressMsg{Code: tea.KeyTab}, "<Tab>"},
		{"backspace", tea.KeyPressMsg{Code: tea.KeyBackspace}, "<BS>"},
		{"up arrow", tea.KeyPressMsg{Code: tea.KeyUp}, "<Up>"},
		{"delete", tea.KeyPressMsg{Code: tea.KeyDelete}, "<Del>"},
		{"f5", tea.KeyPressMsg{Code: tea.KeyF5}, "<F5>"},
		{"ctrl+a", tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}, "<C-a>"},
		{"ctrl+w", tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}, "<C-w>"},
		{"alt+x", tea.KeyPressMsg{Code: 'x', Mod: tea.ModAlt}, "<A-x>"},
		{"shift+tab", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, "<S-Tab>"},
		{"ctrl+left", tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}, "<C-Left>"},
		{"ctrl+alt+a", tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl | tea.ModAlt}, "<C-A-a>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nvimKeys(tt.key); got != tt.want {
				t.Errorf("nvimKeys(%+v) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}
