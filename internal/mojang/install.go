package mojang

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/paths"
)

const resourcesURL = "https://resources.download.minecraft.net/"

// Reporter receives install progress: a human readable stage and counters.
type Reporter func(stage string, done, total int)

// Install downloads everything needed to run the resolved version.
func Install(ctx context.Context, v *Version, report Reporter) error {
	var files []download.File

	if client, ok := v.Downloads["client"]; ok {
		files = append(files, download.File{URL: client.URL, Path: ClientJar(v.JarID), Hash: client.SHA1, Size: client.Size})
	}
	for _, l := range v.Libraries {
		if !Allowed(l.Rules) {
			continue
		}
		for _, a := range []*Artifact{l.MainArtifact(), l.NativeArtifact()} {
			if a == nil || a.URL == "" {
				continue
			}
			files = append(files, download.File{URL: a.URL, Path: filepath.Join(paths.Libraries(), filepath.FromSlash(a.Path)), Hash: a.SHA1, Size: a.Size})
		}
	}
	if lc, ok := v.Logging["client"]; ok && lc.File.URL != "" {
		files = append(files, download.File{URL: lc.File.URL, Path: LogConfig(lc.File.ID), Hash: lc.File.SHA1, Size: lc.File.Size})
	}
	err := download.All(ctx, files, 16, func(d, t int) { report("Библиотеки", d, t) })
	if err != nil {
		return err
	}
	if err := extractNatives(v); err != nil {
		return fmt.Errorf("нативы: %w", err)
	}
	return installAssets(ctx, v, report)
}

// LogConfig is where the log4j configuration for the client is stored.
func LogConfig(id string) string { return filepath.Join(paths.Assets(), "log_configs", id) }

// NativesDir is where legacy native libraries are extracted for a version.
func NativesDir(id string) string { return filepath.Join(paths.Version(id), "natives") }

func extractNatives(v *Version) error {
	dir := NativesDir(v.ID)
	for _, l := range v.Libraries {
		if !Allowed(l.Rules) {
			continue
		}
		a := l.NativeArtifact()
		if a == nil {
			continue
		}
		var exclude []string
		if l.Extract != nil {
			exclude = l.Extract.Exclude
		}
		if err := unzip(filepath.Join(paths.Libraries(), filepath.FromSlash(a.Path)), dir, exclude); err != nil {
			return err
		}
	}
	return nil
}

func unzip(src, dst string, exclude []string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
outer:
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || strings.Contains(f.Name, "..") {
			continue
		}
		for _, ex := range exclude {
			if strings.HasPrefix(f.Name, ex) {
				continue outer
			}
		}
		target := filepath.Join(dst, filepath.FromSlash(f.Name))
		if st, err := os.Stat(target); err == nil && st.Size() == int64(f.UncompressedSize64) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type assetIndex struct {
	Virtual        bool `json:"virtual"`
	MapToResources bool `json:"map_to_resources"`
	Objects        map[string]struct {
		Hash string `json:"hash"`
		Size int64  `json:"size"`
	} `json:"objects"`
}

// AssetsIndexName returns the asset index id used by the version.
func (v *Version) AssetsIndexName() string {
	if v.AssetIndex != nil {
		return v.AssetIndex.ID
	}
	if v.Assets != "" {
		return v.Assets
	}
	return "legacy"
}

// VirtualAssetsDir is used by pre-1.7 versions that read assets by name.
func VirtualAssetsDir(index string) string {
	return filepath.Join(paths.Assets(), "virtual", index)
}

func installAssets(ctx context.Context, v *Version, report Reporter) error {
	if v.AssetIndex == nil {
		return nil
	}
	indexPath := filepath.Join(paths.Assets(), "indexes", v.AssetIndex.ID+".json")
	err := download.One(ctx, download.File{URL: v.AssetIndex.URL, Path: indexPath, Hash: v.AssetIndex.SHA1, Size: v.AssetIndex.Size})
	if err != nil {
		return err
	}
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	var idx assetIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return err
	}

	objects := filepath.Join(paths.Assets(), "objects")
	var files []download.File
	seen := map[string]bool{}
	for _, o := range idx.Objects {
		if seen[o.Hash] {
			continue
		}
		seen[o.Hash] = true
		p := filepath.Join(objects, o.Hash[:2], o.Hash)
		// Hashing thousands of files each launch is slow: trust size for existing ones.
		if st, err := os.Stat(p); err == nil && st.Size() == o.Size {
			continue
		}
		files = append(files, download.File{URL: resourcesURL + o.Hash[:2] + "/" + o.Hash, Path: p, Hash: o.Hash, Size: o.Size})
	}
	if err := download.All(ctx, files, 32, func(d, t int) { report("Ресурсы", d, t) }); err != nil {
		return err
	}

	if !idx.Virtual && !idx.MapToResources {
		return nil
	}
	dir := VirtualAssetsDir(v.AssetIndex.ID)
	for name, o := range idx.Objects {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if st, err := os.Stat(target); err == nil && st.Size() == o.Size {
			continue
		}
		if err := copyFile(filepath.Join(objects, o.Hash[:2], o.Hash), target); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// LibraryClasspath returns the classpath jars of the version, client jar last.
func LibraryClasspath(v *Version) []string {
	var cp []string
	for _, l := range v.Libraries {
		if !Allowed(l.Rules) {
			continue
		}
		if a := l.MainArtifact(); a != nil {
			cp = append(cp, filepath.Join(paths.Libraries(), filepath.FromSlash(a.Path)))
		}
	}
	return append(cp, ClientJar(v.JarID))
}
