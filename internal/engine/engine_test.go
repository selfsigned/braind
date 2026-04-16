package engine

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/braind/braind/internal/config"
	"github.com/braind/braind/internal/interfaces"
)

func TestAllowlistEnforcement(t *testing.T) {
	cfg := &config.Config{
		Vaults: map[string]config.VaultConfig{
			"test": {
				Path: "/tmp/test",
				Tools: []config.ToolConfig{
					{Name: "journal"},
					{Name: "todo"},
				},
			},
		},
	}

	registry := NewToolRegistry()
	registry.Register(&mockTool{name: "journal"})
	registry.Register(&mockTool{name: "todo"})
	registry.Register(&mockTool{name: "search"})
	registry.Register(&mockTool{name: "weather"})

	allowed := getAllowedTools(cfg.Vaults["test"].Tools)

	for _, tool := range registry.tools {
		name := tool.Name()
		allowed := isToolAllowed(name, allowed)

		if name == "journal" && !allowed {
			t.Errorf("journal should be allowed")
		}
		if name == "weather" && allowed {
			t.Errorf("weather should NOT be allowed")
		}
	}
}

func TestToolNotInAllowlist(t *testing.T) {
	allowed := []string{"journal", "todo"}

	if isToolAllowed("journal", allowed) != true {
		t.Error("journal should be allowed")
	}
	if isToolAllowed("todo", allowed) != true {
		t.Error("todo should be allowed")
	}
	if isToolAllowed("search", allowed) != false {
		t.Error("search should NOT be allowed")
	}
	if isToolAllowed("weather", allowed) != false {
		t.Error("weather should NOT be allowed")
	}
}

func isToolAllowed(name string, allowed []string) bool {
	for _, a := range allowed {
		if a == name {
			return true
		}
	}
	return false
}

func getAllowedTools(tools []config.ToolConfig) []string {
	var names []string
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

type mockTool struct {
	name string
}

func (t *mockTool) Name() string                { return t.name }
func (t *mockTool) Description() string         { return t.name + " tool" }
func (t *mockTool) Parameters() json.RawMessage { return nil }
func (t *mockTool) Execute(ctx context.Context, req interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	return nil, nil
}

type ToolRegistry struct {
	tools map[string]interfaces.Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]interfaces.Tool)}
}

func (r *ToolRegistry) Register(t interfaces.Tool) {
	r.tools[t.Name()] = t
}

func (r *ToolRegistry) Get(name string) (interfaces.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) ValidateTool(name string, allowed []string) error {
	if !isToolAllowed(name, allowed) {
		return &ToolNotAllowedError{Tool: name}
	}
	return nil
}

type ToolNotAllowedError struct {
	Tool string
}

func (e *ToolNotAllowedError) Error() string {
	return "tool not allowed: " + e.Tool
}
