// Package modrinth talks to the Modrinth API: search, installing content
// into instances and .mrpack modpacks.
package modrinth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vellnico/rabbit/internal/download"
)

const api = "https://api.modrinth.com/v2"

func get(ctx context.Context, path string, v any) error {
	return download.JSON(ctx, api+path, v)
}

func post(ctx context.Context, path string, body, v any) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := download.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("modrinth: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func jsonList(items ...string) string {
	b, _ := json.Marshal(items)
	return string(b)
}

// Project types Rabbit can browse. Datapacks and plugins are not shown.
const (
	TypeMod          = "mod"
	TypeResourcePack = "resourcepack"
	TypeShader       = "shader"
	TypeModpack      = "modpack"
)

// Folder is where content of a type lives inside an instance.
func Folder(projectType string) string {
	switch projectType {
	case TypeMod:
		return "mods"
	case TypeResourcePack:
		return "resourcepacks"
	case TypeShader:
		return "shaderpacks"
	}
	return ""
}

// loaderNames maps an instance loader to the Modrinth loaders whose mods it runs.
func loaderNames(loader string) []string {
	switch loader {
	case "quilt":
		return []string{"quilt", "fabric"}
	case "fabric", "forge", "neoforge":
		return []string{loader}
	}
	return nil
}

type Hit struct {
	ProjectID   string   `json:"project_id"`
	Slug        string   `json:"slug"`
	Type        string   `json:"project_type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	IconURL     string   `json:"icon_url"`
	Downloads   int      `json:"downloads"`
	Follows     int      `json:"follows"`
	Categories  []string `json:"display_categories"`
	Versions    []string `json:"versions"`
}

type SearchResult struct {
	Hits      []Hit `json:"hits"`
	Offset    int   `json:"offset"`
	Limit     int   `json:"limit"`
	TotalHits int   `json:"total_hits"`
}

type SearchParams struct {
	Query       string `json:"query"`
	Type        string `json:"type"`
	GameVersion string `json:"gameVersion"` // empty: any
	Loader      string `json:"loader"`      // instance loader, used for mods
	Sort        string `json:"sort"`        // relevance, downloads, follows, newest, updated
	Offset      int    `json:"offset"`
}

func Search(ctx context.Context, p SearchParams) (*SearchResult, error) {
	facets := [][]string{{"project_type:" + p.Type}}
	if p.GameVersion != "" {
		facets = append(facets, []string{"versions:" + p.GameVersion})
	}
	if p.Type == TypeMod {
		var or []string
		for _, l := range loaderNames(p.Loader) {
			or = append(or, "categories:"+l)
		}
		if len(or) > 0 {
			facets = append(facets, or)
		}
	} else if p.Type == TypeShader && p.Loader == "optifine" {
		facets = append(facets, []string{"categories:optifine"})
	}
	fb, _ := json.Marshal(facets)
	sort := p.Sort
	switch sort {
	case "downloads", "follows", "newest", "updated":
	default:
		sort = "relevance"
	}
	q := url.Values{
		"query":  {strings.TrimSpace(p.Query)},
		"facets": {string(fb)},
		"index":  {sort},
		"offset": {strconv.Itoa(p.Offset)},
		"limit":  {"20"},
	}
	var res SearchResult
	if err := get(ctx, "/search?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	return &res, nil
}

type File struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int64             `json:"size"`
	Hashes   map[string]string `json:"hashes"`
}

type Dependency struct {
	ProjectID string `json:"project_id"`
	VersionID string `json:"version_id"`
	Type      string `json:"dependency_type"` // required, optional, incompatible, embedded
}

type Version struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"project_id"`
	Name          string       `json:"name"`
	VersionNumber string       `json:"version_number"`
	VersionType   string       `json:"version_type"` // release, beta, alpha
	GameVersions  []string     `json:"game_versions"`
	Loaders       []string     `json:"loaders"`
	Files         []File       `json:"files"`
	Dependencies  []Dependency `json:"dependencies"`
}

// PrimaryFile returns the main downloadable file of a version.
func (v *Version) PrimaryFile() *File {
	for i := range v.Files {
		if v.Files[i].Primary {
			return &v.Files[i]
		}
	}
	if len(v.Files) > 0 {
		return &v.Files[0]
	}
	return nil
}

type Project struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Type        string `json:"project_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
}

// ProjectVersions lists versions of a project, newest first, optionally
// filtered by game version and loaders.
func ProjectVersions(ctx context.Context, id, game string, loaders []string) ([]Version, error) {
	q := url.Values{}
	if game != "" {
		q.Set("game_versions", jsonList(game))
	}
	if len(loaders) > 0 {
		q.Set("loaders", jsonList(loaders...))
	}
	var list []Version
	err := get(ctx, "/project/"+url.PathEscape(id)+"/version?"+q.Encode(), &list)
	return list, err
}

func GetVersion(ctx context.Context, id string) (*Version, error) {
	var v Version
	if err := get(ctx, "/version/"+url.PathEscape(id), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetProject(ctx context.Context, id string) (*Project, error) {
	var p Project
	if err := get(ctx, "/project/"+url.PathEscape(id), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProjects fetches several projects at once.
func GetProjects(ctx context.Context, ids []string) ([]Project, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Project
	err := get(ctx, "/projects?ids="+url.QueryEscape(jsonList(ids...)), &list)
	return list, err
}

// VersionsByHash looks up versions by file sha1 hashes.
func VersionsByHash(ctx context.Context, sha1s []string) (map[string]Version, error) {
	out := map[string]Version{}
	if len(sha1s) == 0 {
		return out, nil
	}
	err := post(ctx, "/version_files", map[string]any{"hashes": sha1s, "algorithm": "sha1"}, &out)
	return out, err
}
