package ui

import (
	"testing"

	"github.com/tardanoir/seshat/internal/config"

	tea "charm.land/bubbletea/v2"
)

// TestHistoryKeyOpensPickerFromAnyFocus verifies that the history shortcut
// (Ctrl+H) opens the picker regardless of which panel is focused — including
// the side panel.
func TestHistoryKeyOpensPickerFromAnyFocus(t *testing.T) {
	cfg := &config.Config{
		VimMode:     false,
		Connections: map[string]config.Connection{},
	}
	ctrlH := tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl}

	cases := []struct {
		name  string
		focus Focus
	}{
		{"sidebar", FocusSidebar},
		{"results", FocusResults},
		{"preview", FocusPreview},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := NewApp(cfg, "test", nil, "")
			a.focus = tc.focus

			m, _ := a.Update(ctrlH)
			got := m.(App)
			if got.modalState != ModalHistory {
				t.Fatalf("focus %s: modalState = %d, want ModalHistory (%d)", tc.name, got.modalState, ModalHistory)
			}
		})
	}
}
