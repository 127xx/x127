// Package config resolves x127's config directory and file paths.
// Precedence: $X127_CONFIG_DIR > $XDG_CONFIG_HOME/x127 > ~/.config/x127.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	// X127_CONFIG_DIR is an explicit override: a relative value would make
	// the config path depend on each command's CWD (e.g. serve and stop run
	// from different directories would miss each other's pid file), so reject
	// it loudly rather than silently resolving it against the CWD.
	if d := os.Getenv("X127_CONFIG_DIR"); d != "" {
		if !filepath.IsAbs(d) {
			return "", fmt.Errorf("X127_CONFIG_DIR must be an absolute path, got %q", d)
		}
		return d, nil
	}
	// XDG spec: a relative XDG_CONFIG_HOME is invalid and must be ignored,
	// so fall through to ~/.config to avoid a CWD-dependent config path.
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" && filepath.IsAbs(x) {
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
	// 0o700: the dir holds registry.json / x127.pid / x127.log, so keep it
	// private to the owning user (other users must not list or traverse it).
	return d, os.MkdirAll(d, 0o700)
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
