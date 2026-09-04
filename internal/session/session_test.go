package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathIsStableAndDistinct(t *testing.T) {
	a := Path("/home/u/projects/api")
	b := Path("/home/u/projects/api")
	c := Path("/home/u/other/api")

	if a != b {
		t.Fatalf("same dir produced different paths: %s vs %s", a, b)
	}
	if a == c {
		t.Fatalf("different dirs with the same basename collided: %s", a)
	}
	if got := filepath.Base(a); got[:4] != "api-" {
		t.Fatalf("path %q does not keep a readable prefix", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := "/some/work/dir"

	want := Session{
		Dir:            dir,
		Query:          "select 1;",
		Connection:     "local",
		SidebarVisible: true,
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil for a saved session")
	}
	if got.Query != want.Query || got.Connection != want.Connection || !got.SidebarVisible {
		t.Fatalf("round trip mismatch: %+v", *got)
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not stamped on save")
	}
}

func TestLoadMissingReturnsNil(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Load("/never/written")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a missing session, got %+v", *got)
	}
}

func TestLoadRejectsMismatchedDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Stand in for a hash collision: a file living under /c/d's key whose
	// recorded Dir belongs to another directory.
	body := `{"dir":"/a/b","query":"select 1;"}`
	if err := os.WriteFile(Path("/c/d"), []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load("/c/d")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a mismatched dir, got %+v", *got)
	}
}
