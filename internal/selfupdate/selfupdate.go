// Package selfupdate replaces the running seshat binary with the latest
// published release. It only ever installs an archive whose SHA-256 matches the
// checksums.txt published alongside it.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releaseURL  = "https://api.github.com/repos/tardanoir/seshat/releases/latest"
	binaryName  = "seshat"
	maxDownload = 100 << 20 // generous ceiling so a bad URL can't stream forever
)

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

// Result describes what an update run did.
type Result struct {
	Current     string
	Latest      string
	Updated     bool
	InstalledTo string
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Run upgrades the running binary to the latest release. When checkOnly is set
// it reports what would happen without touching anything on disk. Progress is
// written to w.
func Run(current string, checkOnly bool, w io.Writer) (Result, error) {
	res := Result{Current: strings.TrimPrefix(current, "v")}

	rel, err := fetchLatest()
	if err != nil {
		return res, fmt.Errorf("checking for updates: %w", err)
	}
	res.Latest = strings.TrimPrefix(rel.TagName, "v")

	if current == "dev" || current == "" {
		return res, fmt.Errorf("this is a development build (version %q); install a release first", current)
	}
	if !newerThan(res.Latest, res.Current) {
		fmt.Fprintf(w, "seshat %s is already up to date.\n", res.Current)
		return res, nil
	}

	fmt.Fprintf(w, "Update available: %s -> %s\n", res.Current, res.Latest)
	if checkOnly {
		return res, nil
	}

	// Resolve the destination first: failing on a read-only install location is
	// much better after a long download than in the middle of swapping files.
	dest, err := executablePath()
	if err != nil {
		return res, err
	}
	if err := checkWritable(dest); err != nil {
		return res, err
	}
	res.InstalledTo = dest

	archiveName := assetName(res.Latest)
	archive := findAsset(rel.Assets, archiveName)
	if archive == nil {
		return res, fmt.Errorf("release %s has no asset for %s/%s (looked for %s)",
			rel.TagName, runtime.GOOS, runtime.GOARCH, archiveName)
	}
	sums := findAsset(rel.Assets, "checksums.txt")
	if sums == nil {
		return res, fmt.Errorf("release %s publishes no checksums.txt; refusing to install unverified binary", rel.TagName)
	}

	fmt.Fprintf(w, "Downloading %s...\n", archive.Name)
	blob, err := download(archive.URL)
	if err != nil {
		return res, fmt.Errorf("downloading %s: %w", archive.Name, err)
	}

	sumBlob, err := download(sums.URL)
	if err != nil {
		return res, fmt.Errorf("downloading checksums.txt: %w", err)
	}
	want, err := checksumFor(sumBlob, archive.Name)
	if err != nil {
		return res, err
	}
	got := sha256.Sum256(blob)
	if hex.EncodeToString(got[:]) != want {
		return res, fmt.Errorf("checksum mismatch for %s: refusing to install", archive.Name)
	}
	fmt.Fprintln(w, "Checksum verified.")

	bin, err := extractBinary(archive.Name, blob)
	if err != nil {
		return res, err
	}
	if err := replaceExecutable(dest, bin); err != nil {
		return res, err
	}

	res.Updated = true
	fmt.Fprintf(w, "Updated seshat to %s (%s)\n", res.Latest, dest)
	return res, nil
}

func fetchLatest() (*release, error) {
	resp, err := httpClient.Get(releaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api: %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("github api returned no tag_name")
	}
	return &rel, nil
}

// assetName mirrors the archive naming in .goreleaser.yaml.
func assetName(version string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", binaryName, version, runtime.GOOS, runtime.GOARCH, ext)
}

func findAsset(assets []asset, name string) *asset {
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i]
		}
	}
	return nil
}

func download(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxDownload))
}

// checksumFor pulls one file's expected digest out of a checksums.txt body.
func checksumFor(sums []byte, name string) (string, error) {
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		// The name column may carry a leading "*" for binary mode.
		if strings.TrimPrefix(fields[1], "*") == name {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksums.txt has no entry for %s", name)
}

// extractBinary pulls the seshat executable out of a release archive.
func extractBinary(name string, blob []byte) ([]byte, error) {
	if strings.HasSuffix(name, ".zip") {
		return extractFromZip(blob)
	}
	return extractFromTarGz(blob)
}

func extractFromTarGz(blob []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != binaryName {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, maxDownload))
	}
	return nil, fmt.Errorf("archive contains no %s binary", binaryName)
}

func extractFromZip(blob []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(blob), int64(len(blob)))
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base != binaryName && base != binaryName+".exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, maxDownload))
	}
	return nil, fmt.Errorf("archive contains no %s binary", binaryName)
}

// executablePath resolves the running binary, following symlinks so we replace
// the real file rather than a link pointing at it.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the running binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// checkWritable rejects install locations we cannot replace in place — most
// often a package-manager-owned prefix, where the package manager should be
// doing the upgrade instead.
func checkWritable(dest string) error {
	dir := filepath.Dir(dest)
	probe, err := os.CreateTemp(dir, ".seshat-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w\n"+
			"seshat looks installed system-wide or by a package manager; "+
			"upgrade it the same way you installed it (brew upgrade, pacman -Syu, apt upgrade, scoop update seshat)", dir, err)
	}
	probe.Close()
	os.Remove(probe.Name())
	return nil
}

// replaceExecutable swaps in the new binary. The temp file is created in the
// destination directory so the final rename is atomic on the same filesystem,
// and the old binary is moved aside first so the swap also works on platforms
// that refuse to overwrite a running image.
func replaceExecutable(dest string, bin []byte) error {
	mode := os.FileMode(0o755)
	if fi, err := os.Stat(dest); err == nil {
		mode = fi.Mode().Perm()
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".seshat-new-*")
	if err != nil {
		return fmt.Errorf("staging the new binary: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return fmt.Errorf("writing the new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing the new binary: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	backup := dest + ".old"
	os.Remove(backup)
	if err := os.Rename(dest, backup); err != nil {
		return fmt.Errorf("moving the current binary aside: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		// Put the original back rather than leaving the user with no binary.
		if rerr := os.Rename(backup, dest); rerr != nil {
			return fmt.Errorf("installing the new binary failed (%w) and restoring the old one failed too (%v); "+
				"the previous binary is at %s", err, rerr, backup)
		}
		return fmt.Errorf("installing the new binary: %w", err)
	}
	os.Remove(backup)
	return nil
}

// newerThan does the same major.minor.patch comparison as the update notifier.
func newerThan(a, b string) bool {
	pa, pb := parseSemver(a), parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseSemver(s string) [3]int {
	var v [3]int
	for i, p := range strings.SplitN(s, ".", 3) {
		if i >= 3 {
			break
		}
		fmt.Sscanf(p, "%d", &v[i])
	}
	return v
}
