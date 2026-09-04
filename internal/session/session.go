// Package session persists per-directory editor state so reopening seshat in
// the same working directory restores the query you were writing and the
// connection you were using.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tardanoir/seshat/internal/config"
)

type Session struct {
	Dir            string    `json:"dir"`
	Query          string    `json:"query"`
	Connection     string    `json:"connection"`
	SidebarVisible bool      `json:"sidebar_visible"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func Dir() string {
	return filepath.Join(config.ConfigDir(), "sessions")
}

// CurrentDir returns the key for the running process: the working directory
// with symlinks resolved, so /home/x and a symlinked path to it share a session.
func CurrentDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		return resolved
	}
	return cwd
}

// Path returns the session file for dir: a readable prefix so the directory can
// be browsed by hand, plus a hash of the full path to keep it unique.
func Path(dir string) string {
	sum := sha256.Sum256([]byte(dir))
	return filepath.Join(Dir(), slug(filepath.Base(dir))+"-"+hex.EncodeToString(sum[:6])+".json")
}

// Load returns the stored session for dir, or nil if there is none.
func Load(dir string) (*Session, error) {
	if dir == "" {
		return nil, nil
	}
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	// Guard against a hash collision handing back another directory's state.
	if s.Dir != "" && s.Dir != dir {
		return nil, nil
	}
	return &s, nil
}

func Save(s Session) error {
	if s.Dir == "" {
		return nil
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(s.Dir), data, 0o600)
}

func slug(name string) string {
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			sb.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			sb.WriteRune(r + 32)
		default:
			sb.WriteByte('_')
		}
	}
	s := sb.String()
	if s == "" {
		return "dir"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
