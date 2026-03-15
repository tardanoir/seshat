package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tardanoir/seshat/internal/config"
)

const (
	checkInterval = 24 * time.Hour
	releaseURL    = "https://api.github.com/repos/tardanoir/seshat/releases/latest"
)

type cachedCheck struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

type UpdateInfo struct {
	Available bool
	Current   string
	Latest    string
}

// Check compares the current version against the latest GitHub release.
// Results are cached for 24h to avoid hitting the API on every launch.
func Check(current string) UpdateInfo {
	if current == "dev" || current == "" {
		return UpdateInfo{}
	}

	cachePath := filepath.Join(config.ConfigDir(), "version_cache.json")

	// Try cached result first
	if info, ok := readCache(cachePath, current); ok {
		return info
	}

	// Fetch from GitHub
	latest, err := fetchLatest()
	if err != nil {
		return UpdateInfo{}
	}

	// Cache the result
	writeCache(cachePath, latest)

	return compare(current, latest)
}

func readCache(path, current string) (UpdateInfo, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return UpdateInfo{}, false
	}
	var c cachedCheck
	if err := json.Unmarshal(data, &c); err != nil {
		return UpdateInfo{}, false
	}
	if time.Since(c.CheckedAt) > checkInterval {
		return UpdateInfo{}, false
	}
	return compare(current, c.LatestVersion), true
}

func writeCache(path, latest string) {
	data, _ := json.Marshal(cachedCheck{
		LatestVersion: latest,
		CheckedAt:     time.Now(),
	})
	os.WriteFile(path, data, 0o600)
}

func fetchLatest() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(releaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func compare(current, latest string) UpdateInfo {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	if latest == "" || current == latest {
		return UpdateInfo{}
	}

	if newerThan(latest, current) {
		return UpdateInfo{
			Available: true,
			Current:   current,
			Latest:    latest,
		}
	}
	return UpdateInfo{}
}

// newerThan does a simple semver comparison (major.minor.patch).
func newerThan(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) [3]int {
	var v [3]int
	parts := strings.SplitN(s, ".", 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		fmt.Sscanf(p, "%d", &v[i])
	}
	return v
}
