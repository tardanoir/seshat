package editor

import (
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

type ContentMsg struct {
	Content string
	Err     error
}

func Open(editor, content string) tea.Cmd {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "seshat-query-*.sql")
	f, err := os.CreateTemp(tmpDir, filepath.Base(tmpFile))
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

	c := exec.Command(editor, tmpPath)
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
