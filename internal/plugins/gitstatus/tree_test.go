package gitstatus

import (
	"os"
	"testing"
)

func TestNewFileTree(t *testing.T) {
	tree := NewFileTree("/tmp/test")
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if tree.workDir != "/tmp/test" {
		t.Errorf("expected workDir /tmp/test, got %s", tree.workDir)
	}
}

func TestFileTreeRefresh(t *testing.T) {
	// Skip if not in a git repo
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("couldn't get working directory")
	}

	tree := NewFileTree(cwd)
	err = tree.Refresh()
	if err != nil {
		t.Skipf("not in git repo or git not available: %v", err)
	}

	// Should have loaded something (or be clean)
	t.Logf("summary: %s", tree.Summary())
	t.Logf("total: %d files", tree.TotalCount())
}

func TestFileTreeSummary(t *testing.T) {
	tree := &FileTree{}

	// Empty tree
	if s := tree.Summary(); s != "clean" {
		t.Errorf("expected 'clean', got %q", s)
	}

	// With staged
	tree.Staged = []*FileEntry{{Path: "a.txt"}}
	if s := tree.Summary(); s != "1 staged" {
		t.Errorf("expected '1 staged', got %q", s)
	}

	// With modified
	tree.Modified = []*FileEntry{{Path: "b.txt"}, {Path: "c.txt"}}
	if s := tree.Summary(); s != "1 staged, 2 modified" {
		t.Errorf("expected '1 staged, 2 modified', got %q", s)
	}

	// With untracked
	tree.Untracked = []*FileEntry{{Path: "d.txt"}}
	if s := tree.Summary(); s != "1 staged, 2 modified, 1 untracked" {
		t.Errorf("expected '1 staged, 2 modified, 1 untracked', got %q", s)
	}
}

func TestFileTreeAllEntries(t *testing.T) {
	tree := &FileTree{
		Staged:    []*FileEntry{{Path: "a.txt"}},
		Modified:  []*FileEntry{{Path: "b.txt"}},
		Untracked: []*FileEntry{{Path: "c.txt"}},
	}

	entries := tree.AllEntries()
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Check order: staged, modified, untracked
	if entries[0].Path != "a.txt" {
		t.Errorf("expected first entry to be a.txt, got %s", entries[0].Path)
	}
	if entries[1].Path != "b.txt" {
		t.Errorf("expected second entry to be b.txt, got %s", entries[1].Path)
	}
	if entries[2].Path != "c.txt" {
		t.Errorf("expected third entry to be c.txt, got %s", entries[2].Path)
	}
}

func TestParseOrdinaryEntry(t *testing.T) {
	tree := &FileTree{}

	// Modified in both index and worktree
	line := "1 MM N... 100644 100644 100644 abc123 def456 src/main.go"
	entry := tree.parseOrdinaryEntry(line)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Path != "src/main.go" {
		t.Errorf("expected path src/main.go, got %s", entry.Path)
	}
	if !entry.Staged {
		t.Error("expected Staged to be true")
	}
	if !entry.Unstaged {
		t.Error("expected Unstaged to be true")
	}

	// Only staged
	line = "1 M. N... 100644 100644 100644 abc123 def456 staged.go"
	entry = tree.parseOrdinaryEntry(line)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if !entry.Staged {
		t.Error("expected Staged to be true")
	}
	if entry.Unstaged {
		t.Error("expected Unstaged to be false")
	}

	// Only unstaged
	line = "1 .M N... 100644 100644 100644 abc123 def456 unstaged.go"
	entry = tree.parseOrdinaryEntry(line)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Staged {
		t.Error("expected Staged to be false")
	}
	if !entry.Unstaged {
		t.Error("expected Unstaged to be true")
	}
}

func TestDiffStats(t *testing.T) {
	ds := DiffStats{Additions: 10, Deletions: 5}
	if ds.Additions != 10 {
		t.Errorf("expected 10 additions, got %d", ds.Additions)
	}
	if ds.Deletions != 5 {
		t.Errorf("expected 5 deletions, got %d", ds.Deletions)
	}
}
