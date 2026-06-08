package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Keybindings struct {
	Execute       string `toml:"execute"`
	Editor        string `toml:"editor"`
	Save          string `toml:"save"`
	Template      string `toml:"template"`
	ConnPick      string `toml:"connection"`
	History       string `toml:"history"`
	ToggleSidebar string `toml:"toggle_sidebar"`
	Export        string `toml:"export"`
	Tab           string `toml:"tab"`
	ShiftTab      string `toml:"shift_tab"`
	Quit          string `toml:"quit"`
	Delete        string `toml:"delete"`
	Suspend       string `toml:"suspend"`
	AIGenerate    string `toml:"ai_generate"`
}

type Config struct {
	DefaultConnection string                `toml:"default_connection"`
	Editor            string                `toml:"editor"`
	VimMode           bool                  `toml:"vim_mode"`
	MaxRows           int                   `toml:"max_rows"`
	Connections       map[string]Connection `toml:"connections"`
	Keys              Keybindings           `toml:"keybindings"`
	AI                AIConfig              `toml:"ai"`
}

type AIConfig struct {
	DefaultProvider string                    `toml:"default_provider"`
	Providers       map[string]AIProviderConf `toml:"providers"`
}

type AIProviderConf struct {
	Kind     string   `toml:"kind"`
	APIKey   string   `toml:"api_key"`
	Model    string   `toml:"model"`
	BaseURL  string   `toml:"base_url"`
	Argv     []string `toml:"argv"`
	TimeoutS int      `toml:"timeout_seconds"`
}

// TODO: right now I have a way to test the GCP and AWS approaches, still need to test other clouds.
type SSHTunnel struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Key      string `toml:"key"`
	Password string `toml:"password"`
}

type Connection struct {
	Type     string     `toml:"type"`
	Host     string     `toml:"host"`
	Port     int        `toml:"port"`
	Database string     `toml:"database"`
	User     string     `toml:"user"`
	Password string     `toml:"password"`
	SSLMode  string     `toml:"ssl_mode"`
	Path     string     `toml:"path"`
	SSH      *SSHTunnel `toml:"ssh"`
}

// Set the type variable for diff types of db's
func (c Connection) DriverType() string {
	if c.Type != "" {
		return c.Type
	}
	if c.Path != "" {
		ext := strings.ToLower(filepath.Ext(c.Path))
		switch ext {
		case ".csv":
			return "csv"
		case ".json":
			return "json"
		default:
			return "sqlite"
		}
	}
	return "postgres"
}

func (c Connection) ConnString() string {
	switch c.DriverType() {
	case "sqlite", "csv", "json":
		return expandPath(c.Path)
	default:
		password := expandEnv(c.Password)
		sslmode := c.SSLMode
		if sslmode == "" {
			sslmode = "disable"
		}
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.User, url.QueryEscape(password), c.Host, c.Port, c.Database, sslmode)
	}
}

// ConnStringVia returns a connection string that routes through a local tunnel
// address (e.g. "127.0.0.1:54321") instead of the configured host:port.
func (c Connection) ConnStringVia(localAddr string) string {
	password := expandEnv(c.Password)
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		c.User, url.QueryEscape(password), localAddr, c.Database, sslmode)
}

func (c Connection) IsFile() bool {
	switch c.DriverType() {
	case "sqlite", "csv", "json":
		return true
	}
	return false
}

// HasSSH returns true if the connection has SSH tunnel config.
func (c Connection) HasSSH() bool {
	return c.SSH != nil && c.SSH.Host != ""
}

// DisplayLabel returns a human-readable label for the connection.
func (c Connection) DisplayLabel() string {
	if c.IsFile() {
		return filepath.Base(c.Path)
	}
	label := fmt.Sprintf("%s:%d/%s", c.Host, c.Port, c.Database)
	if c.HasSSH() {
		label += fmt.Sprintf(" via %s@%s", c.SSH.User, c.SSH.Host)
	}
	return label
}

func expandEnv(s string) string {
	if v, ok := strings.CutPrefix(s, "$"); ok {
		return os.Getenv(v)
	}
	return s
}

func expandPath(s string) string {
	s = expandEnv(s)
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			s = filepath.Join(home, s[2:])
		}
	}
	return s
}

func ConfigDir() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "seshat")
}

func Bootstrap() error {
	base := ConfigDir()
	for _, sub := range []string{"queries", "templates"} {
		if err := os.MkdirAll(filepath.Join(base, sub), 0o755); err != nil {
			return err
		}
	}
	cfgPath := filepath.Join(base, "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return os.WriteFile(cfgPath, []byte(defaultConfig), 0o644)
	}
	return nil
}

func Load() (*Config, error) {
	cfgPath := filepath.Join(ConfigDir(), "config.toml")
	var cfg Config
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("EDITOR")
		if cfg.Editor == "" {
			cfg.Editor = "vi"
		}
	}
	if cfg.Connections == nil {
		cfg.Connections = make(map[string]Connection)
	}
	if cfg.MaxRows <= 0 {
		cfg.MaxRows = 10000
	}
	return &cfg, nil
}

// save writes the config back to disk.
func (cfg *Config) Save() error {
	cfgPath := filepath.Join(ConfigDir(), "config.toml")
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(cfgPath, buf.Bytes(), 0o644)
}

// AddConnection adds a new conn.
func (cfg *Config) AddConnection(name string, conn Connection) error {
	if cfg.Connections == nil {
		cfg.Connections = make(map[string]Connection)
	}
	cfg.Connections[name] = conn
	return cfg.Save()
}

// RemoveConnection removes a conn.
func (cfg *Config) RemoveConnection(name string) error {
	delete(cfg.Connections, name)
	if cfg.DefaultConnection == name {
		cfg.DefaultConnection = ""
	}
	return cfg.Save()
}

const defaultConfig = `default_connection = "local"
editor = "nvim"
vim_mode = false
# max_rows = 10000

[connections.local]
host = "localhost"
port = 5432
database = "postgres"
user = "postgres"
password = ""

# [keybindings]
# execute = "ctrl+r"
# editor = "ctrl+e"
# save = "ctrl+w"
# template = "ctrl+t"
# connection = "ctrl+n"
# history = "ctrl+h"
# toggle_sidebar = "ctrl+\\"
# tab = "tab"
# shift_tab = "shift+tab"
# quit = "ctrl+c"
# delete = "ctrl+d"
# suspend = "ctrl+z"
# ai_generate = "ctrl+a"

# AI SQL generation. Place the cursor on a "-- ..." comment in the editor and
# press the AI hotkey to ask the configured provider to generate SQL. The
# response opens in a modal you can accept or reject.
#
# api_key supports "$ENV_VAR" to read from the environment.
# argv supports "{prompt}" as a placeholder; without it, the prompt is sent
# on stdin instead.
#
# [ai]
# default_provider = "anthropic"
#
# [ai.providers.anthropic]
# kind = "anthropic"
# api_key = "$ANTHROPIC_API_KEY"
# model = "claude-sonnet-4-5"
#
# [ai.providers.openai]
# kind = "openai"
# api_key = "$OPENAI_API_KEY"
# model = "gpt-4.1-mini"
#
# [ai.providers.gemini]
# kind = "gemini"
# api_key = "$GEMINI_API_KEY"
# model = "gemini-2.5-flash"
#
# [ai.providers.claude_cli]
# kind = "cli"
# argv = ["claude", "-p", "{prompt}"]
#
# [ai.providers.opencode_cli]
# kind = "cli"
# argv = ["opencode", "run", "{prompt}"]
`
