package query

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tardanoir/seshat/internal/config"
)

type SavedQuery struct {
	Name    string
	Content string
	Path    string
}

func QueriesDir() string {
	return filepath.Join(config.ConfigDir(), "queries")
}

func TemplatesDir() string {
	return filepath.Join(config.ConfigDir(), "templates")
}

func List() ([]SavedQuery, error) {
	dir := QueriesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var queries []SavedQuery
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".sql")
		queries = append(queries, SavedQuery{
			Name:    name,
			Content: string(data),
			Path:    filepath.Join(dir, e.Name()),
		})
	}
	return queries, nil
}

func Save(name, content string) error {
	safe := sanitizeFilename(name)
	path := filepath.Join(QueriesDir(), safe+".sql")
	return os.WriteFile(path, []byte(content), 0o644)
}

func Delete(name string) error {
	safe := sanitizeFilename(name)
	path := filepath.Join(QueriesDir(), safe+".sql")
	return os.Remove(path)
}

func Load(name string) (string, error) {
	safe := sanitizeFilename(name)
	path := filepath.Join(QueriesDir(), safe+".sql")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}
