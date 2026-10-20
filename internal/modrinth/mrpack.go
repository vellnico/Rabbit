package modrinth

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/msg"
)

// .mrpack format: https://support.modrinth.com/en/articles/8802351-modrinth-modpack-format-mrpack

const indexName = "modrinth.index.json"

type Index struct {
	FormatVersion int               `json:"formatVersion"`
	Game          string            `json:"game"`
	VersionID     string            `json:"versionId"`
	Name          string            `json:"name"`
	Summary       string            `json:"summary,omitempty"`
	Files         []IndexFile       `json:"files"`
	Dependencies  map[string]string `json:"dependencies"`
}

type IndexFile struct {
	Path      string            `json:"path"`
	Hashes    map[string]string `json:"hashes"`
	Env       *Env              `json:"env,omitempty"`
	Downloads []string          `json:"downloads"`
	FileSize  int64             `json:"fileSize"`
}

type Env struct {
	Client string `json:"client"`
	Server string `json:"server"`
}

// Loader dependency keys of the format and the matching instance loaders.
var loaderKeys = map[string]string{
	"fabric-loader": "fabric",
	"quilt-loader":  "quilt",
	"forge":         "forge",
	"neoforge":      "neoforge",
}

// GameAndLoader returns the Minecraft version, instance loader and loader
// version the pack needs.
func (idx *Index) GameAndLoader() (game, loader, loaderVersion string) {
	game = idx.Dependencies["minecraft"]
	loader = "vanilla"
	for key, l := range loaderKeys {
		if v := idx.Dependencies[key]; v != "" {
			loader, loaderVersion = l, v
		}
	}
	return
}

// ReadIndex opens a .mrpack and parses its index.
func ReadIndex(path string) (*Index, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, msg.Wrap("badModpack", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != indexName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		var idx Index
		if err := json.NewDecoder(rc).Decode(&idx); err != nil {
			return nil, msg.Wrap("badModpack", err)
		}
		if idx.Game != "minecraft" || idx.Dependencies["minecraft"] == "" {
			return nil, msg.E("badModpack")
		}
		return &idx, nil
	}
	return nil, msg.E("badModpack")
}

// within joins rel to dir and refuses paths that escape dir.
func within(dir, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", fmt.Errorf("unsafe path %q", rel)
	}
	p := filepath.Join(dir, rel)
	if r, err := filepath.Rel(dir, p); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path %q", rel)
	}
	return p, nil
}

// Hosts the format allows downloads from.
var allowedHosts = []string{"cdn.modrinth.com", "github.com", "raw.githubusercontent.com", "gitlab.com"}

func allowedURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	for _, h := range allowedHosts {
		if u.Hostname() == h {
			return true
		}
	}
	return false
}

