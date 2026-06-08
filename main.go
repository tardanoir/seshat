package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/ui"
	"github.com/tardanoir/seshat/internal/ui/style"

	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("seshat %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	if err := config.Bootstrap(); err != nil {
		fmt.Fprintf(os.Stderr, "seshat: bootstrap error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "seshat: config error: %v\n", err)
		os.Exit(1)
	}

	// seshat + filePath will open the file
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		filePath := os.Args[1]
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "seshat: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stat(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "seshat: %v\n", err)
			os.Exit(1)
		}
		conn := config.Connection{Path: absPath}
		name := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		cfg.Connections[name] = conn
		cfg.DefaultConnection = name
	}

	style.ApplyKeybindings(style.Keybindings{
		Execute:       cfg.Keys.Execute,
		Editor:        cfg.Keys.Editor,
		Save:          cfg.Keys.Save,
		Template:      cfg.Keys.Template,
		ConnPick:      cfg.Keys.ConnPick,
		History:       cfg.Keys.History,
		ToggleSidebar: cfg.Keys.ToggleSidebar,
		Export:        cfg.Keys.Export,
		Tab:           cfg.Keys.Tab,
		ShiftTab:      cfg.Keys.ShiftTab,
		Quit:          cfg.Keys.Quit,
		Delete:        cfg.Keys.Delete,
		Suspend:       cfg.Keys.Suspend,
		AIGenerate:    cfg.Keys.AIGenerate,
	})

	app := ui.NewApp(cfg, version)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "seshat: %v\n", err)
		os.Exit(1)
	}
}
