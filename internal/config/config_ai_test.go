package config

import "testing"

// TestAIProviderCRUD exercises AddAIProvider / SetDefaultAIProvider /
// RemoveAIProvider round-tripping through Save+Load on a temp config dir.
func TestAIProviderCRUD(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// First provider added becomes the default.
	if err := cfg.AddAIProvider("openai", AIProviderConf{Kind: "openai", Model: "gpt", APIKey: "keyring:openai"}); err != nil {
		t.Fatalf("AddAIProvider: %v", err)
	}
	if cfg.AI.DefaultProvider != "openai" {
		t.Errorf("default = %q, want openai", cfg.AI.DefaultProvider)
	}

	// Persisted to disk.
	reloaded, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	p, ok := reloaded.AI.Providers["openai"]
	if !ok {
		t.Fatal("provider not persisted")
	}
	if p.Model != "gpt" || p.APIKey != "keyring:openai" {
		t.Errorf("persisted conf = %+v", p)
	}

	// Switch default.
	if err := reloaded.AddAIProvider("claude", AIProviderConf{Kind: "anthropic"}); err != nil {
		t.Fatalf("AddAIProvider claude: %v", err)
	}
	if err := reloaded.SetDefaultAIProvider("claude"); err != nil {
		t.Fatalf("SetDefaultAIProvider: %v", err)
	}
	if got, _ := Load(); got.AI.DefaultProvider != "claude" {
		t.Errorf("default not persisted: %q", got.AI.DefaultProvider)
	}

	// Removing the default falls back to a remaining provider.
	if err := reloaded.RemoveAIProvider("claude"); err != nil {
		t.Fatalf("RemoveAIProvider: %v", err)
	}
	if reloaded.AI.DefaultProvider != "openai" {
		t.Errorf("fallback default = %q, want openai", reloaded.AI.DefaultProvider)
	}
	final, _ := Load()
	if _, ok := final.AI.Providers["claude"]; ok {
		t.Error("claude not removed on disk")
	}
}
