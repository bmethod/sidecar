package gitstatus

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sst/sidecar/internal/plugin"
	"github.com/sst/sidecar/internal/state"
)

const (
	pluginID   = "git-status"
	pluginName = "Git Status"
	pluginIcon = "G"
)

// ViewMode represents the current view state.
type ViewMode int

const (
	ViewModeStatus       ViewMode = iota // Current file list
	ViewModeHistory                      // Commit browser
	ViewModeCommitDetail                 // Single commit files
	ViewModeDiff                         // Enhanced diff view
	ViewModeCommit                       // Commit message editor
)

// Plugin implements the git status plugin.
type Plugin struct {
	ctx       *plugin.Context
	tree      *FileTree
	focused   bool
	cursor    int
	scrollOff int

	// View mode state machine
	viewMode ViewMode

	// Diff state
	showDiff       bool
	diffContent    string
	diffFile       string
	diffScroll     int
	diffRaw        string       // Raw diff before delta processing
	diffCommit     string       // Commit hash if viewing commit diff
	diffViewMode   DiffViewMode // Line or side-by-side
	diffHorizOff   int          // Horizontal scroll for side-by-side
	parsedDiff     *ParsedDiff  // Parsed diff for enhanced rendering

	// History state
	commits            []*Commit
	selectedCommit     *Commit
	historyCursor      int
	historyScroll      int
	commitDetailCursor int
	commitDetailScroll int

	// External tool integration
	externalTool *ExternalTool

	// View dimensions
	width  int
	height int

	// Watcher
	watcher *Watcher

	// Commit state
	commitMessage    textarea.Model
	commitError      string
	commitInProgress bool
}

// New creates a new git status plugin.
func New() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string { return pluginID }

// Name returns the plugin display name.
func (p *Plugin) Name() string { return pluginName }

// Icon returns the plugin icon character.
func (p *Plugin) Icon() string { return pluginIcon }

// Init initializes the plugin with context.
func (p *Plugin) Init(ctx *plugin.Context) error {
	// Check if we're in a git repository
	gitDir := filepath.Join(ctx.WorkDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return err // Not a git repo, silently degrade
	}

	p.ctx = ctx
	p.tree = NewFileTree(ctx.WorkDir)
	p.externalTool = NewExternalTool(ToolModeAuto)

	// Load saved diff view mode preference
	if state.GetGitDiffMode() == "side-by-side" {
		p.diffViewMode = DiffViewSideBySide
	} else {
		p.diffViewMode = DiffViewUnified
	}

	return nil
}

// Start begins plugin operation.
func (p *Plugin) Start() tea.Cmd {
	return tea.Batch(
		p.refresh(),
		p.startWatcher(),
	)
}

// Stop cleans up plugin resources.
func (p *Plugin) Stop() {
	if p.watcher != nil {
		p.watcher.Stop()
	}
}

// Update handles messages.
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch p.viewMode {
		case ViewModeStatus:
			return p.updateStatus(msg)
		case ViewModeHistory:
			return p.updateHistory(msg)
		case ViewModeCommitDetail:
			return p.updateCommitDetail(msg)
		case ViewModeDiff:
			return p.updateDiff(msg)
		case ViewModeCommit:
			return p.updateCommit(msg)
		}

	case RefreshMsg:
		return p, p.refresh()

	case WatchEventMsg:
		return p, p.refresh()

	case DiffLoadedMsg:
		p.diffContent = msg.Content
		p.diffRaw = msg.Raw
		// Parse diff for built-in rendering (when not using delta)
		if p.externalTool == nil || !p.externalTool.ShouldUseDelta() {
			p.parsedDiff, _ = ParseUnifiedDiff(msg.Raw)
		} else {
			p.parsedDiff = nil
		}
		return p, nil

	case HistoryLoadedMsg:
		p.commits = msg.Commits
		return p, nil

	case CommitDetailLoadedMsg:
		p.selectedCommit = msg.Commit
		return p, nil

	case CommitSuccessMsg:
		// Commit succeeded, return to status view and refresh
		p.viewMode = ViewModeStatus
		p.commitMessage.Reset()
		p.commitInProgress = false
		p.commitError = ""
		return p, p.refresh()

	case CommitErrorMsg:
		// Commit failed, show error and keep message for retry
		p.commitError = msg.Err.Error()
		p.commitInProgress = false
		return p, nil

	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
	}

	return p, nil
}

