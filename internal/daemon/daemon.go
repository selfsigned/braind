package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/braind/braind/internal/config"
)

type Server struct {
	config *config.Config
	mux    *http.ServeMux
	socks  map[string]string
	mu     sync.RWMutex
}

func New(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
		socks:  make(map[string]string),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("POST /v1/{vault}/ask", s.handleAsk)
	s.mux.HandleFunc("GET /v1/{vault}/sources", s.handleSources)
	s.mux.HandleFunc("GET /v1/{vault}/todos", s.handleTodos)
	s.mux.HandleFunc("POST /v1/{vault}/run/{job}", s.handleRun)
	s.mux.HandleFunc("POST /v1/{vault}/undo", s.handleUndo)
	s.mux.HandleFunc("GET /status", s.handleStatus)
}

type AskRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
}

type AskResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources,omitempty"`
}

type SourcesResponse struct {
	Sources []Source `json:"sources"`
}

type Source struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type TodosResponse struct {
	Todos []Todo `json:"todos"`
}

type Todo struct {
	Content   string `json:"content"`
	State     string `json:"state"`
	Path      string `json:"path"`
	Scheduled string `json:"scheduled,omitempty"`
}

type RunRequest struct {
	Job   string `json:"job"`
	Model string `json:"model,omitempty"`
}

type RunResponse struct {
	Job   string `json:"job"`
	Done  bool   `json:"done"`
	Notes string `json:"notes,omitempty"`
}

type UndoRequest struct {
	Commit string `json:"commit,omitempty"`
}

type UndoResponse struct {
	Reverted bool   `json:"reverted"`
	Commit   string `json:"commit"`
	Message  string `json:"message,omitempty"`
}

type StatusResponse struct {
	Status  string   `json:"status"`
	Vaults  []string `json:"vaults"`
	Version string   `json:"version"`
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	vault := r.PathValue("vault")
	if !s.validVault(w, vault) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req AskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("parse body: %v", err), http.StatusBadRequest)
		return
	}

	resp := AskResponse{
		Response: fmt.Sprintf("ask %q on vault %q (stub)", req.Prompt, vault),
	}
	s.writeJSON(w, resp)
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	vault := r.PathValue("vault")
	if !s.validVault(w, vault) {
		return
	}

	resp := SourcesResponse{
		Sources: []Source{
			{ID: "rss-n-gate", Name: "n-gate.com", Type: "rss"},
			{ID: "rss-lwn", Name: "lwn.net", Type: "rss"},
		},
	}
	s.writeJSON(w, resp)
}

func (s *Server) handleTodos(w http.ResponseWriter, r *http.Request) {
	vault := r.PathValue("vault")
	if !s.validVault(w, vault) {
		return
	}

	resp := TodosResponse{
		Todos: []Todo{
			{Content: "Pay taxes", State: "TODO", Path: "journals/2025_04_15.md"},
			{Content: "Fix bike chain", State: "DOING", Path: "pages/Bike Repair.md"},
		},
	}
	s.writeJSON(w, resp)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	vault := r.PathValue("vault")
	if !s.validVault(w, vault) {
		return
	}

	job := r.PathValue("job")
	if job == "" {
		http.Error(w, "job name required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	resp := RunResponse{
		Job:   job,
		Done:  true,
		Notes: fmt.Sprintf("ran job %q on vault %q (stub)", job, vault),
	}
	s.writeJSON(w, resp)
}

func (s *Server) handleUndo(w http.ResponseWriter, r *http.Request) {
	vault := r.PathValue("vault")
	if !s.validVault(w, vault) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req UndoRequest
	if len(body) > 0 {
		json.Unmarshal(body, &req)
	}

	resp := UndoResponse{
		Reverted: true,
		Commit:   req.Commit,
		Message:  fmt.Sprintf("reverted commit %q on vault %q (stub)", req.Commit, vault),
	}
	s.writeJSON(w, resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	vaults := make([]string, 0, len(s.socks))
	for v := range s.socks {
		vaults = append(vaults, v)
	}
	s.mu.RUnlock()

	resp := StatusResponse{
		Status:  "running",
		Vaults:  vaults,
		Version: "0.1.0",
	}
	s.writeJSON(w, resp)
}

func (s *Server) validVault(w http.ResponseWriter, vault string) bool {
	s.mu.RLock()
	_, ok := s.socks[vault]
	s.mu.RUnlock()

	if ok {
		return true
	}

	s.mu.RLock()
	_, ok = s.config.Vaults[vault]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("vault %q not found", vault), http.StatusNotFound)
		return false
	}

	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) RegisterVault(vaultName, sockPath string) {
	s.mu.Lock()
	s.socks[vaultName] = sockPath
	s.mu.Unlock()
}

func (s *Server) Serve(socketPath string) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("daemon: create socket dir: %w", err)
	}

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("daemon: remove old socket: %w", err)
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("daemon: listen unix: %w", err)
	}

	if err := os.Chmod(socketPath, 0777); err != nil {
		return fmt.Errorf("daemon: chmod socket: %w", err)
	}

	server := &http.Server{
		Handler:  s.mux,
		ErrorLog: nil,
	}

	go server.Serve(ln)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}
