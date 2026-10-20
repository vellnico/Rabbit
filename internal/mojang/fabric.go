package mojang

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/vellnico/rabbit/internal/download"
)

// fabricLike describes loaders that publish ready version profiles
// (Fabric and its fork Quilt share the same meta API).
type fabricLike struct {
	meta   string
	prefix string
}

var fabricLoaders = map[string]fabricLike{
	"fabric": {"https://meta.fabricmc.net/v2/versions/loader/", "fabric-loader"},
	"quilt":  {"https://meta.quiltmc.org/v3/versions/loader/", "quilt-loader"},
}

// IsProfileLoader reports whether the loader installs from a meta profile.
func IsProfileLoader(loader string) bool {
	_, ok := fabricLoaders[loader]
	return ok
}

// ProfileLoaderVersions lists loader versions for a game version, newest first.
func ProfileLoaderVersions(ctx context.Context, loader, game string) ([]string, error) {
	fl := fabricLoaders[loader]
	var list []struct {
		Loader struct {
			Version string `json:"version"`
		} `json:"loader"`
	}
	if err := download.JSON(ctx, fl.meta+url.PathEscape(game), &list); err != nil {
		return nil, err
	}
	// Stable versions first; pre-releases ("0.20.0-beta.9") after them.
	var stable, pre []string
	for _, l := range list {
		if strings.Contains(l.Loader.Version, "-") || strings.Contains(l.Loader.Version, "+build") {
			pre = append(pre, l.Loader.Version)
		} else {
			stable = append(stable, l.Loader.Version)
		}
	}
	return append(stable, pre...), nil
}

// EnsureProfileLoader installs the loader profile json (it inherits from
// vanilla) and returns its version id.
func EnsureProfileLoader(ctx context.Context, loader, game, version string) (string, error) {
	fl := fabricLoaders[loader]
	id := fmt.Sprintf("%s-%s-%s", fl.prefix, version, game)
	if _, err := os.Stat(versionFile(id)); err == nil {
		return id, nil
	}
	resp, err := download.Get(ctx, fl.meta+url.PathEscape(game)+"/"+url.PathEscape(version)+"/profile/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return id, SaveLocal(id, data)
}
