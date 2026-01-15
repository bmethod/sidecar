package worktree

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/plugins/gitstatus"
)

// MergeWorkflowStep represents the current step in the merge workflow.
type MergeWorkflowStep int

const (
	MergeStepReviewDiff MergeWorkflowStep = iota
	MergeStepPush
	MergeStepCreatePR
	MergeStepWaitingMerge
	MergeStepCleanup
	MergeStepDone
)

// String returns a display name for the merge step.
func (s MergeWorkflowStep) String() string {
	switch s {
	case MergeStepReviewDiff:
		return "Review Diff"
	case MergeStepPush:
		return "Push Branch"
	case MergeStepCreatePR:
		return "Create PR"
	case MergeStepWaitingMerge:
		return "Waiting for Merge"
	case MergeStepCleanup:
		return "Cleanup"
	case MergeStepDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// MergeWorkflowState holds the state for the merge workflow modal.
type MergeWorkflowState struct {
	Worktree    *Worktree
	Step        MergeWorkflowStep
	DiffSummary string
	PRTitle     string
	PRBody      string
	PRURL       string
	Error       error
	StepStatus  map[MergeWorkflowStep]string // "pending", "running", "done", "error"
}

// MergeStepCompleteMsg signals a merge workflow step completed.
type MergeStepCompleteMsg struct {
	WorktreeName string
	Step         MergeWorkflowStep
	Data         string // Step-specific data (e.g., PR URL)
	Err          error
}

// CheckPRMergedMsg signals the result of checking if a PR was merged.
type CheckPRMergedMsg struct {
	WorktreeName string
	Merged       bool
	Err          error
}

// UncommittedChangesCheckMsg signals the result of checking for uncommitted changes.
type UncommittedChangesCheckMsg struct {
	WorktreeName     string
	HasChanges       bool
	StagedCount      int
	ModifiedCount    int
	UntrackedCount   int
	Err              error
}

// MergeCommitDoneMsg signals that the commit before merge completed.
type MergeCommitDoneMsg struct {
	WorktreeName string
	CommitHash   string
	Err          error
}

// MergeCommitState holds state for the commit-before-merge modal.
type MergeCommitState struct {
	Worktree       *Worktree
	StagedCount    int
	ModifiedCount  int
	UntrackedCount int
	CommitMessage  string
	Error          string
}

// checkUncommittedChanges checks if a worktree has uncommitted changes.
func (p *Plugin) checkUncommittedChanges(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		tree := gitstatus.NewFileTree(wt.Path)
		if err := tree.Refresh(); err != nil {
			return UncommittedChangesCheckMsg{
				WorktreeName: wt.Name,
				HasChanges:   false,
				Err:          err,
			}
		}

		stagedCount := len(tree.Staged)
		modifiedCount := len(tree.Modified)
		untrackedCount := len(tree.Untracked)
		hasChanges := stagedCount > 0 || modifiedCount > 0 || untrackedCount > 0

		return UncommittedChangesCheckMsg{
			WorktreeName:   wt.Name,
			HasChanges:     hasChanges,
			StagedCount:    stagedCount,
			ModifiedCount:  modifiedCount,
			UntrackedCount: untrackedCount,
		}
	}
}

// stageAllAndCommit stages all changes and commits with the given message.
func (p *Plugin) stageAllAndCommit(wt *Worktree, message string) tea.Cmd {
	return func() tea.Msg {
		tree := gitstatus.NewFileTree(wt.Path)

		// Stage all changes
		if err := tree.StageAll(); err != nil {
			return MergeCommitDoneMsg{
				WorktreeName: wt.Name,
				Err:          fmt.Errorf("failed to stage: %w", err),
			}
		}

		// Execute commit
		hash, err := gitstatus.ExecuteCommit(wt.Path, message)
		if err != nil {
			return MergeCommitDoneMsg{
				WorktreeName: wt.Name,
				Err:          err,
			}
		}

		return MergeCommitDoneMsg{
			WorktreeName: wt.Name,
			CommitHash:   hash,
		}
	}
}

// startMergeWorkflow initializes the merge workflow for a worktree.
// It first checks for uncommitted changes and shows a commit modal if needed.
func (p *Plugin) startMergeWorkflow(wt *Worktree) tea.Cmd {
	if wt == nil {
		return nil
	}

	// Check for uncommitted changes before proceeding
	return p.checkUncommittedChanges(wt)
}

// proceedToMergeWorkflow initializes the actual merge workflow (after commit check passes).
func (p *Plugin) proceedToMergeWorkflow(wt *Worktree) tea.Cmd {
	// Initialize merge state
	p.mergeState = &MergeWorkflowState{
		Worktree:   wt,
		Step:       MergeStepReviewDiff,
		PRTitle:    wt.Branch, // Default title to branch name
		PRBody:     "",
		StepStatus: make(map[MergeWorkflowStep]string),
	}
	p.mergeState.StepStatus[MergeStepReviewDiff] = "running"

	p.viewMode = ViewModeMerge

	// Load diff summary for review
	return p.loadMergeDiff(wt)
}

// loadMergeDiff loads the diff summary for the merge workflow.
func (p *Plugin) loadMergeDiff(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		// Get diff against base branch
		baseBranch := wt.BaseBranch
		if baseBranch == "" {
			baseBranch = "main"
		}

		diff, err := getDiffFromBase(wt.Path, baseBranch)
		if err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: wt.Name,
				Step:         MergeStepReviewDiff,
				Data:         "",
				Err:          err,
			}
		}

		// Get a summary (stat output)
		summary, _ := getDiffSummary(wt.Path)

		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepReviewDiff,
			Data:         summary + "\n\n" + truncateDiff(diff, 50),
		}
	}
}

