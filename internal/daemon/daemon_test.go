package daemon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/braind/braind/internal/config"
)

func TestServeMux(t *testing.T) {
	cfg := &config.Config{
		Vaults: map[string]config.VaultConfig{
			"test-vault": {Path: "/tmp/test"},
		},
	}
	s := New(cfg)

	tests := []struct {
		method, path string
		wantCode     int
	}{
		{"POST", "/v1/test-vault/ask", http.StatusOK},
		{"GET", "/v1/test-vault/sources", http.StatusOK},
		{"GET", "/v1/test-vault/todos", http.StatusOK},
		{"POST", "/v1/test-vault/run/journal_prefill", http.StatusOK},
		{"POST", "/v1/test-vault/undo", http.StatusOK},
		{"GET", "/status", http.StatusOK},
		{"POST", "/v1/unknown-vault/ask", http.StatusNotFound},
		{"GET", "/v1/test-vault/unknown", http.StatusNotFound},
	}

	for _, tt := range tests {
		var body *strings.Reader
		if tt.method == "POST" {
			body = strings.NewReader(`{"prompt": "test"}`)
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(tt.method, tt.path, body)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		if w.Code != tt.wantCode {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, w.Code, tt.wantCode)
		}
	}
}

func TestAskRequest(t *testing.T) {
	cfg := &config.Config{
		Vaults: map[string]config.VaultConfig{
			"test": {Path: "/tmp/test"},
		},
	}
	s := New(cfg)

	body := strings.NewReader(`{"prompt": "hello"}`)
	req := httptest.NewRequest("POST", "/v1/test/ask", body)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if !strings.Contains(w.Body.String(), "hello") {
		t.Errorf("body = %q, want containing %q", w.Body.String(), "hello")
	}
}

func TestVaultNotFound(t *testing.T) {
	cfg := &config.Config{
		Vaults: map[string]config.VaultConfig{},
	}
	s := New(cfg)

	req := httptest.NewRequest("GET", "/v1/nonexistent/sources", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
