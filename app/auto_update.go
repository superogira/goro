package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kivutar/goro/glog"
)

// Self-update: at boot the client fetches version.txt from the update
// server and, when it advertises a newer build, downloads the replacement
// binary next to the running one, swaps it in, and re-execs. Players never
// touch the SD card again.

// buildVersion is injected at build time (-ldflags
// "-X github.com/kivutar/goro/app.buildVersion=<hash>-<date>"); "dev" marks
// a locally built binary.
var buildVersion = "dev"

// BuildVersion reports the compiled-in build identifier.
func BuildVersion() string {
	return buildVersion
}

// localVersion prefers the compiled-in build and falls back to the marker
// written next to the binary by a previous update (covers manual binary
// swaps that kept the marker).
func localVersion(exeDir string) string {
	if buildVersion != "dev" && buildVersion != "" {
		return buildVersion
	}
	if data, err := os.ReadFile(filepath.Join(exeDir, "goro.version")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v
		}
	}
	return "dev"
}

const updateHTTPTimeout = 8 * time.Second
const updateDownloadTimeout = 10 * time.Minute

// remoteVersion fetches "<base>/version.txt". The first line is the build
// identifier; an optional second line is the binary's sha256.
func remoteVersion(baseURL string) (version, checksum string, err error) {
	client := &http.Client{Timeout: updateHTTPTimeout}
	resp, err := client.Get(strings.TrimSuffix(baseURL, "/") + "/version.txt")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("version check: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", "", fmt.Errorf("version check: empty response")
	}
	version = strings.TrimSpace(lines[0])
	if len(lines) > 1 {
		checksum = strings.TrimSpace(lines[1])
	}
	return version, checksum, nil
}

// RunSelfUpdate checks the server and, when the advertised build differs
// from the local one, downloads and installs it. It reports whether the
// binary was replaced (the caller should restart or exit).
func RunSelfUpdate(baseURL string) bool {
	exe, err := os.Executable()
	if err != nil {
		glog.Infof("update: executable path unavailable: %v", err)
		return false
	}
	exeDir := filepath.Dir(exe)
	local := localVersion(exeDir)
	remote, checksum, err := remoteVersion(baseURL)
	if err != nil {
		glog.Infof("update: check skipped: %v", err)
		return false
	}
	if remote == local {
		glog.Infof("update: up to date (%s)", local)
		return false
	}
	glog.Infof("update: %s -> %s, downloading", local, remote)
	download := filepath.Join(exeDir, filepath.Base(exe)+".download")
	if err := downloadBinary(strings.TrimSuffix(baseURL, "/")+"/goro", download, checksum); err != nil {
		glog.Warnf("update: download failed: %v", err)
		_ = os.Remove(download)
		return false
	}
	// Swap: rename over the running binary, then remember what we became.
	if err := os.Rename(download, exe); err != nil {
		// Some filesystems refuse to replace a running file; try the
		// remove-then-rename fallback before giving up.
		_ = os.Remove(exe)
		if err := os.Rename(download, exe); err != nil {
			glog.Warnf("update: swap failed: %v", err)
			_ = os.Remove(download)
			return false
		}
	}
	_ = os.Chmod(exe, 0o755)
	_ = os.WriteFile(filepath.Join(exeDir, "goro.version"), []byte(remote), 0o644)
	glog.Infof("update: installed %s — restart required", remote)
	return true
}

// ReexecSelf replaces the current process with the same binary and
// arguments (used after a successful update). Unavailable off Linux.
func ReexecSelf() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("re-exec unsupported on %s", runtime.GOOS)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}

// downloadBinary streams the update into dst, verifying the optional
// sha256 and refusing empty or implausibly small files.
func downloadBinary(url, dst, wantChecksum string) error {
	client := &http.Client{Timeout: updateDownloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if written < 1024*1024 {
		return fmt.Errorf("download too small (%d bytes)", written)
	}
	if wantChecksum != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != strings.ToLower(wantChecksum) {
			return fmt.Errorf("checksum mismatch: got %s", got)
		}
	}
	return nil
}
