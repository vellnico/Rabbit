package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	productName = "Rabbit"
	productExe  = "Rabbit.exe"
	uninstExe   = "uninstall.exe"
	publisher   = "vellnico"
	uninstKey   = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Rabbit`
)

func appData() string {
	dir, _ := os.UserConfigDir()
	return dir
}

// dataDir is where the launcher keeps instances, worlds and accounts.
func dataDir() string { return filepath.Join(appData(), productName) }

func defaultDir() string {
	local, _ := os.UserCacheDir() // %LOCALAPPDATA%
	return filepath.Join(local, "Programs", productName)
}

// knownFolder returns a shell folder, or fallback if it cannot be read: an
// empty path would put the shortcut into the current folder.
func knownFolder(id *windows.KNOWNFOLDERID, fallback string) string {
	if dir, err := windows.KnownFolderPath(id, windows.KF_FLAG_DEFAULT); err == nil && dir != "" {
		return dir
	}
	return fallback
}

func startMenuShortcut() string {
	dir := knownFolder(windows.FOLDERID_Programs, filepath.Join(appData(), `Microsoft\Windows\Start Menu\Programs`))
	return filepath.Join(dir, productName+".lnk")
}

func desktopShortcut() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(knownFolder(windows.FOLDERID_Desktop, filepath.Join(home, "Desktop")), productName+".lnk")
}

// installed returns the folder and version of the current installation.
func installed() (dir, ver string, ok bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, uninstKey, registry.QUERY_VALUE)
	if err != nil {
		return "", "", false
	}
	defer k.Close()
	dir, _, err = k.GetStringValue("InstallLocation")
	ver, _, _ = k.GetStringValue("DisplayVersion")
	return dir, ver, err == nil && dir != ""
}

// inUse reports whether a file exists and is locked, e.g. a running exe.
func inUse(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return !os.IsNotExist(err)
	}
	f.Close()
	return false
}

// gzipSize reads the uncompressed size from the gzip trailer.
func gzipSize(b []byte) int64 {
	if len(b) < 4 {
		return 0
	}
	return int64(binary.LittleEndian.Uint32(b[len(b)-4:]))
}

type counter struct {
	done, total int64
	report      func(done, total int64)
}

func (c *counter) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	if c.report != nil {
		c.report(c.done, c.total)
	}
	return len(p), nil
}

// unpack writes a gzipped payload next to dst and then replaces dst.
func unpack(gz []byte, dst string, c *counter) error {
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(io.MultiWriter(f, c), zr)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func install(dir string, desktop bool, report func(done, total int64)) error {
	if !hasPayload {
		return errors.New("no payload")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	exe := filepath.Join(dir, productExe)
	oldDir, _, wasInstalled := installed()
	if inUse(exe) || (wasInstalled && inUse(filepath.Join(oldDir, productExe))) {
		return errors.New("@running")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	c := &counter{total: gzipSize(rabbitGz) + gzipSize(uninstallGz), report: report}
	if err := unpack(rabbitGz, exe, c); err != nil {
		return err
	}
	if err := unpack(uninstallGz, filepath.Join(dir, uninstExe), c); err != nil {
		return err
	}

	// Moved to another folder: remove the old copy.
	if wasInstalled && !strings.EqualFold(filepath.Clean(oldDir), dir) {
		removeFiles(oldDir)
	}

	if err := createShortcut(startMenuShortcut(), exe); err != nil {
		return errors.New("@shortcut|" + err.Error())
	}
	if desktop {
		if err := createShortcut(desktopShortcut(), exe); err != nil {
			return errors.New("@shortcut|" + err.Error())
		}
	} else {
		removeShortcut(desktopShortcut())
	}
	err = register(dir, c.total)
	refreshShell()
	return err
}

func register(dir string, size int64) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, uninstKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	uninst := `"` + filepath.Join(dir, uninstExe) + `"`
	for name, v := range map[string]string{
		"DisplayName":          productName,
		"DisplayVersion":       version,
		"Publisher":            publisher,
		"DisplayIcon":          filepath.Join(dir, productExe),
		"InstallLocation":      dir,
		"UninstallString":      uninst,
		"QuietUninstallString": uninst + " /S",
	} {
		if err := k.SetStringValue(name, v); err != nil {
			return err
		}
	}
	k.SetDWordValue("EstimatedSize", uint32(size/1024))
	k.SetDWordValue("NoModify", 1)
	k.SetDWordValue("NoRepair", 1)
	return nil
}

// removeFiles deletes only the files setup wrote: the folder may hold
// other things if the user picked it.
func removeFiles(dir string) {
	for _, name := range []string{productExe, uninstExe} {
		p := filepath.Join(dir, name)
		// The uninstaller that started us may still be exiting.
		for i := 0; i < 15; i++ {
			if err := os.Remove(p); err == nil || os.IsNotExist(err) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	os.Remove(dir) // only if empty
}

func uninstall(dir string, removeData bool) error {
	if dir == "" {
		return errors.New("@notInstalled")
	}
	if inUse(filepath.Join(dir, productExe)) {
		return errors.New("@running")
	}
	removeShortcut(startMenuShortcut())
	removeShortcut(desktopShortcut())
	removeFiles(dir)
	os.RemoveAll(filepath.Join(appData(), productExe)) // the launcher's WebView2 data
	err := registry.DeleteKey(registry.CURRENT_USER, uninstKey)
	refreshShell()
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	if removeData {
		if err := os.RemoveAll(dataDir()); err != nil {
			return errors.New("@dataInUse|" + err.Error())
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// startHidden runs a command line without a console window and does not wait.
func startHidden(exe, args string) {
	cmd := exec.Command(exe)
	// Only CREATE_NO_WINDOW: with DETACHED_PROCESS cmd has no console, so a
	// console child (ping) would open a new, visible one.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       exe + " " + args,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		HideWindow:    true,
	}
	cmd.Start()
}
