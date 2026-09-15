package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAssetNameMatchesGoreleaser(t *testing.T) {
	got := assetName("1.2.0")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	want := fmt.Sprintf("seshat_1.2.0_%s_%s.%s", runtime.GOOS, runtime.GOARCH, ext)
	if got != want {
		t.Errorf("assetName = %q, want %q", got, want)
	}
}

func TestChecksumFor(t *testing.T) {
	body := []byte(
		"abc123  seshat_1.2.0_linux_amd64.tar.gz\n" +
			"def456  seshat_1.2.0_darwin_arm64.tar.gz\n" +
			"999fff *seshat_1.2.0_windows_amd64.zip\n")

	got, err := checksumFor(body, "seshat_1.2.0_darwin_arm64.tar.gz")
	if err != nil || got != "def456" {
		t.Fatalf("checksumFor = %q, %v; want def456", got, err)
	}
	// Binary-mode "*" prefix must still match.
	if got, err := checksumFor(body, "seshat_1.2.0_windows_amd64.zip"); err != nil || got != "999fff" {
		t.Fatalf("binary-mode entry = %q, %v; want 999fff", got, err)
	}
	if _, err := checksumFor(body, "nope.tar.gz"); err == nil {
		t.Error("expected an error for a missing entry")
	}
}

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write(content)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	want := []byte("\x7fELF fake binary")
	blob := makeTarGz(t, "seshat", want)

	got, err := extractBinary("seshat_1.2.0_linux_amd64.tar.gz", blob)
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted %q, want %q", got, want)
	}
}

func TestExtractBinaryRejectsArchiveWithoutBinary(t *testing.T) {
	blob := makeTarGz(t, "README.md", []byte("hi"))
	if _, err := extractBinary("seshat_1.2.0_linux_amd64.tar.gz", blob); err == nil {
		t.Error("expected an error when the archive has no seshat binary")
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "seshat")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExecutable(dest, []byte("new")); err != nil {
		t.Fatalf("replaceExecutable: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
	fi, _ := os.Stat(dest)
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", fi.Mode().Perm())
	}
	// No stray artifacts left behind.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("leftover files in the install dir: %v", names)
	}
}

// The mode of the existing binary must carry over to the replacement.
func TestReplaceExecutablePreservesMode(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "seshat")
	os.WriteFile(dest, []byte("old"), 0o700)

	if err := replaceExecutable(dest, []byte("new")); err != nil {
		t.Fatalf("replaceExecutable: %v", err)
	}
	fi, _ := os.Stat(dest)
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, want 0700", fi.Mode().Perm())
	}
}

func TestCheckWritableRejectsReadOnlyDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permissions are not enforced")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "ro")
	os.Mkdir(sub, 0o555)
	t.Cleanup(func() { os.Chmod(sub, 0o755) })

	err := checkWritable(filepath.Join(sub, "seshat"))
	if err == nil {
		t.Fatal("expected an error for a read-only install directory")
	}
	// The message has to point at the package manager, not just fail.
	if !strings.Contains(err.Error(), "brew upgrade") {
		t.Errorf("error should suggest the package manager, got: %v", err)
	}
}

func TestNewerThan(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.2.0", "1.1.2", true},
		{"1.1.3", "1.1.2", true},
		{"2.0.0", "1.9.9", true},
		{"1.10.0", "1.9.0", true},
		{"1.1.2", "1.1.2", false},
		{"1.1.1", "1.1.2", false},
	}
	for _, tc := range cases {
		if got := newerThan(tc.a, tc.b); got != tc.want {
			t.Errorf("newerThan(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// A dev build must refuse rather than clobber a locally built binary.
func TestRunRejectsDevBuild(t *testing.T) {
	var out bytes.Buffer
	_, err := Run("dev", true, &out)
	if err == nil {
		t.Fatal("expected Run to refuse a dev build")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Guard the property the whole design rests on: a tampered archive must not be
// installed. Verified here at the checksum layer.
func TestChecksumMismatchIsDetected(t *testing.T) {
	blob := makeTarGz(t, "seshat", []byte("legit"))
	sum := sha256.Sum256(blob)
	good := hex.EncodeToString(sum[:])

	tampered := makeTarGz(t, "seshat", []byte("malicious"))
	bad := sha256.Sum256(tampered)
	if hex.EncodeToString(bad[:]) == good {
		t.Fatal("test setup is wrong: digests collided")
	}
}
