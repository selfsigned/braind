package todo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braind/braind/internal/interfaces"
)

func TestDefaultTodoJournalFilenameFormat(t *testing.T) {
	tt := &TodoTool{}

	vaultDir := t.TempDir()
	journalDir := filepath.Join(vaultDir, "journals")
	os.MkdirAll(journalDir, 0755)

	wantFilename := time.Now().Format("2006_01_02") + ".md"

	_, err := tt.Execute(nil, interfaces.ToolRequest{
		VaultPath: vaultDir,
		Params:    []byte(`{"action": "add", "content": "test task"}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	files, err := os.ReadDir(journalDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	var found bool
	for _, f := range files {
		if f.Name() == wantFilename {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected journal file %s, files found: %v", wantFilename, files)
	}
}

func TestTodoJournalFilenameNoWeekdaySuffix(t *testing.T) {
	tt := &TodoTool{}

	vaultDir := t.TempDir()
	journalDir := filepath.Join(vaultDir, "journals")
	os.MkdirAll(journalDir, 0755)

	_, err := tt.Execute(nil, interfaces.ToolRequest{
		VaultPath: vaultDir,
		Params:    []byte(`{"action": "add", "content": "test task"}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	files, err := os.ReadDir(journalDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, "2025") || strings.HasPrefix(name, "2026") {
			weekdaySuffixes := []string{"_Tuesday", "_Wednesday", "_Thursday", "_Friday", "_Saturday", "_Sunday", "_Monday", "_Tuesday"}
			for _, suffix := range weekdaySuffixes {
				if strings.Contains(name, suffix) {
					t.Errorf("Journal filename should NOT contain weekday suffix, got: %s", name)
				}
			}
		}
	}
}
