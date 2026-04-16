package todo

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/braind/braind/internal/interfaces"
)

type TodoTool struct{}

func New() *TodoTool { return &TodoTool{} }

func (t *TodoTool) Name() string { return "todo" }
func (t *TodoTool) Description() string {
	return "Manage TODOs in Logseq format (TODO, DOING, DONE, LATER, NOW). Scans existing entries for state changes."
}

func (t *TodoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["add", "done", "later", "now", "cancel"],
				"description": "Action to perform"
			},
			"content": {
				"type": "string",
				"description": "TODO content text"
			},
			"path": {
				"type": "string",
				"description": "File path to modify (defaults to today's journal)"
			},
			"scheduled": {
				"type": "string",
				"description": "Scheduled date in YYYY-MM-DD format"
			}
		}
	}`)
}

type Params struct {
	Action    string `json:"action"`
	Content   string `json:"content"`
	Path      string `json:"path"`
	Scheduled string `json:"scheduled"`
}

var todoLineRegex = regexp.MustCompile(`^\s*(- |\s*)- (TODO|DOING|DONE|LATER|NOW)\s+(.+)`)

func (t *TodoTool) Execute(ctx context.Context, req interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	var p Params
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, fmt.Errorf("todo: parse params: %w", err)
	}

	if p.Path == "" {
		filename := fmt.Sprintf("%s.md", time.Now().Format("2006_01_02"))
		p.Path = filepath.Join(req.VaultPath, "journals", filename)
	}

	dir := filepath.Dir(p.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("todo: create dir: %w", err)
	}

	existing := ""
	if data, err := os.ReadFile(p.Path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("todo: read file: %w", err)
	}

	var newContent string
	var changes []interfaces.FileChange

	switch p.Action {
	case "add":
		newContent = existing + formatTodo(p.Content, "TODO", p.Scheduled)
		changes = append(changes, interfaces.FileChange{Path: p.Path, Action: "modify"})
	case "done":
		newContent, changes = replaceTodoState(existing, p.Path, p.Content, "DONE")
	case "later":
		newContent, changes = replaceTodoState(existing, p.Path, p.Content, "LATER")
	case "now":
		newContent, changes = replaceTodoState(existing, p.Path, p.Content, "NOW")
	case "cancel":
		newContent, changes = removeTodo(existing, p.Path, p.Content)
	default:
		return nil, fmt.Errorf("todo: unknown action %q", p.Action)
	}

	if newContent == existing {
		newContent += formatTodo(p.Content, strings.ToUpper(p.Action), p.Scheduled)
		changes = append(changes, interfaces.FileChange{Path: p.Path, Action: "modify"})
	}

	if err := os.WriteFile(p.Path, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("todo: write file: %w", err)
	}

	return &interfaces.ToolResponse{
		Result:  fmt.Sprintf("%sd TODO: %s", p.Action, p.Content),
		Changes: changes,
	}, nil
}

func formatTodo(content, state, scheduled string) string {
	line := fmt.Sprintf("- %s %s\n", state, content)
	if scheduled != "" {
		line = fmt.Sprintf("- %s %s\n  SCHEDULED: <%s>\n", state, content, scheduled)
	}
	return line
}

func replaceTodoState(content, path, search, newState string) (string, []interfaces.FileChange) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result strings.Builder
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		matches := todoLineRegex.FindStringSubmatch(line)
		if matches != nil && strings.Contains(matches[3], search) {
			state := matches[2]
			if newState == "DONE" {
				result.WriteString(strings.Replace(line, state, "DONE", 1) + "\n")
			} else {
				newLine := strings.Replace(line, state, newState, 1)
				result.WriteString(newLine + "\n")
			}
			found = true
		} else {
			result.WriteString(line + "\n")
		}
	}

	changes := []interfaces.FileChange{{Path: path, Action: "modify"}}
	if !found {
		result.WriteString(formatTodo(search, newState, ""))
	}
	return result.String(), changes
}

func removeTodo(content, path, search string) (string, []interfaces.FileChange) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result strings.Builder
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		matches := todoLineRegex.FindStringSubmatch(line)
		if matches != nil && strings.Contains(matches[3], search) {
			found = true
			continue
		}
		result.WriteString(line + "\n")
	}

	changes := []interfaces.FileChange{{Path: path, Action: "modify"}}
	if !found {
		return content, changes
	}
	return result.String(), changes
}
