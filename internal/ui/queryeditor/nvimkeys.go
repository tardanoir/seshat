package queryeditor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// specialNames maps Bubble Tea key codes to Neovim key-notation names (the part
// that goes inside <...>).
var specialNames = map[rune]string{
	tea.KeyEnter:     "CR",
	tea.KeyTab:       "Tab",
	tea.KeyEscape:    "Esc",
	tea.KeyBackspace: "BS",
	tea.KeySpace:     "Space",
	tea.KeyUp:        "Up",
	tea.KeyDown:      "Down",
	tea.KeyLeft:      "Left",
	tea.KeyRight:     "Right",
	tea.KeyHome:      "Home",
	tea.KeyEnd:       "End",
	tea.KeyPgUp:      "PageUp",
	tea.KeyPgDown:    "PageDown",
	tea.KeyInsert:    "Insert",
	tea.KeyDelete:    "Del",
	tea.KeyF1:        "F1",
	tea.KeyF2:        "F2",
	tea.KeyF3:        "F3",
	tea.KeyF4:        "F4",
	tea.KeyF5:        "F5",
	tea.KeyF6:        "F6",
	tea.KeyF7:        "F7",
	tea.KeyF8:        "F8",
	tea.KeyF9:        "F9",
	tea.KeyF10:       "F10",
	tea.KeyF11:       "F11",
	tea.KeyF12:       "F12",
}

// nvimKeys translates a Bubble Tea key event into the key notation accepted by
// nvim_input. Printable characters pass through verbatim (so shifted symbols and
// capitals just work); special keys and modifier combos become <...> tokens.
func nvimKeys(msg tea.KeyMsg) string {
	k := msg.Key()
	ctrl := k.Mod&tea.ModCtrl != 0
	alt := k.Mod&tea.ModAlt != 0
	shift := k.Mod&tea.ModShift != 0

	name, special := specialNames[k.Code]

	// Plain printable text without ctrl/alt: send as-is. Text already reflects
	// shift state ("A", "!", ...). "<" must be escaped so it isn't read as the
	// start of a key code.
	if !ctrl && !alt && !special && k.Text != "" {
		return escapeLt(k.Text)
	}

	var base string
	switch {
	case special:
		base = name
	case k.Text != "":
		base = k.Text
	case k.Code > 0:
		base = string(k.Code)
	default:
		return ""
	}

	var mods string
	if ctrl {
		mods += "C-"
	}
	if alt {
		mods += "A-"
	}
	if shift && special {
		// Shift on a printable char is already encoded in Text; only meaningful
		// for named keys like <S-Tab>.
		mods += "S-"
	}

	if mods == "" {
		if special {
			return "<" + base + ">"
		}
		return escapeLt(base)
	}
	return "<" + mods + base + ">"
}

func escapeLt(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	return strings.ReplaceAll(s, "<", "<lt>")
}
