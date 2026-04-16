package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg := `
daemon:
  socket_dir: "/tmp/braind"
  protocol: "http+json-over-uds"
  log_level: "info"

llm:
  default:
    provider: "ollama"
    model: "gemma3:4b"
    base_url: "http://localhost:11434"
    timeout: "30s"
    max_tokens: 2048

vaults:
  test:
    path: "~/test-vault"
    git:
      auto_commit: true
      commit_name: "braind"
      commit_email: "braind@localhost"
    changelog:
      enabled: true
    tools:
      - name: "journal"
      - name: "todo"
    providers: []
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Daemon.SocketDir != "/tmp/braind" {
		t.Errorf("SocketDir = %q, want %q", loaded.Daemon.SocketDir, "/tmp/braind")
	}
	if loaded.LLM.Default.Model != "gemma3:4b" {
		t.Errorf("Model = %q, want %q", loaded.LLM.Default.Model, "gemma3:4b")
	}

	vault, ok := loaded.Vault("test")
	if !ok {
		t.Fatal("vault test not found")
	}
	if vault.Path != filepath.Join(os.Getenv("HOME"), "test-vault") {
		t.Errorf("Path = %q, want %q", vault.Path, filepath.Join(os.Getenv("HOME"), "test-vault"))
	}
	if len(vault.Tools) != 2 {
		t.Errorf("Tools count = %d, want 2", len(vault.Tools))
	}
	if vault.Tools[0].Name != "journal" {
		t.Errorf("Tool[0] = %q, want %q", vault.Tools[0].Name, "journal")
	}
}

func TestLoadRequiresModel(t *testing.T) {
	cfg := `
daemon:
  socket_dir: "/tmp/braind"

llm:
  default:
    base_url: "http://localhost:11434"
vaults: {}
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load: expected error for missing model")
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	data := expandEnv([]byte("foo ${TEST_VAR} bar"))
	if string(data) != "foo test-value bar" {
		t.Errorf("expandEnv = %q, want %q", string(data), "foo test-value bar")
	}
}
