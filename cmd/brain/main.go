package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/braind/braind/internal/config"
)

var (
	configPath  string
	activeVault string
)

func main() {
	home, _ := os.UserHomeDir()
	cfgDir := filepath.Join(home, ".config", "braind")
	os.MkdirAll(cfgDir, 0755)
	configPath = filepath.Join(cfgDir, "config.yaml")
	activeVaultFile := filepath.Join(cfgDir, "active_vault")

	if data, _ := os.ReadFile(activeVaultFile); len(data) > 0 {
		activeVault = strings.TrimSpace(string(data))
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage: brain <command>")
		fmt.Println("Commands: vault, ask, journal, todos, sources, log, diff, undo, status, run, provider")
		os.Exit(1)
	}

	switch args[0] {
	case "vault":
		doVault(args[1:])
	case "ask":
		doAsk(args[1:])
	case "journal":
		doJournal(args[1:])
	case "todos":
		doTodos(args[1:])
	case "sources":
		doSources(args[1:])
	case "log":
		doLog(args[1:])
	case "diff":
		doDiff(args[1:])
	case "undo":
		doUndo(args[1:])
	case "status":
		doStatus(args[1:])
	case "run":
		doRun(args[1:])
	case "provider":
		doProvider(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func getVault(cfg *config.Config, name string) string {
	if name != "" {
		return name
	}
	if activeVault != "" {
		return activeVault
	}
	for name := range cfg.Vaults {
		return name
	}
	fmt.Fprintf(os.Stderr, "no vault specified and no vaults configured\n")
	os.Exit(1)
	return ""
}

func socketPath(vault string) string {
	return fmt.Sprintf("/tmp/braind/%s.sock", vault)
}

func doVault(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: brain vault <list|switch>")
		os.Exit(1)
	}

	cfg := loadConfig()

	switch args[0] {
	case "list":
		for name := range cfg.Vaults {
			mark := "  "
			if name == activeVault {
				mark = "* "
			}
			fmt.Printf("%s%s\n", mark, name)
		}
	case "switch":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: brain vault switch <name>\n")
			os.Exit(1)
		}
		name := args[1]
		if _, ok := cfg.Vaults[name]; !ok {
			fmt.Fprintf(os.Stderr, "vault %q not found\n", name)
			os.Exit(1)
		}
		home, _ := os.UserHomeDir()
		cfgDir := filepath.Join(home, ".config", "braind")
		activeVaultFile := filepath.Join(cfgDir, "active_vault")
		os.WriteFile(activeVaultFile, []byte(name), 0644)
		activeVault = name
		fmt.Printf("Switched to vault: %s\n", name)
	default:
		fmt.Fprintf(os.Stderr, "unknown vault command: %s\n", args[0])
		os.Exit(1)
	}
}

func doAsk(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: brain ask <prompt>\n")
		os.Exit(1)
	}

	cfg := loadConfig()
	vault := getVault(cfg, "")
	prompt := strings.Join(args, " ")

	sockPath := socketPath(vault)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	reqBody := map[string]string{"prompt": prompt}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "http://localhost/v1/"+vault+"/ask", strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "server error: %s\n", respBody)
		os.Exit(1)
	}

	var result map[string]string
	json.Unmarshal(respBody, &result)
	fmt.Println(result["response"])
}

func doJournal(args []string) {
	fmt.Println("journal: stub - use brain ask to prefill journal")
}

func doTodos(args []string) {
	cfg := loadConfig()
	vault := getVault(cfg, "")

	sockPath := socketPath(vault)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	req, err := http.NewRequest("GET", "http://localhost/v1/"+vault+"/todos", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
}

func doSources(args []string) {
	cfg := loadConfig()
	vault := getVault(cfg, "")

	sockPath := socketPath(vault)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	req, err := http.NewRequest("GET", "http://localhost/v1/"+vault+"/sources", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "server error: %s\n", respBody)
		os.Exit(1)
	}

	var result struct {
		Sources []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"sources"`
	}
	json.Unmarshal(respBody, &result)
	for _, s := range result.Sources {
		fmt.Printf("%s\t%s\t(%s)\n", s.ID, s.Name, s.Type)
	}
}

func doLog(args []string) {
	fmt.Println("log: stub - would show changelog entries")
}

func doDiff(args []string) {
	fmt.Println("diff: stub - would show git diff")
}

func doUndo(args []string) {
	cfg := loadConfig()
	vault := getVault(cfg, "")

	var commit string
	if len(args) > 0 {
		commit = args[0]
	}

	sockPath := socketPath(vault)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	reqBody := map[string]string{}
	if commit != "" {
		reqBody["commit"] = commit
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "http://localhost/v1/"+vault+"/undo", strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
}

func doStatus(args []string) {
	cfg := loadConfig()

	sockPath := socketPath(getVault(cfg, ""))
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	req, err := http.NewRequest("GET", "http://localhost/status", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon not running or socket not accessible\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "server error: %s\n", respBody)
		os.Exit(1)
	}

	fmt.Println(string(respBody))
}

func doRun(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: brain run <job>\n")
		os.Exit(1)
	}

	cfg := loadConfig()
	vault := getVault(cfg, "")
	job := args[0]

	sockPath := socketPath(vault)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}

	reqBody := map[string]string{"job": job}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "http://localhost/v1/"+vault+"/run/"+job, strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
}

func doProvider(args []string) {
	fmt.Println("provider: stub - would run provider ingestion")
}
