package ai

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the AI section of the application config (see internal/config).
// It is duplicated here as a small struct so the ai package has no dependency
// on internal/config.
type Config struct {
	DefaultProvider string
	Providers       map[string]ProviderConf
}

type ProviderConf struct {
	Kind     string
	APIKey   string
	Model    string
	BaseURL  string
	Argv     []string
	TimeoutS int
}

// Builder builds a concrete Provider for the given config.
// It is injected by the UI layer to keep this package free of provider deps.
type Builder func(cfg Config) (Provider, error)

// ResolveAPIKey resolves "$VAR" prefixes against the environment, mirroring
// internal/config.expandEnv. Plain strings pass through unchanged.
func ResolveAPIKey(s string) string {
	if v, ok := strings.CutPrefix(s, "$"); ok {
		return os.Getenv(v)
	}
	return s
}

// Timeout returns the configured timeout, or 0 when unset.
func (p ProviderConf) Timeout() time.Duration {
	if p.TimeoutS <= 0 {
		return 0
	}
	return time.Duration(p.TimeoutS) * time.Second
}

// SelectProvider extracts the active provider configuration from cfg.
// Errors are returned when default_provider is unset or doesn't match any
// configured provider.
func SelectProvider(cfg Config) (string, ProviderConf, error) {
	if cfg.DefaultProvider == "" {
		return "", ProviderConf{}, errors.New("ai.default_provider is not set")
	}
	conf, ok := cfg.Providers[cfg.DefaultProvider]
	if !ok {
		return "", ProviderConf{}, fmt.Errorf("ai provider %q not configured", cfg.DefaultProvider)
	}
	return cfg.DefaultProvider, conf, nil
}