// updateStatus handles key events in the status view.
func (p *Plugin) updateStatus(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	entries := p.tree.AllEntries()

	switch msg.String() {
	case "j", "down":
		if p.cursor < len(entries)-1 {
			p.cursor++
			p.ensureCursorVisible()
		}

	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
			p.ensureCursorVisible()
		}

	case "g":
		p.cursor = 0
		p.scrollOff = 0

	case "G":
		if len(entries) > 0 {
			p.cursor = len(entries) - 1
			p.ensureCursorVisible()
		}

	case "s":
		if len(entries) > 0 && p.cursor < len(entries) {
			entry := entries[p.cursor]
			if !entry.Staged {
				stagedCount := len(p.tree.Staged)
				totalEntries := len(entries)

				if err := p.tree.StageFile(entry.Path); err == nil {
					// After staging, move cursor to first unstaged file position
					// New staged count will be stagedCount + 1
					newFirstUnstaged := stagedCount + 1
					if newFirstUnstaged < totalEntries {
						p.cursor = newFirstUnstaged
					} else {
						// All files are now staged, stay at last position
						p.cursor = totalEntries - 1
					}
					return p, p.refresh()
				}
			}
		}

	case "u":
		if len(entries) > 0 && p.cursor < len(entries) {
			entry := entries[p.cursor]
			if entry.Staged {
				if err := p.tree.UnstageFile(entry.Path); err == nil {
					return p, p.refresh()
				}
			}
		}

	case "d":
		if len(entries) > 0 && p.cursor < len(entries) {
			entry := entries[p.cursor]
			p.viewMode = ViewModeDiff
			p.diffFile = entry.Path
			p.diffCommit = ""
			p.diffScroll = 0
			return p, p.loadDiff(entry.Path, entry.Staged)
		}

	case "enter":
		if len(entries) > 0 && p.cursor < len(entries) {
			entry := entries[p.cursor]
			return p, p.openFile(entry.Path)
		}

	case "h":
		// Switch to history view
		p.viewMode = ViewModeHistory
		p.historyCursor = 0
		p.historyScroll = 0
		return p, p.loadHistory()

	case "r":
		return p, p.refresh()

	case "S":
		// Stage all files
		if err := p.tree.StageAll(); err == nil {
			return p, p.refresh()
		}

	case "c":
		// Enter commit mode only if staged files exist
		if p.tree.HasStagedFiles() {
			p.viewMode = ViewModeCommit
			p.initCommitTextarea()
			return p, nil
		}
	}

	return p, nil
}

// updateHistory handles key events in the history view.
func (p *Plugin) updateHistory(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		p.viewMode = ViewModeStatus
		p.commits = nil

	case "j", "down":
		if p.commits != nil && p.historyCursor < len(p.commits)-1 {
			p.historyCursor++
			p.ensureHistoryCursorVisible()
		}

	case "k", "up":
		if p.historyCursor > 0 {
			p.historyCursor--
			p.ensureHistoryCursorVisible()
		}

	case "g":
		p.historyCursor = 0
		p.historyScroll = 0

	case "G":
		if p.commits != nil && len(p.commits) > 0 {
			p.historyCursor = len(p.commits) - 1
			p.ensureHistoryCursorVisible()
		}

	case "enter", "d":
		if p.commits != nil && p.historyCursor < len(p.commits) {
			commit := p.commits[p.historyCursor]
			p.viewMode = ViewModeCommitDetail
			p.commitDetailCursor = 0
			p.commitDetailScroll = 0
			return p, p.loadCommitDetail(commit.Hash)
		}
	}

	return p, nil
}

