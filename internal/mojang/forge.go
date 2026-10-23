package mojang

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/paths"
)

// Forge and NeoForge are installed by running their official installers
// with --installClient into the launcher root: it has the same layout as
// .minecraft (versions/, libraries/), so the result is shared by all
// instances.

const (
	forgeMaven    = "https://maven.minecraftforge.net/net/minecraftforge/forge/"
	neoforgeMaven = "https://maven.neoforged.net/releases/net/neoforged/neoforge/"
	neoforgeAPI   = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/neoforge"
)

// IsInstallerLoader reports whether the loader uses an installer jar.
func IsInstallerLoader(loader string) bool { return loader == "forge" || loader == "neoforge" }

// ForgeVersions lists Forge versions for a game version, newest first.
// The returned values are the loader part only (e.g. "47.3.0").
func ForgeVersions(ctx context.Context, game string) ([]string, error) {
	resp, err := download.Get(ctx, forgeMaven+"maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var meta struct {
		Versions []string `xml:"versioning>versions>version"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}
	var out []string
	for _, v := range meta.Versions {
		if rest, ok := strings.CutPrefix(v, game+"-"); ok {
			out = append(out, rest)
		}
	}
	return out, nil
}

// neoforgeGame maps a NeoForge version to its Minecraft version:
// 21.1.77 -> 1.21.1, 21.0.5 -> 1.21, 26.3.0.95 -> 26.3, 26.1.1.4 -> 26.1.1.
func neoforgeGame(v string) string {
	v, _, _ = strings.Cut(v, "-")
	p := strings.Split(v, ".")
	if len(p) < 3 {
		return ""
	}
	major, err := strconv.Atoi(p[0])
	if err != nil {
		return ""
	}
	if major >= 25 { // year based versions
		if p[2] == "0" {
			return p[0] + "." + p[1]
		}
		return p[0] + "." + p[1] + "." + p[2]
	}
	if p[1] == "0" {
		return "1." + p[0]
	}
	return "1." + p[0] + "." + p[1]
}

var digitsDot = regexp.MustCompile(`^[0-9.]+(-beta)?$`)

// NeoForgeVersions lists NeoForge versions for a game version, newest first.
func NeoForgeVersions(ctx context.Context, game string) ([]string, error) {
	var res struct {
		Versions []string `json:"versions"`
	}
	if err := download.JSON(ctx, neoforgeAPI, &res); err != nil {
		return nil, err
	}
	var stable, beta []string
	for i := len(res.Versions) - 1; i >= 0; i-- {
		v := res.Versions[i]
		if !digitsDot.MatchString(v) || neoforgeGame(v) != game {
			continue
		}
		if strings.HasSuffix(v, "-beta") {
			beta = append(beta, v)
		} else {
			stable = append(stable, v)
		}
	}
	return append(stable, beta...), nil
}

func installerURL(loader, game, version string) string {
	if loader == "forge" {
		full := game + "-" + version
		return forgeMaven + full + "/forge-" + full + "-installer.jar"
	}
	return neoforgeMaven + version + "/neoforge-" + version + "-installer.jar"
}

// installerVersionID reads the id of the version the installer creates.
func installerVersionID(jar string) (string, error) {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "version.json" && f.Name != "install_profile.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		var v struct {
			ID          string `json:"id"`
			VersionInfo struct {
				ID string `json:"id"`
			} `json:"versionInfo"` // very old installers
		}
		json.Unmarshal(data, &v)
		if v.ID != "" {
			return v.ID, nil
		}
		if v.VersionInfo.ID != "" {
			return v.VersionInfo.ID, nil
		}
	}
	return "", fmt.Errorf("version id not found in installer")
}

// EnsureInstallerLoader installs Forge/NeoForge using java (a java.exe or
// javaw.exe path) and returns the installed version id.
func EnsureInstallerLoader(ctx context.Context, loader, game, version, java string) (string, error) {
	// Modern installers create these ids: skip downloading the installer
	// (several MB) on every launch once the loader is in place.
	guess := "neoforge-" + version
	if loader == "forge" {
		guess = game + "-forge-" + version
	}
	if _, err := os.Stat(versionFile(guess)); err == nil {
		return guess, nil
	}

	tmp, err := os.MkdirTemp("", "rabbit-installer-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	jar := filepath.Join(tmp, "installer.jar")
	if err := download.One(ctx, download.File{URL: installerURL(loader, game, version), Path: jar}); err != nil {
		return "", err
	}
	id, err := installerVersionID(jar)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(versionFile(id)); err == nil {
		return id, nil
	}

	// The installer refuses to run without a launcher profile file.
	profiles := filepath.Join(paths.Root(), "launcher_profiles.json")
	if _, err := os.Stat(profiles); err != nil {
		os.WriteFile(profiles, []byte(`{"profiles":{}}`), 0o644)
	}

	if dir := filepath.Dir(java); strings.EqualFold(filepath.Base(java), "javaw.exe") {
		java = filepath.Join(dir, "java.exe")
	}
	cmd := exec.CommandContext(ctx, java, "-jar", jar, "--installClient", paths.Root())
	cmd.Dir = tmp
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		return "", fmt.Errorf("%v\n%s", err, strings.Join(lines, "\n"))
	}
	if _, err := os.Stat(versionFile(id)); err != nil {
		return "", fmt.Errorf("installer did not create %s", id)
	}
	return id, nil
}
