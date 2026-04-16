package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braind/braind/internal/interfaces"
)

func TestDefaultJournalFilenameFormat(t *testing.T) {
	jt := &JournalTool{}

	vaultDir := t.TempDir()
	journalDir := filepath.Join(vaultDir, "journals")
	os.MkdirAll(journalDir, 0755)

	wantFilename := "2025_04_15.md"

	_, err := jt.Execute(nil, interfaces.ToolRequest{
		VaultPath: vaultDir,
		Params:    []byte(`{"date": "2025-04-15", "content": "test entry"}`),
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

func TestJournalFilenameNoWeekdaySuffix(t *testing.T) {
	jt := &JournalTool{}

	vaultDir := t.TempDir()
	journalDir := filepath.Join(vaultDir, "journals")
	os.MkdirAll(journalDir, 0755)

	_, err := jt.Execute(nil, interfaces.ToolRequest{
		VaultPath: vaultDir,
		Params:    []byte(`{"date": "2025-04-15", "content": "test"}`),
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
		if strings.HasPrefix(name, "2025") {
			weekdaySuffixes := []string{"_Tuesday", "_Wednesday", "_Thursday", "_Friday", "_Saturday", "_Sunday", "_Monday", "_Tuesday"}
			for _, suffix := range weekdaySuffixes {
				if strings.Contains(name, suffix) {
					t.Errorf("Journal filename should NOT contain weekday suffix, got: %s", name)
				}
			}
		}
	}
}
