package worktree

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/features"
)

// Interactive mode constants
const (
	// doubleEscapeDelay is the max time between Escape presses for double-escape exit.
	// Single Escape is delayed by this amount to detect double-press.
	doubleEscapeDelay = 150 * time.Millisecond

	// pollingDecayFast is the polling interval during active typing.
	pollingDecayFast = 50 * time.Millisecond

	// pollingDecayMedium is the polling interval after brief inactivity.
	pollingDecayMedium = 200 * time.Millisecond

	// pollingDecaySlow is the polling interval after extended inactivity.
	pollingDecaySlow = 500 * time.Millisecond

	// inactivityMediumThreshold triggers medium polling.
	inactivityMediumThreshold = 2 * time.Second

	// inactivitySlowThreshold triggers slow polling.
	inactivitySlowThreshold = 10 * time.Second
)

// escapeTimerMsg is sent when the escape delay timer fires.
// If pendingEscape is still true, we forward the single Escape to tmux.
type escapeTimerMsg struct{}

// MapKeyToTmux translates a Bubble Tea key message to a tmux send-keys argument.
// Returns the tmux key name and whether to use literal mode (-l).
// For modified keys and special keys, returns the tmux key name.
// For literal characters, returns the character with useLiteral=true.
func MapKeyToTmux(msg tea.KeyMsg) (key string, useLiteral bool) {
	// Handle special keys
	// Note: KeyCtrlI == KeyTab and KeyCtrlM == KeyEnter in BubbleTea,
	// so we handle Tab and Enter first, then other Ctrl keys.
	switch msg.Type {
	case tea.KeyEnter: // Also KeyCtrlM
		return "Enter", false
	case tea.KeyBackspace:
		return "BSpace", false
	case tea.KeyDelete:
		return "DC", false
	case tea.KeyTab: // Also KeyCtrlI
		return "Tab", false
	case tea.KeySpace:
		return "Space", false
	case tea.KeyUp:
		return "Up", false
	case tea.KeyDown:
		return "Down", false
	case tea.KeyLeft:
		return "Left", false
	case tea.KeyRight:
		return "Right", false
	case tea.KeyHome:
		return "Home", false
	case tea.KeyEnd:
		return "End", false
	case tea.KeyPgUp:
		return "PPage", false
	case tea.KeyPgDown:
		return "NPage", false
	case tea.KeyInsert:
		return "IC", false
	case tea.KeyEscape:
		return "Escape", false

	// Ctrl combinations (excluding KeyCtrlI/Tab and KeyCtrlM/Enter handled above)
	case tea.KeyCtrlA:
		return "C-a", false
	case tea.KeyCtrlB:
		return "C-b", false
	case tea.KeyCtrlC:
		return "C-c", false
	case tea.KeyCtrlD:
		return "C-d", false
	case tea.KeyCtrlE:
		return "C-e", false
	case tea.KeyCtrlF:
		return "C-f", false
	case tea.KeyCtrlG:
		return "C-g", false
	case tea.KeyCtrlH:
		return "C-h", false
	case tea.KeyCtrlJ:
		return "C-j", false
	case tea.KeyCtrlK:
		return "C-k", false
	case tea.KeyCtrlL:
		return "C-l", false
	case tea.KeyCtrlN:
		return "C-n", false
	case tea.KeyCtrlO:
		return "C-o", false
	case tea.KeyCtrlP:
		return "C-p", false
	case tea.KeyCtrlQ:
		return "C-q", false
	case tea.KeyCtrlR:
		return "C-r", false
	case tea.KeyCtrlS:
		return "C-s", false
	case tea.KeyCtrlT:
		return "C-t", false
	case tea.KeyCtrlU:
		return "C-u", false
	case tea.KeyCtrlV:
		return "C-v", false
	case tea.KeyCtrlW:
		return "C-w", false
	case tea.KeyCtrlX:
		return "C-x", false
	case tea.KeyCtrlY:
		return "C-y", false
	case tea.KeyCtrlZ:
		return "C-z", false

	// Function keys (F1-F12)
	case tea.KeyF1:
		return "F1", false
	case tea.KeyF2:
		return "F2", false
	case tea.KeyF3:
		return "F3", false
	case tea.KeyF4:
		return "F4", false
	case tea.KeyF5:
		return "F5", false
	case tea.KeyF6:
		return "F6", false
	case tea.KeyF7:
		return "F7", false
	case tea.KeyF8:
		return "F8", false
	case tea.KeyF9:
		return "F9", false
	case tea.KeyF10:
		return "F10", false
	case tea.KeyF11:
		return "F11", false
	case tea.KeyF12:
		return "F12", false

	case tea.KeyRunes:
		// Regular character input
		if len(msg.Runes) > 0 {
			return string(msg.Runes), true
		}
		return "", true
	}

	// Fallback for any unhandled key types
	if msg.String() != "" {
		return msg.String(), true
	}
	return "", true
}

// sendKeyToTmux sends a key to a tmux pane using send-keys.
// Uses the tmux key name syntax (e.g., "Enter", "C-c", "Up").
func sendKeyToTmux(sessionName, key string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, key)
	return cmd.Run()
}

// sendLiteralToTmux sends literal text to a tmux pane using send-keys -l.
// This prevents tmux from interpreting special key names.
func sendLiteralToTmux(sessionName, text string) error {
	cmd := exec.Command("tmux", "send-keys", "-l", "-t", sessionName, text)
	return cmd.Run()
}

