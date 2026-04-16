package journal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/braind/braind/internal/interfaces"
)

type JournalTool struct{}

func New() *JournalTool { return &JournalTool{} }

func (t *JournalTool) Name() string { return "journal" }
func (t *JournalTool) Description() string {
	return "Write/edit Logseq journal entries and pages. Creates markdown files with proper [[backlinks]], property drawers, and timestamps."
}

func (t *JournalTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"date": {
				"type": "string",
				"description": "Date for journal entry in YYYY-MM-DD format. Defaults to today."
			},
			"content": {
				"type": "string",
				"description": "Journal content to write. Supports Logseq markdown."
			},
			"page": {
				"type": "string",
				"description": "Page name to write/update instead of journal entry."
			}
		}
	}`)
}

type Params struct {
	Date    string `json:"date"`
	Content string `json:"content"`
	Page    string `json:"page"`
}

func (t *JournalTool) Execute(ctx context.Context, req interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	var p Params
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, fmt.Errorf("journal: parse params: %w", err)
	}

	dateStr := p.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("journal: parse date: %w", err)
	}

	var filePath string
	var fileAction string
	if p.Page != "" {
		filePath = filepath.Join(req.VaultPath, "pages", p.Page+".md")
		fileAction = "modify"
	} else {
		filename := fmt.Sprintf("%s.md", date.Format("2006_01_02"))
		filePath = filepath.Join(req.VaultPath, "journals", filename)
		fileAction = "create"
	}

	existing := ""
	if data, err := os.ReadFile(filePath); err == nil {
		existing = string(data)
		fileAction = "modify"
	}

	var content string
	if existing != "" && p.Page == "" {
		content = existing + "\n\n" + formatTimestamp() + " " + p.Content
	} else if existing != "" {
		content = existing + "\n\n" + p.Content
	} else if p.Page != "" {
		content = fmt.Sprintf("# %s\n\n%s\n", p.Page, p.Content)
	} else {
		content = formatJournalEntry(date, p.Content)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("journal: create dir: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("journal: write file: %w", err)
	}

	return &interfaces.ToolResponse{
		Result: fmt.Sprintf("wrote %s: %s", fileAction, filepath.Base(filePath)),
		Changes: []interfaces.FileChange{
			{Path: filePath, Action: fileAction, Diff: ""},
		},
	}, nil
}

func formatJournalEntry(date time.Time, content string) string {
	header := fmt.Sprintf("- ## [[%s]]\n\n", date.Format("2006-01-02"))
	timestamp := formatTimestamp()
	return header + "- **" + timestamp + "** " + content + "\n"
}

func formatTimestamp() string {
	return time.Now().Format("15:04")
}
