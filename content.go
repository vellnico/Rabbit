package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/modrinth"
	"github.com/vellnico/rabbit/internal/msg"
	"github.com/vellnico/rabbit/internal/paths"
	"github.com/vellnico/rabbit/internal/store"
)

// ---- Modrinth content ----

func (a *App) SearchContent(p modrinth.SearchParams) (*modrinth.SearchResult, error) {
	res, err := modrinth.Search(a.ctx, p)
	if err != nil {
		return nil, msg.Wrap("modrinthUnavailable", err)
	}
	return res, nil
}

// InstallContent installs a project and its required dependencies into an
// instance. Returns the titles of what was installed.
func (a *App) InstallContent(instanceID, projectID, projectType string) ([]string, error) {
	in, err := store.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return modrinth.Install(a.ctx, paths.Instance(in.ID), projectID, projectType, in.GameVersion, in.Loader)
}

func (a *App) InstalledProjects(instanceID string) []string {
	return modrinth.InstalledProjects(paths.Instance(instanceID))
}

func (a *App) ListContent(instanceID string) ([]modrinth.Item, error) {
	if _, err := store.GetInstance(instanceID); err != nil {
		return nil, err
	}
	return modrinth.List(a.ctx, paths.Instance(instanceID))
}

func (a *App) RemoveContent(instanceID, path string) error {
	return modrinth.Remove(paths.Instance(instanceID), path)
}

func (a *App) SetContentEnabled(instanceID, path string, enabled bool) (string, error) {
	return modrinth.SetEnabled(paths.Instance(instanceID), path, enabled)
}

// ---- modpacks ----

type importEvent struct {
	Stage string `json:"stage"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

func (a *App) reportImport(stage string, done, total int) {
	runtime.EventsEmit(a.ctx, "import", importEvent{stage, done, total})
}

// importPack creates an instance from a .mrpack file.
func (a *App) importPack(ctx context.Context, file string) (store.Instance, error) {
	idx, err := modrinth.ReadIndex(file)
	if err != nil {
		return store.Instance{}, err
	}
	game, loader, loaderVersion := idx.GameAndLoader()
	in, err := store.CreateInstance(store.Instance{Name: idx.Name, GameVersion: game, Loader: loader, LoaderVersion: loaderVersion})
	if err != nil {
		return in, err
	}
	err = modrinth.Extract(ctx, file, paths.Instance(in.ID), idx, func(d, t int) { a.reportImport("files", d, t) })
	if err != nil {
		store.DeleteInstance(in.ID)
		return store.Instance{}, err
	}
	a.SelectInstance(in.ID)
	return in, nil
}

// ImportModpack asks for a .mrpack file and creates an instance from it.
// An empty instance id means the dialog was cancelled.
func (a *App) ImportModpack() (store.Instance, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{{DisplayName: "Modrinth modpack (*.mrpack)", Pattern: "*.mrpack"}},
	})
	if err != nil || file == "" {
		return store.Instance{}, err
	}
	a.reportImport("read", 0, 0)
	return a.importPack(a.ctx, file)
}

// InstallModpack downloads the newest version of a Modrinth modpack and
// creates an instance from it.
func (a *App) InstallModpack(projectID string) (store.Instance, error) {
	a.reportImport("download", 0, 0)
	f, err := modrinth.LatestModpackFile(a.ctx, projectID)
	if err != nil {
		return store.Instance{}, err
	}
	tmp, err := os.MkdirTemp("", "rabbit-mrpack-")
	if err != nil {
		return store.Instance{}, err
	}
	defer os.RemoveAll(tmp)
	file := filepath.Join(tmp, "pack.mrpack")
	hash := f.Hashes["sha512"]
	if hash == "" {
		hash = f.Hashes["sha1"]
	}
	if err := download.One(a.ctx, download.File{URL: f.URL, Path: file, Hash: hash, Size: f.Size}); err != nil {
		return store.Instance{}, err
	}
	return a.importPack(a.ctx, file)
}

// ExportModpack asks where to save and writes the instance as .mrpack.
// Returns the saved path, or "" if cancelled.
func (a *App) ExportModpack(instanceID string) (string, error) {
	in, err := store.GetInstance(instanceID)
	if err != nil {
		return "", err
	}
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, in.Name)
	out, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: name + ".mrpack",
		Filters:         []runtime.FileFilter{{DisplayName: "Modrinth modpack (*.mrpack)", Pattern: "*.mrpack"}},
	})
	if err != nil || out == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(out), ".mrpack") {
		out += ".mrpack"
	}
	err = modrinth.Export(a.ctx, paths.Instance(in.ID), modrinth.ExportOptions{
		Name: in.Name, Version: "1.0.0", Game: in.GameVersion, Loader: in.Loader, LoaderVersion: in.LoaderVersion,
	}, out, func(d, t int) { a.reportImport("export", d, t) })
	if err != nil {
		return "", err
	}
	return out, nil
}

// ShowInFolder opens Explorer with the file selected.
func (a *App) ShowInFolder(path string) error {
	return startExplorer("/select,", path)
}
