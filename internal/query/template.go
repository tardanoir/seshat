package query

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Template struct {
	Name        string
	Description string
	Variables   map[string]TemplateVar
	SQL         string
	Path        string
}

type TemplateVar struct {
	Description string `toml:"description"`
	Default     string `toml:"default"`
}

type templateFrontmatter struct {
	Name        string                  `toml:"name"`
	Description string                  `toml:"description"`
	Variables   map[string]TemplateVar  `toml:"variables"`
}

var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

func ListTemplates() ([]Template, error) {
	dir := TemplatesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var templates []Template
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		t, err := ParseTemplate(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		templates = append(templates, *t)
	}
	return templates, nil
}

func ParseTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	const delim = "+++"
	parts := strings.SplitN(content, delim, 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid template format: missing +++ frontmatter delimiters")
	}

	var fm templateFrontmatter
	if _, err := toml.Decode(parts[1], &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	sql := strings.TrimSpace(parts[2])

	return &Template{
		Name:        fm.Name,
		Description: fm.Description,
		Variables:   fm.Variables,
		SQL:         sql,
		Path:        path,
	}, nil
}

func ExtractVariables(sql string) []string {
	matches := varPattern.FindAllStringSubmatch(sql, -1)
	seen := make(map[string]bool)
	var vars []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			vars = append(vars, m[1])
		}
	}
	return vars
}

func Substitute(sql string, values map[string]string) string {
	return varPattern.ReplaceAllStringFunc(sql, func(match string) string {
		name := match[2 : len(match)-2]
		if val, ok := values[name]; ok {
			return val
		}
		return match
	})
}
