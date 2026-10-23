// Package java installs Java runtimes published by Mojang.
package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/paths"
)

const allURL = "https://launchermeta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"

type runtimeEntry struct {
	Manifest struct {
		SHA1 string `json:"sha1"`
		URL  string `json:"url"`
	} `json:"manifest"`
	Version struct {
		Name string `json:"name"`
	} `json:"version"`
}

type fileManifest struct {
	Files map[string]struct {
		Type      string `json:"type"`
		Downloads struct {
			Raw struct {
				SHA1 string `json:"sha1"`
				Size int64  `json:"size"`
				URL  string `json:"url"`
			} `json:"raw"`
		} `json:"downloads"`
	} `json:"files"`
}

func platform() string {
	switch runtime.GOARCH {
	case "386":
		return "windows-x86"
	case "arm64":
		return "windows-arm64"
	}
	return "windows-x64"
}

func dir(component string) string { return filepath.Join(paths.Runtimes(), component) }

// Executable returns javaw.exe of an installed component.
func Executable(component string) string {
	return filepath.Join(dir(component), "bin", "javaw.exe")
}

// Ensure installs the given runtime component (e.g. "java-runtime-delta")
// if it is missing or outdated, and returns the path to javaw.exe.
func Ensure(ctx context.Context, component string, report func(done, total int)) (string, error) {
	marker := filepath.Join(dir(component), ".rabbit-version")
	exe := Executable(component)

	var all map[string]map[string][]runtimeEntry
	if err := download.JSON(ctx, allURL, &all); err != nil {
		if _, serr := os.Stat(exe); serr == nil {
			return exe, nil // offline but installed
		}
		return "", err
	}
	entries := all[platform()][component]
	if len(entries) == 0 && platform() == "windows-arm64" {
		entries = all["windows-x64"][component]
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no Java runtime %s for %s", component, platform())
	}
	e := entries[0]
	if cur, err := os.ReadFile(marker); err == nil && string(cur) == e.Manifest.SHA1 {
		if _, err := os.Stat(exe); err == nil {
			return exe, nil
		}
	}

	var fm fileManifest
	if err := download.JSON(ctx, e.Manifest.URL, &fm); err != nil {
		return "", err
	}
	var files []download.File
	for name, f := range fm.Files {
		target := filepath.Join(dir(component), filepath.FromSlash(name))
		switch f.Type {
		case "directory":
			os.MkdirAll(target, 0o755)
		case "file":
			r := f.Downloads.Raw
			files = append(files, download.File{URL: r.URL, Path: target, Hash: r.SHA1, Size: r.Size})
		}
	}
	if err := download.All(ctx, files, 16, report); err != nil {
		return "", err
	}
	if err := os.WriteFile(marker, []byte(e.Manifest.SHA1), 0o644); err != nil {
		return "", err
	}
	return exe, nil
}
