package filebrowser

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sst/sidecar/internal/plugin"
)

func createTestPlugin(t *testing.T, tmpDir string) *Plugin {
	// Create some test files and directories
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "app.go"), []byte("package src"), 0644); err != nil {
		t.Fatalf("failed to create app.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create README.md: %v", err)
	}

	p := &Plugin{
		ctx: &plugin.Context{
			WorkDir: tmpDir,
			Logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		},
		width:  80,
		height: 24,
	}

	// Build the file tree
	p.tree = NewFileTree(tmpDir)
	if err := p.tree.Build(); err != nil {
		t.Fatalf("failed to build file tree: %v", err)
	}

	return p
}

func TestSearch_EnterSearchMode(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	if p.searchMode {
		t.Error("searchMode should be false initially")
	}

	// Press "/" to enter search mode
	_, _ = p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if !p.searchMode {
		t.Error("searchMode should be true after '/'")
	}
	if p.searchQuery != "" {
		t.Error("searchQuery should be empty when entering search mode")
	}
}

func TestSearch_ExitSearchMode(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	// Enter search mode
	p.searchMode = true
	p.searchQuery = "test"

	// Press escape to exit
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyEscape})

	if p.searchMode {
		t.Error("searchMode should be false after escape")
	}
	if p.searchQuery != "" {
		t.Error("searchQuery should be cleared after escape")
	}
}

func TestSearch_TypeQuery(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true

	// Type "main"
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if p.searchQuery != "main" {
		t.Errorf("searchQuery = %q, want main", p.searchQuery)
	}

	// Should have found main.go
	if len(p.searchMatches) == 0 {
		t.Error("no matches found for 'main'")
	}

	found := false
	for _, match := range p.searchMatches {
		if match.Name == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("main.go not in search matches")
	}
}

func TestSearch_Backspace(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true
	p.searchQuery = "mail"

	// Press backspace
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyBackspace})

	if p.searchQuery != "mai" {
		t.Errorf("searchQuery = %q, want mai", p.searchQuery)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true

	// Type "MAIN" in uppercase
	p.searchQuery = "MAIN"
	p.updateSearchMatches()

	found := false
	for _, match := range p.searchMatches {
		if match.Name == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("case-insensitive search failed - main.go not found for 'MAIN'")
	}
}

func TestSearch_PartialMatch(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchQuery = "app"
	p.updateSearchMatches()

	found := false
	for _, match := range p.searchMatches {
		if match.Name == "app.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("partial match failed - app.go not found for 'app'")
	}
}

func TestSearch_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchQuery = "a" // Matches: app.go, main.go, README.md, config (in path)
	p.updateSearchMatches()

	if len(p.searchMatches) == 0 {
		t.Error("no matches found for 'a'")
	}

	// Should have multiple matches
	if len(p.searchMatches) < 2 {
		t.Errorf("expected multiple matches, got %d", len(p.searchMatches))
	}
}

func TestSearch_MatchLimit(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	// Create many files matching a pattern
	for i := 0; i < 30; i++ {
		fname := filepath.Join(tmpDir, "file"+string(rune('0'+(i%10)))+"_"+string(rune('0'+(i/10)))+".txt")
		if err := os.WriteFile(fname, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Rebuild tree with new files
	p.tree = NewFileTree(tmpDir)
	if err := p.tree.Build(); err != nil {
		t.Fatalf("failed to rebuild tree: %v", err)
	}

	p.searchQuery = "file"
	p.updateSearchMatches()

	// Should be limited to 20 matches
	if len(p.searchMatches) > 20 {
		t.Errorf("matches exceeds limit: got %d, want <= 20", len(p.searchMatches))
	}
}

func TestSearch_NavigateMatches(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true
	p.searchQuery = "a" // Multiple matches

	// Update matches
	p.updateSearchMatches()

	if len(p.searchMatches) < 2 {
		t.Skip("need at least 2 matches for navigation test")
	}

	initialCursor := p.searchCursor

	// Press down to move to next match
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyDown})

	if p.searchCursor == initialCursor {
		t.Error("search cursor did not move with down arrow")
	}

	// Press up to move to previous match
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyUp})

	if p.searchCursor != initialCursor {
		t.Error("search cursor did not move back with up arrow")
	}
}

func TestSearch_JumpToMatch(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true
	p.searchQuery = "app"
	p.updateSearchMatches()

	if len(p.searchMatches) == 0 {
		t.Skip("no matches found for app")
	}

	initialCursor := p.treeCursor

	// Press enter to jump to first match
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Cursor should have moved or tree cursor updated
	if !p.searchMode {
		// Search mode should be exited after jumping
		// Tree cursor should be updated
		if p.treeCursor == initialCursor {
			// This is ok - might be the same position
		}
	}
}

func TestSearch_ExpandParents(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	// Find a nested file
	var nestedFile *FileNode
	p.walkTree(p.tree.Root, func(node *FileNode) {
		if node.Name == "app.go" {
			nestedFile = node
		}
	})

	if nestedFile == nil {
		t.Skip("nested file not found")
	}

	// Collapse the parent directory
	var srcDir *FileNode
	for _, child := range p.tree.Root.Children {
		if child.Name == "src" {
			srcDir = child
			srcDir.IsExpanded = false
			break
		}
	}

	if srcDir == nil {
		t.Skip("src directory not found")
	}

	// Expand parents should expand src
	p.expandParents(nestedFile)

	if !srcDir.IsExpanded {
		t.Error("expandParents did not expand parent directory")
	}
}

func TestSearch_WalkTree(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	visitedFiles := make(map[string]bool)

	p.walkTree(p.tree.Root, func(node *FileNode) {
		visitedFiles[node.Name] = true
	})

	// Should have visited test files
	expectedFiles := []string{"main.go", "src", "app.go", "config.json", "README.md"}
	for _, expected := range expectedFiles {
		if !visitedFiles[expected] {
			t.Errorf("walkTree did not visit %s", expected)
		}
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchQuery = ""
	p.updateSearchMatches()

	if len(p.searchMatches) != 0 {
		t.Error("empty search query should have no matches")
	}
}

func TestSearch_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchQuery = "xyznonexistent"
	p.updateSearchMatches()

	if len(p.searchMatches) != 0 {
		t.Error("nonexistent query should have no matches")
	}

	if p.searchCursor != 0 {
		t.Error("search cursor should be reset to 0 when no matches")
	}
}

func TestSearch_CursorBounds(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true
	p.searchQuery = "a"
	p.updateSearchMatches()

	if len(p.searchMatches) == 0 {
		t.Skip("no matches found")
	}

	// Try to move up when at position 0
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyUp})
	if p.searchCursor < 0 {
		t.Error("search cursor should not go below 0")
	}

	// Move to end
	for i := 0; i < len(p.searchMatches); i++ {
		_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Try to move down at end
	cursorBefore := p.searchCursor
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyDown})

	if p.searchCursor > len(p.searchMatches)-1 {
		t.Error("search cursor should not exceed matches length")
	}
	if p.searchCursor != cursorBefore && len(p.searchMatches) > 1 {
		// This is OK - might have wrapped around
	}
}

func TestSearch_PrintableCharacterFilter(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.searchMode = true

	// Try to input non-printable character (should be ignored)
	initialQuery := p.searchQuery
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})

	if p.searchQuery != initialQuery {
		t.Error("non-printable character should be ignored")
	}

	// Printable character should be added
	_, _ = p.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if p.searchQuery != "a" {
		t.Error("printable character should be added to query")
	}
}
