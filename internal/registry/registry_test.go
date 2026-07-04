package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	r, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if r.Version != 1 || len(r.Ports) != 0 {
		t.Fatalf("Load() = %+v, want empty v1 registry", r)
	}
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, _ := Load(path)
	r.Set(12701, Label{Name: "llama.cpp", Note: "テスト"})
	if err := r.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	r2, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := r2.Ports[12701]; got.Name != "llama.cpp" || got.Note != "テスト" {
		t.Fatalf("Ports[12701] = %+v", got)
	}
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, _ := Load(path)
	r.Set(8080, Label{Name: "a"})
	r.Delete(8080)
	if _, ok := r.Ports[8080]; ok {
		t.Fatal("Delete() did not remove entry")
	}
}

func TestLoadFileWithoutPortsKeyIsUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	os.WriteFile(path, []byte(`{"version":1}`), 0o600)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// ports キーが無くても Set が panic しない(Ports が非 nil であること)
	r.Set(12701, Label{Name: "a"})
	if r.Ports[12701].Name != "a" {
		t.Fatalf("Set() after Load = %+v", r.Ports)
	}
}

func TestLoadCorruptFileErrorsWithPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	os.WriteFile(path, []byte("{broken"), 0o644)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() = nil error, want corrupt-file error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error %q does not mention path", err)
	}
}