// updateCommitDetail handles key events in the commit detail view.
func (p *Plugin) updateCommitDetail(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		p.viewMode = ViewModeHistory
		p.selectedCommit = nil

	case "j", "down":
		if p.selectedCommit != nil && p.commitDetailCursor < len(p.selectedCommit.Files)-1 {
			p.commitDetailCursor++
			p.ensureCommitDetailCursorVisible()
		}

	case "k", "up":
		if p.commitDetailCursor > 0 {
			p.commitDetailCursor--
			p.ensureCommitDetailCursorVisible()
		}

	case "g":
		p.commitDetailCursor = 0
		p.commitDetailScroll = 0

	case "G":
		if p.selectedCommit != nil && len(p.selectedCommit.Files) > 0 {
			p.commitDetailCursor = len(p.selectedCommit.Files) - 1
			p.ensureCommitDetailCursorVisible()
		}

	case "enter", "d":
		if p.selectedCommit != nil && p.commitDetailCursor < len(p.selectedCommit.Files) {
			file := p.selectedCommit.Files[p.commitDetailCursor]
			p.viewMode = ViewModeDiff
			p.diffFile = file.Path
			p.diffCommit = p.selectedCommit.Hash
			p.diffScroll = 0
			return p, p.loadCommitFileDiff(p.selectedCommit.Hash, file.Path)
		}
	}

	return p, nil
}

// updateDiff handles key events in the diff view.
func (p *Plugin) updateDiff(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Return to previous view based on context
		p.diffContent = ""
		p.diffRaw = ""
		p.parsedDiff = nil
		p.diffHorizOff = 0
		if p.diffCommit != "" {
			// Came from commit detail
			p.viewMode = ViewModeCommitDetail
			p.diffCommit = ""
		} else {
			// Came from status
			p.viewMode = ViewModeStatus
		}
		p.diffFile = ""

	case "j", "down":
		p.diffScroll++

	case "k", "up":
		if p.diffScroll > 0 {
			p.diffScroll--
		}

	case "g":
		p.diffScroll = 0
		p.diffHorizOff = 0

	case "G":
		lines := countLines(p.diffContent)
		maxScroll := lines - (p.height - 2)
		if maxScroll > 0 {
			p.diffScroll = maxScroll
		}

	case "v":
		// Toggle between unified and side-by-side view
		if p.diffViewMode == DiffViewUnified {
			p.diffViewMode = DiffViewSideBySide
			_ = state.SetGitDiffMode("side-by-side")
		} else {
			p.diffViewMode = DiffViewUnified
			_ = state.SetGitDiffMode("unified")
		}
		p.diffHorizOff = 0

	case "<", "H":
		// Horizontal scroll left in side-by-side mode
		if p.diffViewMode == DiffViewSideBySide && p.diffHorizOff > 0 {
			p.diffHorizOff -= 10
			if p.diffHorizOff < 0 {
				p.diffHorizOff = 0
			}
		}

	case ">", "L":
		// Horizontal scroll right in side-by-side mode
		if p.diffViewMode == DiffViewSideBySide {
			p.diffHorizOff += 10
		}
	}

	return p, nil
}

// View renders the plugin.
func (p *Plugin) View(width, height int) string {
	p.width = width
	p.height = height

	switch p.viewMode {
	case ViewModeHistory:
		return p.renderHistory()
	case ViewModeCommitDetail:
		return p.renderCommitDetail()
	case ViewModeDiff:
		return p.renderDiffModal()
	case ViewModeCommit:
		return p.renderCommit()
	default:
		return p.renderMain()
	}
}

// IsFocused returns whether the plugin is focused.
func (p *Plugin) IsFocused() bool { return p.focused }

// SetFocused sets the focus state.
func (p *Plugin) SetFocused(f bool) { p.focused = f }

