package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Daemon     DaemonConfig           `yaml:"daemon"`
	Logging    LoggingConfig          `yaml:"logging"`
	Limits     LimitsConfig           `yaml:"limits"`
	Embeddings EmbeddingsConfig       `yaml:"embeddings"`
	LLM        LLMConfig              `yaml:"llm"`
	Vaults     map[string]VaultConfig `yaml:"vaults"`
}

type DaemonConfig struct {
	SocketDir string `yaml:"socket_dir"`
	Protocol  string `yaml:"protocol"`
	LogLevel  string `yaml:"log_level"`
	PidFile   string `yaml:"pid_file"`
}

type LoggingConfig struct {
	Path     string `yaml:"path"`
	Format   string `yaml:"format"`
	RotateMB int    `yaml:"rotate_mb"`
	MaxFiles int    `yaml:"max_files"`
}

type LimitsConfig struct {
	Interactive LimitSettings `yaml:"interactive"`
	Maintenance LimitSettings `yaml:"maintenance"`
}

type LimitSettings struct {
	MaxTurns     int           `yaml:"max_turns"`
	MaxToolCalls int           `yaml:"max_tool_calls"`
	MaxWallTime  time.Duration `yaml:"max_wall_time"`
}

type EmbeddingsConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	Dimension int    `yaml:"dimension"`
	Metric    string `yaml:"metric"`
}

type LLMConfig struct {
	Default LLMModelConfig `yaml:"default"`
	Heavy   LLMModelConfig `yaml:"heavy"`
}

type LLMModelConfig struct {
	Provider  string        `yaml:"provider"`
	Model     string        `yaml:"model"`
	BaseURL   string        `yaml:"base_url"`
	Timeout   time.Duration `yaml:"timeout"`
	MaxTokens int           `yaml:"max_tokens"`
}

type VaultConfig struct {
	Path      string           `yaml:"path"`
	Git       GitConfig        `yaml:"git"`
	Changelog ChangelogConfig  `yaml:"changelog"`
	Tools     []ToolConfig     `yaml:"tools"`
	Providers []ProviderConfig `yaml:"providers"`
	Schedule  []ScheduleConfig `yaml:"schedule"`
}

type GitConfig struct {
	AutoCommit  bool   `yaml:"auto_commit"`
	CommitName  string `yaml:"commit_name"`
	CommitEmail string `yaml:"commit_email"`
}

type ChangelogConfig struct {
	Enabled bool `yaml:"enabled"`
}

type ToolConfig struct {
	Name   string          `yaml:"name"`
	Config json.RawMessage `yaml:"config"`
}

type ProviderConfig struct {
	Name   string          `yaml:"name"`
	TTL    string          `yaml:"ttl"`
	Config json.RawMessage `yaml:"config"`
}

type ScheduleConfig struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Job   string `yaml:"job"`
	Model string `yaml:"model"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	data = expandEnv(data)

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validate: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Daemon.SocketDir == "" {
		c.Daemon.SocketDir = "/tmp/braind"
	}
	if c.Daemon.Protocol == "" {
		c.Daemon.Protocol = "http+json-over-uds"
	}
	if c.Daemon.LogLevel == "" {
		c.Daemon.LogLevel = "info"
	}
	if c.LLM.Default.Model == "" {
		return fmt.Errorf("llm.default.model is required")
	}
	if c.LLM.Default.BaseURL == "" {
		c.LLM.Default.BaseURL = "http://localhost:11434"
	}
	if c.LLM.Default.Timeout == 0 {
		c.LLM.Default.Timeout = 30 * time.Second
	}
	if c.LLM.Default.MaxTokens == 0 {
		c.LLM.Default.MaxTokens = 2048
	}
	for name, vault := range c.Vaults {
		if vault.Path == "" {
			return fmt.Errorf("vault %q: path is required", name)
		}
		vault.Path = expandPath(vault.Path)
		c.Vaults[name] = vault
	}
	return nil
}

func (c *Config) Vault(name string) (*VaultConfig, bool) {
	v, ok := c.Vaults[name]
	return &v, ok
}

func expandEnv(data []byte) []byte {
	s := string(data)
	s = os.Expand(s, func(key string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		switch key {
		case "XDG_RUNTIME_DIR":
			if val := os.Getenv("XDG_RUNTIME_DIR"); val != "" {
				return val
			}
			return "/tmp"
		case "XDG_CONFIG_HOME":
			if val := os.Getenv("XDG_CONFIG_HOME"); val != "" {
				return val
			}
			return filepath.Join(os.Getenv("HOME"), ".config")
		case "XDG_DATA_HOME":
			if val := os.Getenv("XDG_DATA_HOME"); val != "" {
				return val
			}
			return filepath.Join(os.Getenv("HOME"), ".local", "share")
		case "HOME":
			if val := os.Getenv("HOME"); val != "" {
				return val
			}
			return "/tmp"
		default:
			return "${" + key + "}"
		}
	})
	return []byte(s)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return os.Expand(path, func(key string) string {
		if key == "HOME" {
			if home := os.Getenv("HOME"); home != "" {
				return home
			}
			return "/tmp"
		}
		return os.Getenv(key)
	})
}
