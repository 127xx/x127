package config

import (
	"path/filepath"
	"testing"
)

func TestDirPrecedence(t *testing.T) {
	t.Setenv("X127_CONFIG_DIR", "/tmp/explicit")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got, err := Dir()
	if err != nil || got != "/tmp/explicit" {
		t.Fatalf("Dir() = %q, %v; want /tmp/explicit", got, err)
	}

	t.Setenv("X127_CONFIG_DIR", "")
	got, err = Dir()
	if err != nil || got != filepath.Join("/tmp/xdg", "x127") {
		t.Fatalf("Dir() = %q, %v; want /tmp/xdg/x127", got, err)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")
	got, err = Dir()
	if err != nil || got != filepath.Join("/tmp/home", ".config", "x127") {
		t.Fatalf("Dir() = %q, %v; want ~/.config/x127", got, err)
	}
}

func TestDirIgnoresRelativeXDG(t *testing.T) {
	// XDG spec: a relative XDG_CONFIG_HOME is invalid and must be ignored,
	// falling back to ~/.config so the path never depends on the CWD.
	t.Setenv("X127_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", ".config")
	t.Setenv("HOME", "/tmp/home")
	got, err := Dir()
	if err != nil || got != filepath.Join("/tmp/home", ".config", "x127") {
		t.Fatalf("Dir() = %q, %v; want ~/.config/x127 (relative XDG ignored)", got, err)
	}
}

func TestFilePaths(t *testing.T) {
	t.Setenv("X127_CONFIG_DIR", "/tmp/cfg")
	reg, err := RegistryPath()
	if err != nil {
		t.Fatalf("RegistryPath() error = %v", err)
	}
	pid, err := PIDPath()
	if err != nil {
		t.Fatalf("PIDPath() error = %v", err)
	}
	log, err := LogPath()
	if err != nil {
		t.Fatalf("LogPath() error = %v", err)
	}
	if reg != "/tmp/cfg/registry.json" || pid != "/tmp/cfg/x127.pid" || log != "/tmp/cfg/x127.log" {
		t.Fatalf("paths = %q %q %q", reg, pid, log)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "x127")
	t.Setenv("X127_CONFIG_DIR", dir)
	got, err := EnsureDir()
	if err != nil || got != dir {
		t.Fatalf("EnsureDir() = %q, %v", got, err)
	}
}
