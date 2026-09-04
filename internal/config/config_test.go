package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapCreatesSessionsDirAndDefaultsOn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)

	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ConfigDir(), "sessions")); err != nil {
		t.Fatalf("sessions dir not created: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.PersistSessions {
		t.Error("PersistSessions = false, want true from the generated default config")
	}
}

func TestPersistSessionsDefaultsTrueWhenAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := ConfigDir()
	os.MkdirAll(dir, 0o755)
	// A pre-existing config from before the feature landed: no key at all.
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("default_connection = \"local\"\n"), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.PersistSessions {
		t.Error("PersistSessions = false for an old config, want true")
	}
}

func TestPersistSessionsRespectsExplicitFalse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := ConfigDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("persist_sessions = false\n"), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PersistSessions {
		t.Error("PersistSessions = true, want false when explicitly disabled")
	}
}
