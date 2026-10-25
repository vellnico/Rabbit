// Rabbit Setup: installs the launcher for the current user, and the same
// program without a payload is its uninstaller.
//
//	Rabbit-Setup.exe        (built with -tags setup) installs, /S silently, /D=folder
//	                        --run starts the launcher after a silent install
//	uninstall.exe           removes the launcher, /S silently
package main

import (
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:dist
var assets embed.FS

// version is set by build.ps1 from wails.json.
var version = "dev"

func main() {
	silent := false
	from := "" // install folder, when the uninstaller runs from a temp copy
	dir := ""  // install folder for a silent install
	run := false
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "/S", "/s", "--silent":
			silent = true
		case "--run": // used by the launcher updater
			run = true
		case "--from":
			if i+1 < len(os.Args) {
				from = os.Args[i+1]
				i++
			}
		default:
			if d, ok := strings.CutPrefix(os.Args[i], "/D="); ok { // NSIS-style folder for /S
				dir = d
			}
		}
	}

	s := &Setup{mode: modeInstall, dir: dir, run: run}
	if !hasPayload {
		s.mode = modeUninstall
		if from == "" {
			// A running exe cannot delete itself: continue from a copy in TEMP.
			relaunchFromTemp(silent)
			return
		}
		s.dir = from
	}

	if silent {
		code := 0
		if err := s.runSilent(); err != nil {
			code = 1
		}
		cleanupAfterExit()
		os.Exit(code)
	}

	err := wails.Run(&options.App{
		Title:            "Rabbit",
		Width:            640,
		Height:           460,
		DisableResize:    true,
		Frameless:        true,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 11, A: 255},
		OnStartup:        s.startup,
		Bind:             []interface{}{s},
		Windows: &windows.Options{
			Theme:               windows.Dark,
			WebviewUserDataPath: webviewDataDir(),
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
	cleanupAfterExit()
}

// webviewDataDir keeps WebView2 data out of %APPDATA%; it is removed on exit.
func webviewDataDir() string { return filepath.Join(os.TempDir(), "RabbitSetup.WebView2") }

func relaunchFromTemp(silent bool) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	tmp := filepath.Join(os.TempDir(), "Rabbit-uninstall-"+strconv.FormatInt(time.Now().UnixNano(), 36)+".exe")
	if err := copyFile(self, tmp); err != nil {
		return
	}
	args := []string{"--from", filepath.Dir(self)}
	if silent {
		args = append(args, "/S")
	}
	// Not waited for: the copy deletes this exe, which must not be running.
	exec.Command(tmp, args...).Start()
}

// cleanupAfterExit deletes the WebView2 data and, for the temp uninstaller,
// the exe itself once this process has exited.
func cleanupAfterExit() {
	script := `ping -n 3 127.0.0.1 >nul & rd /s /q "` + webviewDataDir() + `"`
	if !hasPayload {
		if self, err := os.Executable(); err == nil {
			script += ` & del /f /q "` + self + `"`
		}
	}
	startHidden("cmd.exe", "/c "+script)
}
