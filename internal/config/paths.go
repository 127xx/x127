// Package config resolves x127's config directory and file paths.
// Precedence: $X127_CONFIG_DIR > $XDG_CONFIG_HOME/x127 > ~/.config/x127.
package config

import (
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	if d := os.Getenv("X127_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "x127"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "x127"), nil
}

func EnsureDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return d, os.MkdirAll(d, 0o755)
}

func RegistryPath() (string, error) { return join("registry.json") }
func PIDPath() (string, error)      { return join("x127.pid") }
func LogPath() (string, error)      { return join("x127.log") }

func join(name string) (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}
