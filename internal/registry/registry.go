// Package registry persists port labels to registry.json.
// The file is normally updated through the Web UI, but stays
// human-readable so it can be inspected or hand-fixed if needed;
// a corrupt file is a hard error, never silently re-initialized.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Label struct {
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
}

type Registry struct {
	Version int           `json:"version"`
	Ports   map[int]Label `json:"ports"`
}

func New() *Registry {
	return &Registry{Version: 1, Ports: map[int]Label{}}
}

func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}

	r := New()
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("registry file %s is corrupt: %w (fix or remove it manually)", path, err)
	}
	// 明示的な "ports": null は Unmarshal で nil マップになるため、
	// 後続の Set が nil マップへの代入で panic しないよう空マップに戻す。
	if r.Ports == nil {
		r.Ports = map[int]Label{}
	}

	return r, nil
}

func (r *Registry) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // rename 失敗時に中間ファイルを残さない
		return err
	}

	return nil
}

func (r *Registry) Set(port int, l Label) { r.Ports[port] = l }
func (r *Registry) Delete(port int)       { delete(r.Ports, port) }
