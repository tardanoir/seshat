package providers

import (
	"fmt"

	"github.com/tardanoir/seshat/internal/ai"
)

// Build returns the concrete provider selected by cfg.DefaultProvider.
// Kinds: "anthropic", "openai", "gemini", "cli".
func Build(cfg ai.Config) (ai.Provider, error) {
	name, conf, err := ai.SelectProvider(cfg)
	if err != nil {
		return nil, err
	}
	timeout := conf.Timeout()
	switch conf.Kind {
	case "anthropic":
		return NewAnthropic(ai.ResolveAPIKey(conf.APIKey), conf.Model, conf.BaseURL, timeout), nil
	case "openai":
		return NewOpenAI(ai.ResolveAPIKey(conf.APIKey), conf.Model, conf.BaseURL, timeout), nil
	case "gemini":
		return NewGemini(ai.ResolveAPIKey(conf.APIKey), conf.Model, conf.BaseURL, timeout), nil
	case "cli":
		return NewCLI(name, conf.Argv, timeout), nil
	default:
		return nil, fmt.Errorf("unknown ai provider kind %q", conf.Kind)
	}
}