// Extract installs a pack into dir: downloads its files and copies the
// overrides. report gets file counts.
func Extract(ctx context.Context, mrpack, dir string, idx *Index, report func(done, total int)) error {
	var files []download.File
	for _, f := range idx.Files {
		if f.Env != nil && f.Env.Client == "unsupported" {
			continue
		}
		p, err := within(dir, f.Path)
		if err != nil {
			return err
		}
		var src string
		for _, d := range f.Downloads {
			if allowedURL(d) {
				src = d
				break
			}
		}
		if src == "" {
			return fmt.Errorf("%s: no allowed download URL", f.Path)
		}
		hash := f.Hashes["sha512"]
		if hash == "" {
			hash = f.Hashes["sha1"]
		}
		files = append(files, download.File{URL: src, Path: p, Hash: hash, Size: f.FileSize})
	}
	if err := download.All(ctx, files, 8, report); err != nil {
		return err
	}

	zr, err := zip.OpenReader(mrpack)
	if err != nil {
		return err
	}
	defer zr.Close()
	// client-overrides are applied after overrides so they win.
	for _, prefix := range []string{"overrides/", "client-overrides/"} {
		for _, f := range zr.File {
			rel, ok := strings.CutPrefix(f.Name, prefix)
			if !ok || rel == "" || f.FileInfo().IsDir() {
				continue
			}
			p, err := within(dir, rel)
			if err != nil {
				return err
			}
			if err := extractFile(f, p); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractFile(f *zip.File, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, rc)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

// LatestModpackFile returns the .mrpack of the newest version of a modpack.
func LatestModpackFile(ctx context.Context, projectID string) (*File, error) {
	list, err := ProjectVersions(ctx, projectID, "", nil)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		for i := range v.Files {
			if strings.HasSuffix(v.Files[i].Filename, ".mrpack") {
				return &v.Files[i], nil
			}
		}
	}
	return nil, msg.E("noCompatibleVersion")
}

type ExportOptions struct {
	Name          string
	Version       string
	Game          string
	Loader        string
	LoaderVersion string
}

// Folders copied into overrides as is (configs) when exporting.
var exportOverrides = []string{"config", "defaultconfigs", "kubejs"}

// Export writes the instance at dir as a .mrpack to out. Files found on
// Modrinth are referenced by URL, everything else goes into overrides.
func Export(ctx context.Context, dir string, o ExportOptions, out string, report func(done, total int)) error {
	type local struct {
		rel          string
		sha1, sha512 string
		size         int64
	}
	var locals []local
	for folder := range contentFolders {
		entries, _ := os.ReadDir(filepath.Join(dir, folder))
		for _, e := range entries {
			if e.IsDir() || strings.HasSuffix(e.Name(), disabledSuffix) {
				continue
			}
			p := filepath.Join(dir, folder, e.Name())
			h1, h512, size, err := hashes(p)
			if err != nil {
				return err
			}
			locals = append(locals, local{folder + "/" + e.Name(), h1, h512, size})
		}
	}

	var sums []string
	for _, l := range locals {
		sums = append(sums, l.sha1)
	}
	found, err := VersionsByHash(ctx, sums)
	if err != nil {
		return msg.Wrap("modrinthUnavailable", err)
	}

	idx := Index{
		FormatVersion: 1, Game: "minecraft", VersionID: o.Version, Name: o.Name,
		Dependencies: map[string]string{"minecraft": o.Game},
	}
	for key, l := range loaderKeys {
		if l == o.Loader {
			idx.Dependencies[key] = o.LoaderVersion
		}
	}
	var overrides []string
	for _, l := range locals {
		v, ok := found[l.sha1]
		var src string
		if ok {
			for _, f := range v.Files {
				if f.Hashes["sha1"] == l.sha1 {
					src = f.URL
				}
			}
		}
		if src == "" {
			overrides = append(overrides, l.rel)
			continue
		}
		idx.Files = append(idx.Files, IndexFile{
			Path:      l.rel,
			Hashes:    map[string]string{"sha1": l.sha1, "sha512": l.sha512},
			Downloads: []string{src},
			FileSize:  l.size,
		})
	}
	for _, d := range exportOverrides {
		filepath.WalkDir(filepath.Join(dir, d), func(p string, e os.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				rel, _ := filepath.Rel(dir, p)
				overrides = append(overrides, filepath.ToSlash(rel))
			}
			return nil
		})
	}

	tmp := out + ".part"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(fh)
	w, err := zw.Create(indexName)
	if err == nil {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		err = enc.Encode(&idx)
	}
	for i, rel := range overrides {
		if err != nil {
			break
		}
		err = addFile(zw, filepath.Join(dir, filepath.FromSlash(rel)), "overrides/"+rel)
		if report != nil {
			report(i+1, len(overrides))
		}
	}
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	os.Remove(out)
	return os.Rename(tmp, out)
}

func hashes(path string) (h1, h512 string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s1, s512 := sha1.New(), sha512.New()
	size, err = io.Copy(io.MultiWriter(s1, s512), f)
	return hex.EncodeToString(s1.Sum(nil)), hex.EncodeToString(s512.Sum(nil)), size, err
}

func addFile(zw *zip.Writer, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer in.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}