// enterInteractiveMode enters interactive mode for the current selection.
// Returns a tea.Cmd if mode entry succeeded, nil otherwise.
// Requires tmux_interactive_input feature flag to be enabled.
func (p *Plugin) enterInteractiveMode() tea.Cmd {
	// Check feature flag
	if !features.IsEnabled(features.TmuxInteractiveInput.Name) {
		return nil
	}

	// Determine target based on current selection
	var sessionName, paneID string

	if p.shellSelected {
		// Shell session
		if p.selectedShellIdx < 0 || p.selectedShellIdx >= len(p.shells) {
			return nil
		}
		shell := p.shells[p.selectedShellIdx]
		if shell.Agent == nil {
			return nil
		}
		sessionName = shell.TmuxName
		paneID = shell.Agent.TmuxPane
	} else {
		// Worktree
		wt := p.selectedWorktree()
		if wt == nil || wt.Agent == nil {
			return nil
		}
		sessionName = wt.Agent.TmuxSession
		paneID = wt.Agent.TmuxPane
	}

	// Initialize interactive state
	p.interactiveState = &InteractiveState{
		Active:        true,
		TargetPane:    paneID,
		TargetSession: sessionName,
		LastKeyTime:   time.Now(),
	}

	p.viewMode = ViewModeInteractive

	// Trigger immediate poll for fresh content
	return p.pollInteractivePane()
}

// exitInteractiveMode exits interactive mode and returns to list view.
func (p *Plugin) exitInteractiveMode() {
	if p.interactiveState != nil {
		p.interactiveState.Active = false
	}
	p.interactiveState = nil
	p.viewMode = ViewModeList
}

// handleInteractiveKeys processes key input in interactive mode.
// Returns a tea.Cmd for any async operations needed.
func (p *Plugin) handleInteractiveKeys(msg tea.KeyMsg) tea.Cmd {
	if p.interactiveState == nil || !p.interactiveState.Active {
		p.exitInteractiveMode()
		return nil
	}

	// Check for exit keys

	// Primary exit: Ctrl+\ (immediate, unambiguous)
	if msg.String() == "ctrl+\\" {
		p.exitInteractiveMode()
		return nil
	}

	// Secondary exit: Double-Escape with 150ms delay
	// Per spec: first Escape is delayed to detect double-press
	if msg.Type == tea.KeyEscape {
		if p.interactiveState.EscapePressed {
			// Second Escape within window: exit interactive mode
			p.interactiveState.EscapePressed = false
			p.exitInteractiveMode()
			return nil
		}
		// First Escape: mark pending and start delay timer
		// Do NOT forward to tmux yet - wait for timer or next key
		p.interactiveState.EscapePressed = true
		p.interactiveState.EscapeTime = time.Now()
		return tea.Tick(doubleEscapeDelay, func(t time.Time) tea.Msg {
			return escapeTimerMsg{}
		})
	}

	// Non-escape key: check if we have a pending Escape to forward first
	var cmds []tea.Cmd
	if p.interactiveState.EscapePressed {
		p.interactiveState.EscapePressed = false
		// Forward the pending Escape before this key
		if err := sendKeyToTmux(p.interactiveState.TargetSession, "Escape"); err != nil {
			p.exitInteractiveMode()
			return nil
		}
	}

	// Update last key time for polling decay
	p.interactiveState.LastKeyTime = time.Now()

	// Map key to tmux format and send
	key, useLiteral := MapKeyToTmux(msg)
	if key == "" {
		return tea.Batch(cmds...)
	}

	sessionName := p.interactiveState.TargetSession
	var err error
	if useLiteral {
		err = sendLiteralToTmux(sessionName, key)
	} else {
		err = sendKeyToTmux(sessionName, key)
	}

	if err != nil {
		// Session may have died - exit interactive mode
		p.exitInteractiveMode()
		return nil
	}

	// Schedule fast poll to show updated output quickly
	cmds = append(cmds, p.pollInteractivePane())
	return tea.Batch(cmds...)
}

// handleEscapeTimer processes the escape delay timer firing.
// If a single Escape is still pending (no second Escape arrived), forward it to tmux.
func (p *Plugin) handleEscapeTimer() tea.Cmd {
	if p.interactiveState == nil || !p.interactiveState.Active {
		return nil
	}

	if !p.interactiveState.EscapePressed {
		// Escape was already handled (double-press or another key arrived)
		return nil
	}

	// Timer fired with pending Escape: forward the single Escape to tmux
	p.interactiveState.EscapePressed = false
	if err := sendKeyToTmux(p.interactiveState.TargetSession, "Escape"); err != nil {
		p.exitInteractiveMode()
		return nil
	}

	// Update last key time and poll
	p.interactiveState.LastKeyTime = time.Now()
	return p.pollInteractivePane()
}

// pollInteractivePane schedules a poll for interactive mode with adaptive timing.
func (p *Plugin) pollInteractivePane() tea.Cmd {
	if p.interactiveState == nil || !p.interactiveState.Active {
		return nil
	}

	// Determine polling interval based on activity
	interval := pollingDecayFast
	inactivity := time.Since(p.interactiveState.LastKeyTime)

	if inactivity > inactivitySlowThreshold {
		interval = pollingDecaySlow
	} else if inactivity > inactivityMediumThreshold {
		interval = pollingDecayMedium
	}

	// Use existing shell or worktree polling mechanism
	if p.shellSelected && p.selectedShellIdx >= 0 && p.selectedShellIdx < len(p.shells) {
		return p.scheduleShellPollByName(p.shells[p.selectedShellIdx].TmuxName, interval)
	}
	if wt := p.selectedWorktree(); wt != nil {
		return p.scheduleAgentPoll(wt.Name, interval)
	}
	return nil
}
