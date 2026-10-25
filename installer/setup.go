package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	modeInstall   = "install"
	modeUninstall = "uninstall"
)

// Setup is bound to the frontend.
// Errors are "@key" codes the frontend translates, or plain text.
type Setup struct {
	ctx  context.Context
	mode string
	dir  string // install folder: uninstall mode, or /D= of a silent install
	run  bool   // start the launcher after a silent install
}

func (s *Setup) startup(ctx context.Context) { s.ctx = ctx }

type Info struct {
	Mode             string `json:"mode"` // install | uninstall
	Version          string `json:"version"`
	InstalledVersion string `json:"installedVersion"` // "" when not installed
	Dir              string `json:"dir"`              // current or suggested folder
	Desktop          bool   `json:"desktop"`          // desktop shortcut exists (or is suggested)
	DataDir          string `json:"dataDir"`
	Accent           string `json:"accent"`   // from the launcher settings
	Language         string `json:"language"` // from the launcher settings, "" = system
}

func (s *Setup) Info() Info {
	in := Info{Mode: s.mode, Version: version, DataDir: dataDir(), Desktop: true}
	if dir, ver, ok := installed(); ok {
		in.Dir, in.InstalledVersion = dir, ver
		_, err := os.Stat(desktopShortcut())
		in.Desktop = err == nil
	} else {
		in.Dir = defaultDir()
	}
	if s.mode == modeUninstall {
		in.Dir = s.dir
	}
	var settings struct {
		Accent   string `json:"accent"`
		Language string `json:"language"`
	}
	if data, err := os.ReadFile(filepath.Join(dataDir(), "settings.json")); err == nil {
		json.Unmarshal(data, &settings)
	}
	in.Accent, in.Language = settings.Accent, settings.Language
	return in
}

// PickDir asks for an install folder. A "Rabbit" subfolder is added unless the
// picked folder already is one, so other files never share the folder.
func (s *Setup) PickDir(current string) string {
	dir, err := runtime.OpenDirectoryDialog(s.ctx, runtime.OpenDialogOptions{DefaultDirectory: filepath.Dir(current)})
	if err != nil || dir == "" {
		return current
	}
	if !strings.EqualFold(filepath.Base(dir), productName) {
		dir = filepath.Join(dir, productName)
	}
	return dir
}

type progressEvent struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

func (s *Setup) Install(dir string, desktop bool) error {
	return install(dir, desktop, func(done, total int64) {
		runtime.EventsEmit(s.ctx, "progress", progressEvent{done, total})
	})
}

func (s *Setup) Uninstall(removeData bool) error {
	return uninstall(s.dir, removeData)
}

// Launch starts the installed launcher and closes the setup.
func (s *Setup) Launch() {
	if dir, _, ok := installed(); ok {
		cmd := exec.Command(filepath.Join(dir, productExe))
		cmd.Dir = dir
		cmd.Start()
	}
	runtime.Quit(s.ctx)
}

// runSilent installs into the current or default folder, or uninstalls
// keeping the launcher data.
func (s *Setup) runSilent() error {
	if s.mode == modeUninstall {
		return uninstall(s.dir, false)
	}
	in := s.Info()
	if s.dir != "" {
		in.Dir = s.dir
	}
	// The updater starts setup and quits: give the launcher time to exit.
	exe := filepath.Join(in.Dir, productExe)
	for i := 0; i < 100 && inUse(exe); i++ {
		time.Sleep(200 * time.Millisecond)
	}
	_, err := os.Stat(desktopShortcut())
	if err := install(in.Dir, in.InstalledVersion != "" && err == nil, nil); err != nil {
		return err
	}
	if s.run {
		cmd := exec.Command(exe)
		cmd.Dir = in.Dir
		cmd.Start()
	}
	return nil
}
