// Package mojang reads Mojang version metadata and installs game files.
package mojang

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/paths"
)

const manifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

type ManifestVersion struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	SHA1        string    `json:"sha1"`
	ReleaseTime time.Time `json:"releaseTime"`
}

type Manifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []ManifestVersion `json:"versions"`
}

// FetchManifest downloads the version list and caches it for offline use.
func FetchManifest(ctx context.Context) (*Manifest, error) {
	cache := filepath.Join(paths.Versions(), "version_manifest_v2.json")
	var m Manifest
	if err := download.JSON(ctx, manifestURL, &m); err == nil {
		if data, err := json.Marshal(&m); err == nil {
			os.MkdirAll(paths.Versions(), 0o755)
			os.WriteFile(cache, data, 0o644)
		}
		return &m, nil
	} else if data, rerr := os.ReadFile(cache); rerr == nil && json.Unmarshal(data, &m) == nil {
		return &m, nil
	} else {
		return nil, err
	}
}

type Artifact struct {
	Path string `json:"path"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type Rule struct {
	Action string `json:"action"`
	OS     *struct {
		Name string `json:"name"`
		Arch string `json:"arch"`
	} `json:"os"`
	Features map[string]bool `json:"features"`
}

// Allowed evaluates a rule list for Windows with no optional features.
func Allowed(rules []Rule) bool {
	if len(rules) == 0 {
		return true
	}
	allow := false
	for _, r := range rules {
		match := true
		if r.OS != nil {
			if r.OS.Name != "" && r.OS.Name != "windows" {
				match = false
			}
			if r.OS.Arch != "" && r.OS.Arch != archName() {
				match = false
			}
		}
		if len(r.Features) > 0 {
			match = false
		}
		if match {
			allow = r.Action == "allow"
		}
	}
	return allow
}

func archName() string {
	if runtime.GOARCH == "386" {
		return "x86"
	}
	return runtime.GOARCH
}

type Library struct {
	Name      string `json:"name"`
	URL       string `json:"url"` // maven repo for loader libraries (Fabric, etc.)
	SHA1      string `json:"sha1"`
	Size      int64  `json:"size"`
	Downloads struct {
		Artifact    *Artifact           `json:"artifact"`
		Classifiers map[string]Artifact `json:"classifiers"`
	} `json:"downloads"`
	Natives map[string]string `json:"natives"`
	Extract *struct {
		Exclude []string `json:"exclude"`
	} `json:"extract"`
	Rules []Rule `json:"rules"`
}

// MavenPath converts "group:artifact:version[:classifier]" to a repo path.
func MavenPath(name string) string {
	ext := "jar"
	if i := strings.Index(name, "@"); i >= 0 {
		name, ext = name[:i], name[i+1:]
	}
	p := strings.Split(name, ":")
	if len(p) < 3 {
		return ""
	}
	file := p[1] + "-" + p[2]
	if len(p) > 3 {
		file += "-" + p[3]
	}
	return strings.ReplaceAll(p[0], ".", "/") + "/" + p[1] + "/" + p[2] + "/" + file + "." + ext
}

// key identifies a library ignoring its version, used for de-duplication.
func (l Library) key() string {
	p := strings.Split(l.Name, ":")
	if len(p) < 3 {
		return l.Name
	}
	k := p[0] + ":" + p[1]
	if len(p) > 3 {
		k += ":" + p[3]
	}
	return k
}

// MainArtifact returns the classpath jar of a library, or nil.
func (l Library) MainArtifact() *Artifact {
	if a := l.Downloads.Artifact; a != nil {
		if a.Path == "" {
			a.Path = MavenPath(l.Name)
		}
		return a
	}
	if l.Natives != nil {
		return nil // legacy natives-only entry
	}
	path := MavenPath(l.Name)
	if path == "" {
		return nil
	}
	repo := l.URL
	if repo == "" {
		repo = "https://libraries.minecraft.net/"
	}
	if !strings.HasSuffix(repo, "/") {
		repo += "/"
	}
	return &Artifact{Path: path, URL: repo + path, SHA1: l.SHA1, Size: l.Size}
}

// NativeArtifact returns the legacy natives jar for Windows, or nil.
func (l Library) NativeArtifact() *Artifact {
	cls, ok := l.Natives["windows"]
	if !ok {
		return nil
	}
	bits := "64"
	if runtime.GOARCH == "386" {
		bits = "32"
	}
	cls = strings.ReplaceAll(cls, "${arch}", bits)
	if a, ok := l.Downloads.Classifiers[cls]; ok {
		return &a
	}
	return nil
}

// Argument is either a plain string or a conditional value.
type Argument struct {
	Rules []Rule
	Value []string
}

func (a *Argument) UnmarshalJSON(data []byte) error {
	var s string
	if json.Unmarshal(data, &s) == nil {
		a.Value = []string{s}
		return nil
	}
	var obj struct {
		Rules []Rule          `json:"rules"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a.Rules = obj.Rules
	if json.Unmarshal(obj.Value, &s) == nil {
		a.Value = []string{s}
		return nil
	}
	return json.Unmarshal(obj.Value, &a.Value)
}

