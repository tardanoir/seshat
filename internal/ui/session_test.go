package ui

import (
	"os"
	"testing"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/session"
)

func TestNewAppRestoresSession(t *testing.T) {
	cfg := &config.Config{Connections: map[string]config.Connection{}}
	sess := &session.Session{
		Dir:            "/work/api",
		Query:          "select * from users;",
		Connection:     "local",
		SidebarVisible: false,
	}

	a := NewApp(cfg, "test", sess, "/work/api")

	if got := a.preview.Value(); got != sess.Query {
		t.Errorf("editor buffer = %q, want %q", got, sess.Query)
	}
	if a.sidebarVisible {
		t.Error("sidebarVisible = true, want false from the restored session")
	}
}

func TestNewAppWithoutSessionKeepsDefaults(t *testing.T) {
	cfg := &config.Config{Connections: map[string]config.Connection{}}

	a := NewApp(cfg, "test", nil, "")

	if got := a.preview.Value(); got != "" {
		t.Errorf("editor buffer = %q, want empty", got)
	}
	if !a.sidebarVisible {
		t.Error("sidebarVisible = false, want true by default")
	}
}

func TestSaveSessionRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := "/work/etl"

	cfg := &config.Config{Connections: map[string]config.Connection{}}
	a := NewApp(cfg, "test", nil, dir)
	a.preview.SetValue("select 42;")
	a.connName = "warehouse"
	a.sidebarVisible = false
	a.saveSession()

	got, err := session.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("saveSession wrote nothing")
	}
	if got.Query != "select 42;" {
		t.Errorf("Query = %q, want %q", got.Query, "select 42;")
	}
	if got.Connection != "warehouse" {
		t.Errorf("Connection = %q, want %q", got.Connection, "warehouse")
	}
	if got.SidebarVisible {
		t.Error("SidebarVisible = true, want false")
	}
}

// An empty sessionDir means persistence is off; saving must be a no-op rather
// than writing to some default location.
func TestSaveSessionDisabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := &config.Config{Connections: map[string]config.Connection{}}
	a := NewApp(cfg, "test", nil, "")
	a.preview.SetValue("select 1;")
	a.saveSession()

	if _, err := session.Load("/anything"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, _ := readDirNames(session.Dir())
	if len(entries) != 0 {
		t.Errorf("expected no session files, got %v", entries)
	}
}

func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