// Commands returns the available commands.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{ID: "stage-file", Name: "Stage file", Context: "git-status"},
		{ID: "unstage-file", Name: "Unstage file", Context: "git-status"},
		{ID: "stage-all", Name: "Stage all", Context: "git-status"},
		{ID: "commit", Name: "Commit", Context: "git-status"},
		{ID: "show-diff", Name: "Show diff", Context: "git-status"},
		{ID: "show-history", Name: "Show history", Context: "git-status"},
		{ID: "open-file", Name: "Open file", Context: "git-status"},
		{ID: "back", Name: "Back", Context: "git-history"},
		{ID: "view-commit", Name: "View commit", Context: "git-history"},
		{ID: "back", Name: "Back", Context: "git-commit-detail"},
		{ID: "view-diff", Name: "View diff", Context: "git-commit-detail"},
		{ID: "close-diff", Name: "Close diff", Context: "git-diff"},
		{ID: "scroll", Name: "Scroll", Context: "git-diff"},
		{ID: "cancel", Name: "Cancel", Context: "git-commit"},
		{ID: "execute-commit", Name: "Commit", Context: "git-commit"},
	}
}

// FocusContext returns the current focus context.
func (p *Plugin) FocusContext() string {
	switch p.viewMode {
	case ViewModeHistory:
		return "git-history"
	case ViewModeCommitDetail:
		return "git-commit-detail"
	case ViewModeDiff:
		return "git-diff"
	case ViewModeCommit:
		return "git-commit"
	default:
		return "git-status"
	}
}

// Diagnostics returns plugin health info.
func (p *Plugin) Diagnostics() []plugin.Diagnostic {
	status := "ok"
	detail := p.tree.Summary()
	if p.tree.TotalCount() == 0 {
		status = "clean"
	}
	return []plugin.Diagnostic{
		{ID: "git-status", Status: status, Detail: detail},
	}
}

// refresh reloads the git status.
func (p *Plugin) refresh() tea.Cmd {
	return func() tea.Msg {
		if err := p.tree.Refresh(); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshDoneMsg{}
	}
}

// startWatcher starts the file system watcher.
func (p *Plugin) startWatcher() tea.Cmd {
	return func() tea.Msg {
		watcher, err := NewWatcher(p.ctx.WorkDir)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		p.watcher = watcher
		return WatchStartedMsg{}
	}
}

// loadDiff loads the diff for a file, rendering through delta if available.
func (p *Plugin) loadDiff(path string, staged bool) tea.Cmd {
	return func() tea.Msg {
		rawDiff, err := GetDiff(p.ctx.WorkDir, path, staged)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Try to render with delta if available
		content := rawDiff
		if p.externalTool != nil && p.externalTool.ShouldUseDelta() {
			rendered, _ := p.externalTool.RenderWithDelta(rawDiff, false, p.width)
			content = rendered
		}

		return DiffLoadedMsg{Content: content, Raw: rawDiff}
	}
}

// openFile opens a file in the default editor.
func (p *Plugin) openFile(path string) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		fullPath := filepath.Join(p.ctx.WorkDir, path)
		return OpenFileMsg{Editor: editor, Path: fullPath}
	}
}

// ensureCursorVisible adjusts scroll to keep cursor visible.
func (p *Plugin) ensureCursorVisible() {
	visibleRows := p.height - 4 // Account for header and section spacing
	if visibleRows < 1 {
		visibleRows = 1
	}

	if p.cursor < p.scrollOff {
		p.scrollOff = p.cursor
	} else if p.cursor >= p.scrollOff+visibleRows {
		p.scrollOff = p.cursor - visibleRows + 1
	}
}

// countLines counts newlines in a string.
func countLines(s string) int {
	n := 1
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}

// Message types
type RefreshMsg struct{}
type RefreshDoneMsg struct{}
type WatchEventMsg struct{}
type WatchStartedMsg struct{}
type ErrorMsg struct{ Err error }
type DiffLoadedMsg struct {
	Content string // Rendered content (may be from delta)
	Raw     string // Raw diff for built-in rendering
}
type OpenFileMsg struct {
	Editor string
	Path   string
}
type HistoryLoadedMsg struct {
	Commits []*Commit
}
type CommitDetailLoadedMsg struct {
	Commit *Commit
}
type CommitSuccessMsg struct {
	Hash    string
	Subject string
}
type CommitErrorMsg struct {
	Err error
}

