package modrinth

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/msg"
)

// Each instance keeps rabbit-content.json with what is known about its
// content files, keyed by "folder/file.jar" (without ".disabled").
const manifestName = "rabbit-content.json"

const disabledSuffix = ".disabled"

type Meta struct {
	ProjectID string `json:"projectId,omitempty"`
	VersionID string `json:"versionId,omitempty"`
	Title     string `json:"title,omitempty"`
	IconURL   string `json:"iconUrl,omitempty"`
	Version   string `json:"version,omitempty"`
	Size      int64  `json:"size"` // detects files replaced by hand
}

type manifest map[string]Meta

var mu sync.Mutex // guards manifest files

func loadManifest(dir string) manifest {
	m := manifest{}
	if data, err := os.ReadFile(filepath.Join(dir, manifestName)); err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

func (m manifest) save(dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manifestName), data, 0o644)
}

// projectIDs returns the projects installed in an instance.
func (m manifest) projectIDs() map[string]bool {
	ids := map[string]bool{}
	for _, meta := range m {
		if meta.ProjectID != "" {
			ids[meta.ProjectID] = true
		}
	}
	return ids
}

// InstalledProjects returns ids of Modrinth projects installed in dir.
func InstalledProjects(dir string) []string {
	mu.Lock()
	m := loadManifest(dir)
	mu.Unlock()
	var out []string
	for id := range m.projectIDs() {
		out = append(out, id)
	}
	return out
}

func fileHash(f *File) string {
	if h := f.Hashes["sha512"]; h != "" {
		return h
	}
	return f.Hashes["sha1"]
}

// Install installs the newest compatible version of a project (and its
// required dependencies) into the instance at dir. It returns the titles of
// everything installed.
func Install(ctx context.Context, dir, projectID, projectType, game, loader string) ([]string, error) {
	in := &installer{dir: dir, game: game, loader: loader, seen: map[string]bool{}}
	mu.Lock()
	in.m = loadManifest(dir)
	mu.Unlock()
	err := in.project(ctx, projectID, projectType, "")
	mu.Lock()
	// Merge with entries written meanwhile (e.g. a listing).
	cur := loadManifest(dir)
	for k, v := range in.m {
		cur[k] = v
	}
	for _, k := range in.removed {
		delete(cur, k)
	}
	cur.save(dir)
	mu.Unlock()
	return in.installed, err
}

type installer struct {
	dir, game, loader string
	m                 manifest
	seen              map[string]bool
	installed         []string
	removed           []string
}

func (in *installer) project(ctx context.Context, id, projectType, versionID string) error {
	if in.seen[id] {
		return nil
	}
	in.seen[id] = true

	project, err := GetProject(ctx, id)
	if err != nil {
		return err
	}
	if projectType == "" {
		projectType = project.Type
	}
	var v *Version
	if versionID != "" {
		v, err = GetVersion(ctx, versionID)
	} else {
		v, err = in.pickVersion(ctx, id, projectType)
	}
	if err != nil {
		return err
	}
	folder := Folder(projectType)
	f := v.PrimaryFile()
	if folder == "" || f == nil {
		return msg.E("contentUnsupported")
	}

	// Replace an older version of the same project.
	for key, meta := range in.m {
		if meta.ProjectID == id && key != folder+"/"+f.Filename {
			p := filepath.Join(in.dir, filepath.FromSlash(key))
			os.Remove(p)
			os.Remove(p + disabledSuffix)
			delete(in.m, key)
			in.removed = append(in.removed, key)
		}
	}
	dest := filepath.Join(in.dir, folder, f.Filename)
	if err := download.One(ctx, download.File{URL: f.URL, Path: dest, Hash: fileHash(f), Size: f.Size}); err != nil {
		return err
	}
	in.m[folder+"/"+f.Filename] = Meta{
		ProjectID: id, VersionID: v.ID, Title: project.Title, IconURL: project.IconURL,
		Version: v.VersionNumber, Size: f.Size,
	}
	in.installed = append(in.installed, project.Title)

	installed := in.m.projectIDs()
	for _, d := range v.Dependencies {
		if d.Type != "required" || d.ProjectID == "" || installed[d.ProjectID] {
			continue
		}
		if err := in.project(ctx, d.ProjectID, "", d.VersionID); err != nil {
			return err
		}
	}
	return nil
}

func (in *installer) pickVersion(ctx context.Context, id, projectType string) (*Version, error) {
	var loaders []string
	if projectType == TypeMod || projectType == "" {
		loaders = loaderNames(in.loader)
	} else if projectType == TypeShader && in.loader == "optifine" {
		loaders = []string{"optifine"} // OptiFine reads its own shader format only
	}
	list, err := ProjectVersions(ctx, id, in.game, loaders)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 && (projectType == TypeResourcePack || projectType == TypeShader) {
		// Packs often work on newer game versions than they declare.
		list, err = ProjectVersions(ctx, id, "", loaders)
		if err != nil {
			return nil, err
		}
	}
	if len(list) == 0 {
		return nil, msg.E("noCompatibleVersion")
	}
	// Newest release, or the newest beta/alpha if there is no release.
	for i := range list {
		if list[i].VersionType == "release" {
			return &list[i], nil
		}
	}
	return &list[0], nil
}

