// Package paths describes the on-disk layout of the launcher data directory.
//
//	%APPDATA%\Rabbit\
//	  assets\          shared between all instances
//	  libraries\       shared between all instances
//	  versions\<id>\   version json, client jar, natives
//	  runtimes\<comp>\ Java runtimes downloaded from Mojang
//	  instances\<id>\  game directory of each instance (mods, saves, config...)
package paths

import (
	"os"
	"path/filepath"
)

var root string

func init() {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	root = filepath.Join(dir, "Rabbit")
}

func Root() string                  { return root }
func Assets() string                { return filepath.Join(root, "assets") }
func Libraries() string             { return filepath.Join(root, "libraries") }
func Versions() string              { return filepath.Join(root, "versions") }
func Version(id string) string      { return filepath.Join(root, "versions", id) }
func Runtimes() string              { return filepath.Join(root, "runtimes") }
func Instances() string             { return filepath.Join(root, "instances") }
func Instance(id string) string     { return filepath.Join(root, "instances", id) }
func AuthlibInjector() string       { return filepath.Join(root, "authlib-injector.jar") }
func ConfigFile(name string) string { return filepath.Join(root, name) }
