package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Changelog struct {
	path string
}

func Open(vaultPath string) (*Changelog, error) {
	path := filepath.Join(vaultPath, "changelog.org")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("changelog: open: %w", err)
	}
	f.Close()
	return &Changelog{path: path}, nil
}

func (c *Changelog) Append(entry *Entry) error {
	f, err := os.OpenFile(c.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("changelog: open: %w", err)
	}
	defer f.Close()

	header := fmt.Sprintf("* %s\n\n", time.Now().Format("2006-01-02"))

	action := fmt.Sprintf("** %s — %s\n", time.Now().Format("15:04"), entry.Action)
	if entry.Scheduled {
		action = fmt.Sprintf("** %s — %s (scheduled)\n", time.Now().Format("15:04"), entry.Action)
	}

	model := fmt.Sprintf("   - Model: %s\n", entry.Model)
	tools := fmt.Sprintf("   - Tools used: %s\n", entry.Tools)

	var changes string
	for _, change := range entry.Changes {
		changes += fmt.Sprintf("     - %s\n", change)
	}

	rationale := fmt.Sprintf("   - Rationale: %s\n", entry.Rationale)

	content := header + action + model + tools + changes + rationale + "\n"
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("changelog: write: %w", err)
	}

	return nil
}

type Entry struct {
	Action    string
	Model     string
	Tools     string
	Changes   []string
	Rationale string
	Scheduled bool
}
