package config

import (
	"fmt"
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
	ToggleSidebar string `toml:"toggle_sidebar"`
	Tab           string `toml:"tab"`
	ShiftTab      string `toml:"shift_tab"`
	Quit          string `toml:"quit"`
	Delete        string `toml:"delete"`
	Suspend       string `toml:"suspend"`
}

type Config struct {
	DefaultConnection string                `toml:"default_connection"`
	Editor            string                `toml:"editor"`
	Connections       map[string]Connection `toml:"connections"`
	Keys              Keybindings           `toml:"keybindings"`
}

type Connection struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Database string `toml:"database"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	SSLMode  string `toml:"sslmode"`
}

func (c Connection) ConnString() string {
	password := expandEnv(c.Password)
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, password, c.Host, c.Port, c.Database, sslmode)
}

func expandEnv(s string) string {
	if strings.HasPrefix(s, "$") {
		return os.Getenv(strings.TrimPrefix(s, "$"))
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
	return &cfg, nil
}

const defaultConfig = `default_connection = "local"
editor = "nvim"

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
# toggle_sidebar = "ctrl+\\"
# tab = "tab"
# shift_tab = "shift+tab"
# quit = "ctrl+c"
# delete = "ctrl+d"
# suspend = "ctrl+z"
`
