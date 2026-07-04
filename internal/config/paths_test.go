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

func TestFilePaths(t *testing.T) {
	t.Setenv("X127_CONFIG_DIR", "/tmp/cfg")
	reg, _ := RegistryPath()
	pid, _ := PIDPath()
	log, _ := LogPath()
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
