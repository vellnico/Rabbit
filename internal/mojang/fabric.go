package mojang

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/vellnico/rabbit/internal/download"
)

const fabricMeta = "https://meta.fabricmc.net/v2/versions/loader/"

// FabricLoaders lists stable Fabric loader versions for a game version.
func FabricLoaders(ctx context.Context, game string) ([]string, error) {
	var list []struct {
		Loader struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
		} `json:"loader"`
	}
	if err := download.JSON(ctx, fabricMeta+url.PathEscape(game), &list); err != nil {
		return nil, err
	}
	var out []string
	for _, l := range list {
		out = append(out, l.Loader.Version)
	}
	return out, nil
}

// FabricVersionID is the local version id of a Fabric profile.
func FabricVersionID(game, loader string) string {
	return fmt.Sprintf("fabric-loader-%s-%s", loader, game)
}

// EnsureFabric installs the Fabric profile json (it inherits from vanilla).
func EnsureFabric(ctx context.Context, game, loader string) (string, error) {
	id := FabricVersionID(game, loader)
	if _, err := os.Stat(versionFile(id)); err == nil {
		return id, nil
	}
	resp, err := download.Get(ctx, fabricMeta+url.PathEscape(game)+"/"+url.PathEscape(loader)+"/profile/json")
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
