package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		name      string
		current   string
		latest    string
		available bool
	}{
		{"newer patch", "1.1.2", "1.1.3", true},
		{"newer minor", "1.1.2", "1.2.0", true},
		{"newer major", "1.1.2", "2.0.0", true},
		{"same", "1.2.0", "1.2.0", false},
		{"older latest", "1.2.0", "1.1.2", false},
		{"v prefix on both", "v1.1.2", "v1.2.0", true},
		{"v prefix on latest only", "1.1.2", "v1.2.0", true},
		{"empty latest", "1.1.2", "", false},
		{"10 beats 9 numerically", "1.9.0", "1.10.0", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compare(tc.current, tc.latest)
			if got.Available != tc.available {
				t.Fatalf("compare(%q, %q).Available = %v, want %v",
					tc.current, tc.latest, got.Available, tc.available)
			}
			if tc.available && got.Latest != "1.2.0" && got.Latest == "" {
				t.Errorf("Available result carried no Latest version")
			}
		})
	}
}

// A dev build must never nag about updates.
func TestCheckSkipsDevBuilds(t *testing.T) {
	for _, v := range []string{"dev", ""} {
		if got := Check(v); got.Available {
			t.Errorf("Check(%q).Available = true, want false", v)
		}
	}
}

func TestCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version_cache.json")
	writeCache(path, "v1.2.0")

	info, ok := readCache(path, "1.1.2")
	if !ok {
		t.Fatal("readCache returned ok=false for a freshly written cache")
	}
	if !info.Available || info.Latest != "1.2.0" {
		t.Fatalf("readCache = %+v, want an available update to 1.2.0", info)
	}
}

// A cache older than the check interval must be ignored so the app re-fetches.
func TestStaleCacheIsIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version_cache.json")
	data, _ := json.Marshal(cachedCheck{
		LatestVersion: "v1.2.0",
		CheckedAt:     time.Now().Add(-checkInterval - time.Hour),
	})
	os.WriteFile(path, data, 0o600)

	if _, ok := readCache(path, "1.1.2"); ok {
		t.Error("readCache accepted a cache older than checkInterval")
	}
}

func TestMissingAndCorruptCache(t *testing.T) {
	dir := t.TempDir()
	if _, ok := readCache(filepath.Join(dir, "nope.json"), "1.1.2"); ok {
		t.Error("readCache returned ok=true for a missing file")
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o600)
	if _, ok := readCache(bad, "1.1.2"); ok {
		t.Error("readCache returned ok=true for a corrupt file")
	}
}

func TestParseSemver(t *testing.T) {
	cases := map[string][3]int{
		"1.2.3":     {1, 2, 3},
		"1.2":       {1, 2, 0},
		"1":         {1, 0, 0},
		"1.10.0":    {1, 10, 0},
		"1.2.3-rc1": {1, 2, 3},
	}
	for in, want := range cases {
		if got := parseSemver(in); got != want {
			t.Errorf("parseSemver(%q) = %v, want %v", in, got, want)
		}
	}
}