// truncateDiff truncates a diff to a maximum number of lines.
func truncateDiff(diff string, maxLines int) string {
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

// pushForMerge pushes the branch for the merge workflow.
func (p *Plugin) pushForMerge(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		err := doPush(wt.Path, wt.Branch, false, true)
		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepPush,
			Err:          err,
		}
	}
}

// createPR creates a pull request using gh CLI.
func (p *Plugin) createPR(wt *Worktree, title, body string) tea.Cmd {
	return func() tea.Msg {
		baseBranch := wt.BaseBranch
		if baseBranch == "" {
			baseBranch = "main"
		}

		// Build gh pr create command
		args := []string{"pr", "create",
			"--title", title,
			"--body", body,
			"--base", baseBranch,
		}

		cmd := exec.Command("gh", args...)
		cmd.Dir = wt.Path
		output, err := cmd.CombinedOutput()

		if err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: wt.Name,
				Step:         MergeStepCreatePR,
				Err:          fmt.Errorf("gh pr create: %s: %w", strings.TrimSpace(string(output)), err),
			}
		}

		// Output should contain the PR URL
		prURL := strings.TrimSpace(string(output))

		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepCreatePR,
			Data:         prURL,
		}
	}
}

// checkPRMerged checks if a PR has been merged using gh CLI.
func (p *Plugin) checkPRMerged(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		// Use gh pr view to check status
		cmd := exec.Command("gh", "pr", "view", "--json", "state,mergedAt")
		cmd.Dir = wt.Path
		output, err := cmd.Output()

		if err != nil {
			return CheckPRMergedMsg{
				WorktreeName: wt.Name,
				Merged:       false,
				Err:          err,
			}
		}

		// Parse JSON response
		var prStatus struct {
			State    string `json:"state"`
			MergedAt string `json:"mergedAt"`
		}

		merged := false
		if err := json.Unmarshal(output, &prStatus); err == nil {
			merged = prStatus.MergedAt != "" || prStatus.State == "MERGED"
		}

		return CheckPRMergedMsg{
			WorktreeName: wt.Name,
			Merged:       merged,
		}
	}
}

