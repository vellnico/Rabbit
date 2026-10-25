package mojang

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/paths"
)

// OptiFine has no API: versions are read from the downloads page and the
// jar is fetched through its mirror page, which hands out a one-time link.
// Installation repeats what the official installer does: the OptiFine
// library is patched against the vanilla jar and a version inheriting from
// vanilla starts the game through launchwrapper with the OptiFine tweaker.

const optifineSite = "https://optifine.net/"

var (
	optifineFile = regexp.MustCompile(`adloadx\?f=(preview_)?OptiFine_([0-9.]+)_(HD_U_[A-Za-z0-9_]+)\.jar`)
	optifineLink = regexp.MustCompile(`downloadx\?f=[^'"\s<>]+`)
)

// IsOptiFine reports whether the loader is OptiFine.
func IsOptiFine(loader string) bool { return loader == "optifine" }

// OptiFineVersions lists OptiFine editions (e.g. "HD_U_I6") for a game
// version: releases newest first, then previews ("HD_U_J1_pre9").
func OptiFineVersions(ctx context.Context, game string) ([]string, error) {
	resp, err := download.Get(ctx, optifineSite+"downloads")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	page, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var stable, pre []string
	for _, m := range optifineFile.FindAllStringSubmatch(string(page), -1) {
		if m[2] != game || seen[m[3]] {
			continue
		}
		seen[m[3]] = true
		if m[1] != "" {
			pre = append(pre, m[3])
		} else {
			stable = append(stable, m[3])
		}
	}
	return append(stable, pre...), nil
}

func optifineFileName(game, edition string) string {
	name := "OptiFine_" + game + "_" + edition + ".jar"
	if strings.Contains(edition, "_pre") {
		name = "preview_" + name
	}
	return name
}

// OptiFineVersionID is the id of the installed version, the same as the
// official installer uses.
func OptiFineVersionID(game, edition string) string { return game + "-OptiFine_" + edition }

func optifineLibrary(game, edition string) (name, path string) {
	v := game + "_" + edition
	name = "optifine:OptiFine:" + v
	return name, MavenPath(name)
}

func downloadOptiFine(ctx context.Context, game, edition, dest string) error {
	resp, err := download.Get(ctx, optifineSite+"adloadx?f="+url.QueryEscape(optifineFileName(game, edition)))
	if err != nil {
		return err
	}
	page, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	link := optifineLink.FindString(string(page))
	if link == "" {
		return fmt.Errorf("OptiFine %s %s: download link not found", game, edition)
	}
	return download.One(ctx, download.File{URL: optifineSite + html.UnescapeString(link), Path: dest})
}

// EnsureOptiFine installs OptiFine for an installed vanilla version using
// java (needed by the OptiFine patcher) and returns the version id.
func EnsureOptiFine(ctx context.Context, game, edition, java string) (string, error) {
	id := OptiFineVersionID(game, edition)
	libName, libPath := optifineLibrary(game, edition)
	lib := filepath.Join(paths.Libraries(), filepath.FromSlash(libPath))
	if _, err := os.Stat(versionFile(id)); err == nil {
		if _, err := os.Stat(lib); err == nil {
			return id, nil
		}
	}
	parent, err := ReadLocal(game)
	if err != nil {
		return "", err
	}

	tmp, err := os.MkdirTemp("", "rabbit-optifine-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	jar := filepath.Join(tmp, "OptiFine.jar")
	if err := downloadOptiFine(ctx, game, edition, jar); err != nil {
		return "", err
	}

	zr, err := zip.OpenReader(jar)
	if err != nil {
		return "", fmt.Errorf("OptiFine jar: %w", err)
	}
	defer zr.Close()
	patch := false
	var lwVersion string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "patch/") {
			patch = true
		}
		if f.Name == "launchwrapper-of.txt" {
			data, err := readZipFile(f)
			if err != nil {
				return "", err
			}
			lwVersion = strings.TrimSpace(string(data))
		}
	}

	// Newer builds ship their own launchwrapper; older ones use Mojang's.
	lwLib := map[string]any{"name": "net.minecraft:launchwrapper:1.12"}
	if lwVersion != "" {
		name := "optifine:launchwrapper-of:" + lwVersion
		path := MavenPath(name)
		if err := extractZipFile(&zr.Reader, "launchwrapper-of-"+lwVersion+".jar", filepath.Join(paths.Libraries(), filepath.FromSlash(path))); err != nil {
			return "", err
		}
		lwLib = map[string]any{"name": name, "downloads": map[string]any{"artifact": map[string]any{"path": path}}}
	}

	if err := os.MkdirAll(filepath.Dir(lib), 0o755); err != nil {
		return "", err
	}
	if patch {
		// The patcher writes OptiFine classes merged with the vanilla ones.
		out := lib + ".part"
		if err := runJava(ctx, java, tmp, "-cp", jar, "optifine.Patcher", ClientJar(game), jar, out); err != nil {
			os.Remove(out)
			return "", fmt.Errorf("OptiFine patcher: %w", err)
		}
		if err := os.Rename(out, lib); err != nil {
			return "", err
		}
	} else if err := copyFile(jar, lib); err != nil {
		return "", err
	}

	mainClass := parent.MainClass
	if !strings.HasPrefix(mainClass, "net.minecraft.launchwrapper.") {
		mainClass = "net.minecraft.launchwrapper.Launch"
	}
	now := time.Now().Format(time.RFC3339)
	v := map[string]any{
		"id":           id,
		"inheritsFrom": game,
		"type":         "release",
		"time":         now,
		"releaseTime":  now,
		"mainClass":    mainClass,
		"libraries": []any{
			map[string]any{"name": libName, "downloads": map[string]any{"artifact": map[string]any{"path": libPath}}},
			lwLib,
		},
	}
	if parent.MinecraftArguments != "" {
		v["minecraftArguments"] = parent.MinecraftArguments + " --tweakClass optifine.OptiFineTweaker"
	} else {
		v["arguments"] = map[string]any{"game": []string{"--tweakClass", "optifine.OptiFineTweaker"}}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return id, SaveLocal(id, data)
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func extractZipFile(zr *zip.Reader, name, dst string) error {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		data, err := readZipFile(f)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	}
	return fmt.Errorf("%s not found in OptiFine jar", name)
}
