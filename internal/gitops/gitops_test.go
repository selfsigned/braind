package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevertCreatesNewCommit(t *testing.T) {
	tmpDir := t.TempDir()

	g, err := Init(tmpDir, "test", "test@test.com")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	hash1, err := g.Commit("first commit", []FileChange{{Path: "test.txt", Action: "create"}})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	hash2, err := g.Commit("second commit", []FileChange{{Path: "test.txt", Action: "modify"}})
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}

	revertHash, err := g.Revert(nil, hash2)
	if err != nil {
		t.Fatalf("Revert: %v", err)
	}

	if revertHash == "" {
		t.Error("Revert should return new commit hash")
	}

	if revertHash == hash1 || revertHash == hash2 {
		t.Error("Revert should create a NEW commit, not reuse existing hash")
	}

	lastCommit, err := g.LastCommit()
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}

	if lastCommit != revertHash {
		t.Errorf("LastCommit = %s, want %s", lastCommit, revertHash)
	}
}

func TestRevertWithEmptyHashUsesHEAD(t *testing.T) {
	tmpDir := t.TempDir()

	g, err := Init(tmpDir, "test", "test@test.com")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err = g.Commit("first commit", []FileChange{{Path: "test.txt", Action: "create"}})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	hash2, err := g.Commit("second commit", []FileChange{{Path: "test.txt", Action: "modify"}})
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}

	revertHash, err := g.Revert(nil, "")
	if err != nil {
		t.Fatalf("Revert with empty hash: %v", err)
	}

	if revertHash == hash2 {
		t.Error("Revert should create a new commit, not reuse HEAD hash")
	}

	lastCommit, err := g.LastCommit()
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}

	if lastCommit != revertHash {
		t.Errorf("LastCommit = %s, want %s", lastCommit, revertHash)
	}
}

func TestRevertDoesNotLoseHistory(t *testing.T) {
	tmpDir := t.TempDir()

	g, err := Init(tmpDir, "test", "test@test.com")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err = g.Commit("first commit", []FileChange{{Path: "test.txt", Action: "create"}})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err = g.Commit("second commit", []FileChange{{Path: "test.txt", Action: "modify"}})
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}

	_, err = g.Revert(nil, "")
	if err != nil {
		t.Fatalf("Revert: %v", err)
	}

	commits, err := g.Log()
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	if len(commits) < 3 {
		t.Errorf("Log should have at least 3 commits (initial, second, revert), got %d", len(commits))
	}
}