func (a Argument) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Rules []Rule   `json:"rules,omitempty"`
		Value []string `json:"value"`
	}{a.Rules, a.Value})
}

type Version struct {
	ID           string `json:"id"`
	InheritsFrom string `json:"inheritsFrom,omitempty"`
	Type         string `json:"type"`
	MainClass    string `json:"mainClass"`
	Arguments    *struct {
		Game []Argument `json:"game"`
		JVM  []Argument `json:"jvm"`
	} `json:"arguments,omitempty"`
	MinecraftArguments string `json:"minecraftArguments,omitempty"`
	AssetIndex         *struct {
		ID   string `json:"id"`
		SHA1 string `json:"sha1"`
		Size int64  `json:"size"`
		URL  string `json:"url"`
	} `json:"assetIndex,omitempty"`
	Assets      string              `json:"assets,omitempty"`
	Downloads   map[string]Artifact `json:"downloads,omitempty"`
	Libraries   []Library           `json:"libraries"`
	JavaVersion *struct {
		Component    string `json:"component"`
		MajorVersion int    `json:"majorVersion"`
	} `json:"javaVersion,omitempty"`
	Logging map[string]struct {
		Argument string `json:"argument"`
		File     struct {
			ID   string `json:"id"`
			SHA1 string `json:"sha1"`
			Size int64  `json:"size"`
			URL  string `json:"url"`
		} `json:"file"`
	} `json:"logging,omitempty"`

	// JarID is the version whose client jar is used (set after merging).
	JarID string `json:"-"`
}

// JavaComponent returns the Mojang Java runtime required by the version.
func (v *Version) JavaComponent() string {
	if v.JavaVersion != nil && v.JavaVersion.Component != "" {
		return v.JavaVersion.Component
	}
	return "jre-legacy"
}

func versionFile(id string) string { return filepath.Join(paths.Version(id), id+".json") }

// ClientJar is the location of the client jar of a vanilla version.
func ClientJar(id string) string { return filepath.Join(paths.Version(id), id+".jar") }

// ReadLocal reads a version json from disk.
func ReadLocal(id string) (*Version, error) {
	data, err := os.ReadFile(versionFile(id))
	if err != nil {
		return nil, err
	}
	var v Version
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("%s.json: %w", id, err)
	}
	return &v, nil
}

// SaveLocal stores raw version json for id.
func SaveLocal(id string, data []byte) error {
	if err := os.MkdirAll(paths.Version(id), 0o755); err != nil {
		return err
	}
	return os.WriteFile(versionFile(id), data, 0o644)
}

// EnsureVanilla makes sure the vanilla version json is on disk.
func EnsureVanilla(ctx context.Context, id string) error {
	m, err := FetchManifest(ctx)
	if err != nil {
		if _, lerr := os.Stat(versionFile(id)); lerr == nil {
			return nil // offline, but already installed
		}
		return err
	}
	for _, mv := range m.Versions {
		if mv.ID == id {
			return download.One(ctx, download.File{URL: mv.URL, Path: versionFile(id), Hash: mv.SHA1})
		}
	}
	if _, err := os.Stat(versionFile(id)); err == nil {
		return nil
	}
	return fmt.Errorf("версия %s не найдена", id)
}

// Resolve reads a version and merges its inheritsFrom chain.
func Resolve(id string) (*Version, error) {
	v, err := ReadLocal(id)
	if err != nil {
		return nil, err
	}
	v.JarID = v.ID
	if v.InheritsFrom == "" {
		return v, nil
	}
	parent, err := Resolve(v.InheritsFrom)
	if err != nil {
		return nil, err
	}
	return merge(parent, v), nil
}

func merge(parent, child *Version) *Version {
	out := *parent
	out.ID = child.ID
	out.InheritsFrom = ""
	if child.MainClass != "" {
		out.MainClass = child.MainClass
	}
	if child.MinecraftArguments != "" {
		out.MinecraftArguments = child.MinecraftArguments
	}
	if child.Arguments != nil {
		if out.Arguments == nil {
			out.Arguments = child.Arguments
		} else {
			args := *out.Arguments
			args.Game = append(append([]Argument{}, args.Game...), child.Arguments.Game...)
			args.JVM = append(append([]Argument{}, args.JVM...), child.Arguments.JVM...)
			out.Arguments = &args
		}
	}
	if child.JavaVersion != nil {
		out.JavaVersion = child.JavaVersion
	}
	// Child libraries win over parent ones with the same group:artifact.
	seen := map[string]bool{}
	var libs []Library
	for _, l := range append(append([]Library{}, child.Libraries...), parent.Libraries...) {
		k := l.key()
		if l.Natives == nil && seen[k] {
			continue
		}
		seen[k] = true
		libs = append(libs, l)
	}
	out.Libraries = libs
	return &out
}