// loadHistory loads commit history.
func (p *Plugin) loadHistory() tea.Cmd {
	return func() tea.Msg {
		commits, err := GetCommitHistory(p.ctx.WorkDir, 50)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return HistoryLoadedMsg{Commits: commits}
	}
}

// loadCommitDetail loads full commit information.
func (p *Plugin) loadCommitDetail(hash string) tea.Cmd {
	return func() tea.Msg {
		commit, err := GetCommitDetail(p.ctx.WorkDir, hash)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return CommitDetailLoadedMsg{Commit: commit}
	}
}

// loadCommitFileDiff loads diff for a file in a commit.
func (p *Plugin) loadCommitFileDiff(hash, path string) tea.Cmd {
	return func() tea.Msg {
		rawDiff, err := GetCommitDiff(p.ctx.WorkDir, hash, path)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Try to render with delta if available
		content := rawDiff
		if p.externalTool != nil && p.externalTool.ShouldUseDelta() {
			rendered, _ := p.externalTool.RenderWithDelta(rawDiff, false, p.width)
			content = rendered
		}

		return DiffLoadedMsg{Content: content, Raw: rawDiff}
	}
}

// ensureHistoryCursorVisible adjusts scroll to keep history cursor visible.
func (p *Plugin) ensureHistoryCursorVisible() {
	visibleRows := p.height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	if p.historyCursor < p.historyScroll {
		p.historyScroll = p.historyCursor
	} else if p.historyCursor >= p.historyScroll+visibleRows {
		p.historyScroll = p.historyCursor - visibleRows + 1
	}
}

// ensureCommitDetailCursorVisible adjusts scroll to keep commit detail cursor visible.
func (p *Plugin) ensureCommitDetailCursorVisible() {
	visibleRows := p.height - 12 // Account for commit metadata
	if visibleRows < 1 {
		visibleRows = 1
	}

	if p.commitDetailCursor < p.commitDetailScroll {
		p.commitDetailScroll = p.commitDetailCursor
	} else if p.commitDetailCursor >= p.commitDetailScroll+visibleRows {
		p.commitDetailScroll = p.commitDetailCursor - visibleRows + 1
	}
}

// TickCmd returns a command that triggers a refresh every second.
func TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}

// initCommitTextarea initializes the commit message textarea.
func (p *Plugin) initCommitTextarea() {
	p.commitMessage = textarea.New()
	p.commitMessage.SetValue("") // Ensure empty
	p.commitMessage.Placeholder = "Type your commit message..."
	// Make placeholder more visible (default color 240 is too dim)
	p.commitMessage.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	p.commitMessage.Focus()
	p.commitMessage.CharLimit = 0
	p.commitMessage.SetWidth(60)
	p.commitMessage.SetHeight(5)
	p.commitError = ""
}

// updateCommit handles key events in the commit view.
func (p *Plugin) updateCommit(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel commit, return to status
		p.viewMode = ViewModeStatus
		p.commitError = ""
		return p, nil

	case "alt+enter", "alt+s":
		// Execute commit if message not empty
		message := strings.TrimSpace(p.commitMessage.Value())
		if message == "" {
			p.commitError = "Commit message cannot be empty"
			return p, nil
		}
		p.commitInProgress = true
		return p, p.doCommit(message)
	}

	// Pass other keys to textarea
	var cmd tea.Cmd
	p.commitMessage, cmd = p.commitMessage.Update(msg)
	return p, cmd
}

// doCommit executes the git commit asynchronously.
func (p *Plugin) doCommit(message string) tea.Cmd {
	workDir := p.ctx.WorkDir
	return func() tea.Msg {
		hash, err := ExecuteCommit(workDir, message)
		if err != nil {
			return CommitErrorMsg{Err: err}
		}
		// Extract first line as subject
		subject := strings.Split(message, "\n")[0]
		return CommitSuccessMsg{Hash: hash, Subject: subject}
	}
}
