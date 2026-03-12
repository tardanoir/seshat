package main

import (
	"fmt"
	"os"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/ui"
	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if err := config.Bootstrap(); err != nil {
		fmt.Fprintf(os.Stderr, "seshat: bootstrap error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "seshat: config error: %v\n", err)
		os.Exit(1)
	}

	style.ApplyKeybindings(style.Keybindings{
		Execute:       cfg.Keys.Execute,
		Editor:        cfg.Keys.Editor,
		Save:          cfg.Keys.Save,
		Template:      cfg.Keys.Template,
		ConnPick:      cfg.Keys.ConnPick,
		ToggleSidebar: cfg.Keys.ToggleSidebar,
		Tab:           cfg.Keys.Tab,
		ShiftTab:      cfg.Keys.ShiftTab,
		Quit:          cfg.Keys.Quit,
		Delete:        cfg.Keys.Delete,
	})

	app := ui.NewApp(cfg)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "seshat: %v\n", err)
		os.Exit(1)
	}
}
