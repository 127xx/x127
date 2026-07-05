// Package server は x127 の HTTP API を提供し、埋め込み UI を配信する。
// registry.json はリクエストごとに読み直すため、registry.json を手動編集
// しても再起動なしで反映される。書き込みは mutex で直列化する(設計書の
// シングルプロセス前提)。
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/127xx/x127/internal/ports"
	"github.com/127xx/x127/internal/registry"
	"github.com/127xx/x127/web"
)

const version = "v0.1.0-dev"

// PortView は LISTEN 中のポート情報に台帳のラベルを重ねた API レスポンス。
// Active は実際に LISTEN 中かどうかを表す(台帳のみのエントリは false)。
type PortView struct {
	ports.Entry
	Name   string `json:"name,omitempty"`
	Note   string `json:"note,omitempty"`
	Active bool   `json:"active"`
}

// Server は HTTP API のハンドラを束ねる。scan はテスト可能にするため
// 注入し、mu は台帳への書き込みを直列化する。
type Server struct {
	regPath string
	scan    func() ([]ports.Entry, error)
	mu      sync.Mutex
}

// New は registry.json のパスとポートスキャナを受け取り Server を生成する。
func New(regPath string, scan func() ([]ports.Entry, error)) *Server {
	return &Server{regPath: regPath, scan: scan}
}

// Handler は API ルートと埋め込み UI を配線した http.Handler を返す。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/ports", s.listPorts)
	mux.HandleFunc("PUT /api/ports/{port}/label", s.putLabel)
	mux.HandleFunc("DELETE /api/ports/{port}/label", s.deleteLabel)

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err) // embed がビルド時に壊れている場合のみ。実行時には到達しない
	}
	mux.Handle("GET /", http.FileServerFS(dist))

	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
}

func (s *Server) listPorts(w http.ResponseWriter, r *http.Request) {
	entries, err := s.scan()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "port scan failed: "+err.Error())
		return
	}
	reg, err := registry.Load(s.regPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// LISTEN 中のエントリに台帳のラベルを重ねる。ここで見たポートを
	// activePorts に記録し、後段で台帳のみのエントリを判別する。
	views := make([]PortView, 0, len(entries))
	activePorts := map[int]bool{}
	for _, e := range entries {
		v := PortView{Entry: e, Active: true}
		if l, ok := reg.Ports[e.Port]; ok {
			v.Name, v.Note = l.Name, l.Note
		}
		views = append(views, v)
		activePorts[e.Port] = true
	}

	// 台帳にあるが LISTEN していないポートを active:false として追加する。
	for port, l := range reg.Ports {
		if !activePorts[port] {
			views = append(views, PortView{
				Entry: ports.Entry{Port: port, Proto: "tcp"},
				Name:  l.Name, Note: l.Note, Active: false,
			})
		}
	}

	writeJSON(w, http.StatusOK, views)
}

func (s *Server) putLabel(w http.ResponseWriter, r *http.Request) {
	port, ok := parsePort(w, r)
	if !ok {
		return
	}

	var l registry.Label
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(l.Name) == "" {
		writeError(w, http.StatusBadRequest, "name must not be empty")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := registry.Load(s.regPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reg.Set(port, l)
	if err := reg.Save(s.regPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteLabel(w http.ResponseWriter, r *http.Request) {
	port, ok := parsePort(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := registry.Load(s.regPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reg.Delete(port)
	if err := reg.Save(s.regPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parsePort はパスの {port} を検証して返す。不正なら 400 を書き込み false を返す。
func parsePort(w http.ResponseWriter, r *http.Request) (int, bool) {
	port, err := strconv.Atoi(r.PathValue("port"))
	if err != nil || port < 1 || port > 65535 {
		writeError(w, http.StatusBadRequest, "invalid port")
		return 0, false
	}
	return port, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