// cleanupAfterMerge removes the worktree and branch after a successful merge.
func (p *Plugin) cleanupAfterMerge(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		name := wt.Name
		path := wt.Path
		branch := wt.Branch

		// Stop agent if running
		if wt.Agent != nil {
			sessionName := wt.Agent.TmuxSession
			exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		}

		// Remove worktree
		if err := doDeleteWorktree(path); err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: name,
				Step:         MergeStepCleanup,
				Err:          fmt.Errorf("remove worktree: %w", err),
			}
		}

		// Delete the branch (it's been merged)
		cmd := exec.Command("git", "branch", "-d", branch)
		cmd.Dir = p.ctx.WorkDir
		if output, err := cmd.CombinedOutput(); err != nil {
			// Try force delete if regular delete fails
			cmd = exec.Command("git", "branch", "-D", branch)
			cmd.Dir = p.ctx.WorkDir
			if output, err = cmd.CombinedOutput(); err != nil {
				return MergeStepCompleteMsg{
					WorktreeName: name,
					Step:         MergeStepCleanup,
					Err:          fmt.Errorf("delete branch: %s: %w", strings.TrimSpace(string(output)), err),
				}
			}
		}

		return MergeStepCompleteMsg{
			WorktreeName: name,
			Step:         MergeStepCleanup,
		}
	}
}

// schedulePRCheck schedules a periodic check for PR merge status.
func (p *Plugin) schedulePRCheck(worktreeName string, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return checkPRMergeMsg{WorktreeName: worktreeName}
	})
}

// checkPRMergeMsg triggers a PR merge status check.
type checkPRMergeMsg struct {
	WorktreeName string
}

// advanceMergeStep moves to the next step in the merge workflow.
// It marks the current step as "done" and advances to the next step.
func (p *Plugin) advanceMergeStep() tea.Cmd {
	if p.mergeState == nil {
		return nil
	}

	switch p.mergeState.Step {
	case MergeStepReviewDiff:
		// Move to push step (ReviewDiff marked done by message handler)
		p.mergeState.Step = MergeStepPush
		p.mergeState.StepStatus[MergeStepPush] = "running"
		return p.pushForMerge(p.mergeState.Worktree)

	case MergeStepPush:
		// Mark Push as done, move to create PR step
		p.mergeState.StepStatus[MergeStepPush] = "done"
		p.mergeState.Step = MergeStepCreatePR
		p.mergeState.StepStatus[MergeStepCreatePR] = "running"
		title := p.mergeState.PRTitle
		if title == "" {
			title = p.mergeState.Worktree.Branch
		}
		body := p.mergeState.PRBody
		if body == "" {
			body = "Created from worktree manager"
		}
		return p.createPR(p.mergeState.Worktree, title, body)

	case MergeStepCreatePR:
		// Mark CreatePR as done, move to waiting for merge
		p.mergeState.StepStatus[MergeStepCreatePR] = "done"
		p.mergeState.Step = MergeStepWaitingMerge
		p.mergeState.StepStatus[MergeStepWaitingMerge] = "running"
		// Schedule periodic checks
		return p.schedulePRCheck(p.mergeState.Worktree.Name, 10*time.Second)

	case MergeStepWaitingMerge:
		// Mark WaitingMerge as done, move to cleanup
		p.mergeState.StepStatus[MergeStepWaitingMerge] = "done"
		p.mergeState.Step = MergeStepCleanup
		p.mergeState.StepStatus[MergeStepCleanup] = "running"
		return p.cleanupAfterMerge(p.mergeState.Worktree)

	case MergeStepCleanup:
		// Done
		p.mergeState.Step = MergeStepDone
		p.mergeState.StepStatus[MergeStepDone] = "done"
		return nil
	}

	return nil
}

// cancelMergeWorkflow cancels the merge workflow and returns to list view.
func (p *Plugin) cancelMergeWorkflow() {
	p.mergeState = nil
	p.viewMode = ViewModeList
}
