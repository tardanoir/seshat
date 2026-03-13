package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type ContentMsg struct {
	Content string
	Err     error
}

type editorInfo struct {
	bin  string
	args []string
	gui  bool // true for editors that don't need the terminal
}

// resolveEditor returns the binary, extra args, and whether it's a GUI editor.
func resolveEditor(editor, filePath string) editorInfo {
	base := filepath.Base(editor)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	switch base {
	case "code", "vscode", "cursor":
		return editorInfo{editor, []string{"--wait", filePath}, true}
	case "zed", "zeditor":
		return editorInfo{editor, []string{"--wait", filePath}, true}
	case "subl", "sublime_text":
		return editorInfo{editor, []string{"--wait", filePath}, true}
	case "atom":
		return editorInfo{editor, []string{"--wait", filePath}, true}
	case "gedit":
		return editorInfo{editor, []string{"--wait", filePath}, true}
	case "kate":
		return editorInfo{editor, []string{"--block", filePath}, true}
	default:
		return editorInfo{editor, []string{filePath}, false}
	}
}

func Open(editor, content string) tea.Cmd {
	tmpDir := os.TempDir()
	f, err := os.CreateTemp(tmpDir, "seshat-query-*.sql")
	if err != nil {
		return func() tea.Msg { return ContentMsg{Err: err} }
	}
	tmpPath := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return func() tea.Msg { return ContentMsg{Err: err} }
	}
	f.Close()

	info := resolveEditor(editor, tmpPath)

	if info.gui {
		// GUI editors: run in background, wait for exit, then read file.
		return func() tea.Msg {
			cmd := exec.Command(info.bin, info.args...)
			err := cmd.Run()
			defer os.Remove(tmpPath)
			if err != nil {
				return ContentMsg{Err: err}
			}
			data, err := os.ReadFile(tmpPath)
			if err != nil {
				return ContentMsg{Err: err}
			}
			return ContentMsg{Content: string(data)}
		}
	}

	c := exec.Command(info.bin, info.args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return ContentMsg{Err: err}
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return ContentMsg{Err: err}
		}
		return ContentMsg{Content: string(data)}
	})
}