// Item is a content file shown in the "installed" list.
type Item struct {
	Path      string `json:"path"` // relative, with .disabled if disabled
	File      string `json:"file"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	ProjectID string `json:"projectId"`
	Title     string `json:"title"`
	IconURL   string `json:"iconUrl"`
	Version   string `json:"version"`
}

var contentFolders = map[string]string{"mods": TypeMod, "resourcepacks": TypeResourcePack, "shaderpacks": TypeShader}

func sha1File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// List returns the content of an instance. Files added by hand are looked
// up on Modrinth by hash once and remembered.
func List(ctx context.Context, dir string) ([]Item, error) {
	mu.Lock()
	m := loadManifest(dir)
	mu.Unlock()

	var items []Item
	unknown := map[string]string{} // sha1 -> key
	sizes := map[string]int64{}
	for folder, typ := range contentFolders {
		entries, _ := os.ReadDir(filepath.Join(dir, folder))
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() && typ != TypeResourcePack && typ != TypeShader {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			base := strings.TrimSuffix(name, disabledSuffix)
			key := folder + "/" + base
			it := Item{Path: folder + "/" + name, File: base, Type: typ, Enabled: base == name}
			if meta, ok := m[key]; ok && (meta.Size == info.Size() || e.IsDir()) {
				it.ProjectID, it.Title, it.IconURL, it.Version = meta.ProjectID, meta.Title, meta.IconURL, meta.Version
			} else if !e.IsDir() {
				if sum, err := sha1File(filepath.Join(dir, folder, name)); err == nil {
					unknown[sum] = key
					sizes[key] = info.Size()
				}
			}
			items = append(items, it)
		}
	}

	if len(unknown) > 0 {
		var hashes []string
		for h := range unknown {
			hashes = append(hashes, h)
		}
		if found, err := VersionsByHash(ctx, hashes); err == nil {
			var ids []string
			for _, v := range found {
				ids = append(ids, v.ProjectID)
			}
			projects := map[string]Project{}
			if list, err := GetProjects(ctx, ids); err == nil {
				for _, p := range list {
					projects[p.ID] = p
				}
			}
			mu.Lock()
			cur := loadManifest(dir)
			for h, key := range unknown {
				meta := Meta{Size: sizes[key]} // remembered even when not on Modrinth
				if v, ok := found[h]; ok {
					p := projects[v.ProjectID]
					meta.ProjectID, meta.VersionID, meta.Version = v.ProjectID, v.ID, v.VersionNumber
					meta.Title, meta.IconURL = p.Title, p.IconURL
				}
				cur[key] = meta
			}
			cur.save(dir)
			mu.Unlock()
			for i, it := range items {
				meta := cur[strings.TrimSuffix(it.Path, disabledSuffix)]
				if it.ProjectID == "" && meta.ProjectID != "" {
					items[i].ProjectID, items[i].Title, items[i].IconURL, items[i].Version = meta.ProjectID, meta.Title, meta.IconURL, meta.Version
				}
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return strings.ToLower(a.title()) < strings.ToLower(b.title())
	})
	return items, nil
}

func (it Item) title() string {
	if it.Title != "" {
		return it.Title
	}
	return it.File
}

// safePath validates a relative content path and returns it as absolute.
func safePath(dir, rel string) (string, error) {
	rel = filepath.ToSlash(rel)
	folder, name, ok := strings.Cut(rel, "/")
	if _, known := contentFolders[folder]; !ok || !known || name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", errors.New("invalid path")
	}
	return filepath.Join(dir, folder, name), nil
}

// Remove deletes a content file.
func Remove(dir, rel string) error {
	p, err := safePath(dir, rel)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(p); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadManifest(dir)
	delete(m, strings.TrimSuffix(filepath.ToSlash(rel), disabledSuffix))
	return m.save(dir)
}

// SetEnabled renames a file to or from "*.disabled" and returns the new path.
func SetEnabled(dir, rel string, enabled bool) (string, error) {
	p, err := safePath(dir, rel)
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(p, disabledSuffix)
	target := base
	if !enabled {
		target = base + disabledSuffix
	}
	if target == p {
		return rel, nil
	}
	if err := os.Rename(p, target); err != nil {
		return "", err
	}
	relDir := filepath.ToSlash(filepath.Dir(filepath.ToSlash(rel)))
	return relDir + "/" + filepath.Base(target), nil
}
