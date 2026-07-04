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

func TestLoadNilPortsIsUsable(t *testing.T) {
	// ports キー欠落・明示的な null のどちらでも、Load 後に Set しても
	// nil マップへの代入で panic しないこと(Ports が非 nil であること)。
	for name, body := range map[string]string{
		"missing key":   `{"version":1}`,
		"explicit null": `{"version":1,"ports":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "registry.json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			r, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			r.Set(12701, Label{Name: "a"})
			if r.Ports[12701].Name != "a" {
				t.Fatalf("Set() after Load = %+v", r.Ports)
			}
		})
	}
}

func TestLoadCorruptFileErrorsWithPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() = nil error, want corrupt-file error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error %q does not mention path", err)
	}
}
