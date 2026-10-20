// Package store persists launcher settings, accounts and instances.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/auth"
	"github.com/vellnico/rabbit/internal/paths"
)

type Settings struct {
	MemoryMB         int    `json:"memoryMb"`
	SelectedAccount  string `json:"selectedAccount"`
	SelectedInstance string `json:"selectedInstance"`
	CloseOnLaunch    bool   `json:"closeOnLaunch"`
	JVMArgs          string `json:"jvmArgs"`
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func LoadSettings() Settings {
	s := Settings{MemoryMB: 4096}
	readJSON(paths.ConfigFile("settings.json"), &s)
	if s.MemoryMB < 512 {
		s.MemoryMB = 4096
	}
	return s
}

func SaveSettings(s Settings) error { return writeJSON(paths.ConfigFile("settings.json"), s) }

func LoadAccounts() []auth.Account {
	var list []auth.Account
	readJSON(paths.ConfigFile("accounts.json"), &list)
	return list
}

func SaveAccounts(list []auth.Account) error {
	return writeJSON(paths.ConfigFile("accounts.json"), list)
}

type Instance struct {
	ID            string    `json:"id"` // folder name
	Name          string    `json:"name"`
	GameVersion   string    `json:"gameVersion"`
	Loader        string    `json:"loader"` // "vanilla" or "fabric"
	LoaderVersion string    `json:"loaderVersion,omitempty"`
	Created       time.Time `json:"created"`
	LastPlayed    time.Time `json:"lastPlayed,omitempty"`
}

// GameDirs are created inside every instance.
var GameDirs = []string{"mods", "resourcepacks", "shaderpacks", "saves", "config"}

func instanceFile(id string) string { return filepath.Join(paths.Instance(id), "instance.json") }

func LoadInstances() []Instance {
	entries, _ := os.ReadDir(paths.Instances())
	var list []Instance
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var in Instance
		if readJSON(instanceFile(e.Name()), &in) == nil {
			in.ID = e.Name()
			list = append(list, in)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Created.Before(list[j].Created) })
	return list
}

func GetInstance(id string) (Instance, error) {
	var in Instance
	if err := readJSON(instanceFile(id), &in); err != nil {
		return in, errors.New("сборка не найдена")
	}
	in.ID = id
	return in, nil
}

func SaveInstance(in Instance) error { return writeJSON(instanceFile(in.ID), in) }

var unsafeChars = regexp.MustCompile(`[^\p{L}\p{N} ._-]+`)

// CreateInstance makes a new instance folder with a unique id.
func CreateInstance(in Instance) (Instance, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = in.GameVersion
	}
	base := strings.Trim(unsafeChars.ReplaceAllString(in.Name, "_"), " .")
	if base == "" {
		base = "instance"
	}
	id := base
	for i := 2; ; i++ {
		if _, err := os.Stat(paths.Instance(id)); os.IsNotExist(err) {
			break
		}
		id = base + " (" + strconv.Itoa(i) + ")"
	}
	in.ID = id
	in.Created = time.Now()
	for _, d := range GameDirs {
		if err := os.MkdirAll(filepath.Join(paths.Instance(id), d), 0o755); err != nil {
			return in, err
		}
	}
	return in, SaveInstance(in)
}

func DeleteInstance(id string) error {
	if id == "" || strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return errors.New("неверная сборка")
	}
	return os.RemoveAll(paths.Instance(id))
}
