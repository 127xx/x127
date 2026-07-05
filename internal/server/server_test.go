package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/127xx/x127/internal/ports"
	"github.com/127xx/x127/internal/registry"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	regPath := filepath.Join(t.TempDir(), "registry.json")
	fake := func() ([]ports.Entry, error) {
		return []ports.Entry{
			{Port: 8080, Proto: "tcp", Address: "127.0.0.1", PID: 42, Process: "llama-server"},
		}, nil
	}
	return New(regPath, fake), regPath
}

func TestHealth(t *testing.T) {
	s, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestListPortsMergesRegistry(t *testing.T) {
	s, regPath := newTestServer(t)
	r, err := registry.Load(regPath)
	if err != nil {
		t.Fatal(err)
	}
	r.Set(8080, registry.Label{Name: "llama.cpp"})
	r.Set(9999, registry.Label{Name: "予約だけ", Note: "止まってる"})
	if err := r.Save(regPath); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/ports", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var views []PortView
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatal(err)
	}
	byPort := map[int]PortView{}
	for _, v := range views {
		byPort[v.Port] = v
	}
	if v := byPort[8080]; !v.Active || v.Name != "llama.cpp" || v.Process != "llama-server" {
		t.Fatalf("8080 = %+v", v)
	}
	if v := byPort[9999]; v.Active || v.Name != "予約だけ" {
		t.Fatalf("9999 = %+v", v)
	}
}

func TestListPortsKeepsMultipleListenersPerPort(t *testing.T) {
	// 1 プロセスが同一ポートを IPv4/IPv6 の両方で LISTEN するなど、
	// scanner は同一ポートに複数エントリを返しうる。マージで潰さず
	// 全件保持し、それぞれの Address/PID が正しく返ることを確認する。
	regPath := filepath.Join(t.TempDir(), "registry.json")
	fake := func() ([]ports.Entry, error) {
		return []ports.Entry{
			{Port: 3000, Proto: "tcp", Address: "127.0.0.1", PID: 10, Process: "node"},
			{Port: 3000, Proto: "tcp", Address: "::1", PID: 10, Process: "node"},
		}, nil
	}
	s := New(regPath, fake)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/ports", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var views []PortView
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatal(err)
	}

	addrs := map[string]bool{}
	for _, v := range views {
		if v.Port == 3000 {
			addrs[v.Address] = true
		}
	}
	if !addrs["127.0.0.1"] || !addrs["::1"] {
		t.Fatalf("port 3000 の複数リスナーが保持されていない: %+v", views)
	}
}

func TestPutAndDeleteLabel(t *testing.T) {
	s, regPath := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/ports/8080/label",
		strings.NewReader(`{"name":"llama.cpp","note":"main"}`))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d body = %s", rec.Code, rec.Body.String())
	}
	r, _ := registry.Load(regPath)
	if r.Ports[8080].Name != "llama.cpp" {
		t.Fatalf("registry not saved: %+v", r.Ports)
	}

	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/ports/8080/label", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d", rec.Code)
	}
	r, _ = registry.Load(regPath)
	if _, ok := r.Ports[8080]; ok {
		t.Fatal("label not deleted")
	}
}

func TestPutLabelValidation(t *testing.T) {
	s, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/ports/8080/label", strings.NewReader(`{"name":""}`))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: status = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("PUT", "/api/ports/notaport/label", strings.NewReader(`{"name":"x"}`))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad port: status = %d, want 400", rec.Code)
	}
}

func TestServesEmbeddedUI(t *testing.T) {
	s, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "127xx") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
