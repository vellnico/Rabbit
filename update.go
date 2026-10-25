package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/launch"
	"github.com/vellnico/rabbit/internal/msg"
)

// Updates come from the GitHub releases of the repository. An installed
// launcher runs the new Rabbit-Setup.exe silently (it waits for the launcher
// to exit and starts it again); a portable one replaces its own exe.

const (
	releasesAPI = "https://api.github.com/repos/vellnico/Rabbit/releases/latest"
	uninstKey   = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Rabbit` // written by the installer
)

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"` // "sha256:..."
}

type release struct {
	Tag    string         `json:"tag_name"`
	Body   string         `json:"body"`
	URL    string         `json:"html_url"`
	Draft  bool           `json:"draft"`
	Pre    bool           `json:"prerelease"`
	Assets []releaseAsset `json:"assets"`
}

// UpdateInfo describes a newer release; nil means the launcher is up to date.
type UpdateInfo struct {
	Version string `json:"version"`
	Current string `json:"current"`
	Notes   string `json:"notes"` // markdown
	URL     string `json:"url"`   // release page
}

// newerVersion reports whether version a ("0.3.1", "v0.4") is newer than b.
func newerVersion(a, b string) bool {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// CheckUpdate looks for a newer release on GitHub. Development builds never
// update.
func (a *App) CheckUpdate() (*UpdateInfo, error) {
	if launch.LauncherVersion == "dev" {
		return nil, nil
	}
	var r release
	if err := download.JSON(a.ctx, releasesAPI, &r); err != nil {
		return nil, msg.Wrap("updateUnavailable", err)
	}
	if r.Draft || r.Pre || !newerVersion(r.Tag, launch.LauncherVersion) {
		return nil, nil
	}
	a.mu.Lock()
	a.release = &r
	a.mu.Unlock()
	return &UpdateInfo{Version: strings.TrimPrefix(r.Tag, "v"), Current: launch.LauncherVersion, Notes: r.Body, URL: r.URL}, nil
}

// installDir returns the folder of the installed launcher if this exe is it.
func installDir(exe string) (string, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, uninstKey, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer k.Close()
	dir, _, err := k.GetStringValue("InstallLocation")
	if err != nil || !strings.EqualFold(filepath.Clean(dir), filepath.Dir(exe)) {
		return "", false
	}
	return dir, true
}

// InstallUpdate downloads the release found by CheckUpdate, starts it and
// quits the launcher. Progress is sent as "update" events.
func (a *App) InstallUpdate() error {
	a.mu.Lock()
	r := a.release
	busy := len(a.running) > 0
	a.mu.Unlock()
	if r == nil {
		return errors.New("no update")
	}
	if busy {
		return msg.E("closeGameFirst")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	dir, installed := installDir(exe)
	name := "Rabbit.exe"
	if installed {
		name = "Rabbit-Setup.exe"
	}
	var asset *releaseAsset
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			asset = &r.Assets[i]
		}
	}
	if asset == nil {
		return msg.Wrap("updateFailed", fmt.Errorf("%s not found in %s", name, r.Tag))
	}

	if installed {
		tmp, err := os.MkdirTemp("", "rabbit-update-")
		if err != nil {
			return err
		}
		setup := filepath.Join(tmp, name)
		if err := a.fetchUpdate(*asset, setup); err != nil {
			os.RemoveAll(tmp)
			return msg.Wrap("updateFailed", err)
		}
		// Setup waits until this process has exited, then starts the new version.
		if err := exec.Command(setup, "/S", "/D="+dir, "--run").Start(); err != nil {
			return msg.Wrap("updateFailed", err)
		}
	} else {
		// A running exe cannot be overwritten but can be renamed.
		next, old := exe+".new", exe+".old"
		if err := a.fetchUpdate(*asset, next); err != nil {
			os.Remove(next)
			return msg.Wrap("updateFailed", err)
		}
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			os.Remove(next)
			return msg.Wrap("updateFailed", err)
		}
		if err := os.Rename(next, exe); err != nil {
			os.Rename(old, exe)
			return msg.Wrap("updateFailed", err)
		}
		cmd := exec.Command(exe)
		cmd.Dir = filepath.Dir(exe)
		if err := cmd.Start(); err != nil {
			return msg.Wrap("updateFailed", err)
		}
	}
	runtime.Quit(a.ctx)
	return nil
}

// removeOldExe deletes the exe a portable update left behind.
func removeOldExe() {
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
	}
}

type updateEvent struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

func (a *App) fetchUpdate(f releaseAsset, path string) error {
	resp, err := download.Get(a.ctx, f.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	pw := &progressWriter{total: f.Size, report: func(d, t int64) { runtime.EventsEmit(a.ctx, "update", updateEvent{d, t}) }}
	_, err = io.Copy(io.MultiWriter(out, h, pw), resp.Body)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if f.Size > 0 && pw.done != f.Size {
		return fmt.Errorf("%s: incomplete download", f.Name)
	}
	if sum, ok := strings.CutPrefix(f.Digest, "sha256:"); ok && !strings.EqualFold(sum, hex.EncodeToString(h.Sum(nil))) {
		return fmt.Errorf("%s: checksum mismatch", f.Name)
	}
	return nil
}

// progressWriter reports progress at most once per percent.
type progressWriter struct {
	done, total, last int64
	report            func(done, total int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.total <= 0 || (p.done-p.last)*100 >= p.total || p.done == p.total {
		p.last = p.done
		p.report(p.done, p.total)
	}
	return len(b), nil
}
