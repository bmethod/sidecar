package gitstatus

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/modal"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

const (
	pullMenuOptionMerge     = "pull-merge"      // List item ID for merge strategy
	pullMenuOptionRebase    = "pull-rebase"     // List item ID for rebase strategy
	pullMenuOptionFFOnly    = "pull-ff-only"    // List item ID for fast-forward only
	pullMenuOptionAutostash = "pull-autostash"  // List item ID for rebase + autostash
	pullMenuActionID        = "pull-menu-action" // Primary action (Enter key)

	pullMenuModalWidth = 50 // Default modal width
	pullMenuMinWidth   = 20 // Minimum modal width
)

// ensurePullModal builds/rebuilds the pull menu modal.
func (p *Plugin) ensurePullModal() {
	modalW := pullMenuModalWidth
	if modalW > p.width-4 {
		modalW = p.width - 4
	}
	if modalW < pullMenuMinWidth {
		modalW = pullMenuMinWidth
	}

	// Only rebuild if modal doesn't exist or width changed
	if p.pullModal != nil && p.pullModalWidth == modalW {
		return
	}
	p.pullModalWidth = modalW

	items := []modal.ListItem{
		{ID: pullMenuOptionMerge, Label: "Pull (merge)"},
		{ID: pullMenuOptionRebase, Label: "Pull (rebase)"},
		{ID: pullMenuOptionFFOnly, Label: "Pull (fast-forward only)"},
		{ID: pullMenuOptionAutostash, Label: "Pull (rebase + autostash)"},
	}

	p.pullModal = modal.New("Pull",
		modal.WithWidth(modalW),
		modal.WithPrimaryAction(pullMenuActionID),
	).
		AddSection(modal.List("pull-options", items, &p.pullSelectedIdx, modal.WithMaxVisible(4)))
}

// renderPullMenu renders the pull options popup menu.
func (p *Plugin) renderPullMenu() string {
	background := p.renderThreePaneView()

	p.ensurePullModal()
	if p.pullModal == nil {
		return background
	}

	modalContent := p.pullModal.Render(p.width, p.height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, p.width, p.height)
}

// renderPullConflict renders the pull conflict resolution modal.
func (p *Plugin) renderPullConflict() string {
	background := p.renderThreePaneView()

	var sb strings.Builder

	sb.WriteString(styles.StatusDeleted.Render(" Conflicts "))
	sb.WriteString("\n\n")

	// Show conflict type
	conflictLabel := "Merge"
	if p.pullConflictType == "rebase" {
		conflictLabel = "Rebase"
	}
	sb.WriteString(styles.Muted.Render(fmt.Sprintf("%s produced conflicts in %d file(s):", conflictLabel, len(p.pullConflictFiles))))
	sb.WriteString("\n\n")

	// Show conflicted files (max 8)
	maxFiles := 8
	for i, f := range p.pullConflictFiles {
		if i >= maxFiles {
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("  ... and %d more", len(p.pullConflictFiles)-maxFiles)))
			sb.WriteString("\n")
			break
		}
		sb.WriteString(styles.StatusModified.Render("  U " + f))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Action options
	abortStyle := styles.ButtonDanger
	sb.WriteString(styles.KeyHint.Render(" a "))
	sb.WriteString(" ")
	sb.WriteString(abortStyle.Render(" Abort "))

	sb.WriteString("\n\n")
	sb.WriteString(styles.Muted.Render("Resolve conflicts in your editor, then commit."))

	menuWidth := ui.ModalWidthMedium

	menuContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Error).
		Padding(1, 2).
		Width(menuWidth).
		Render(sb.String())

	return ui.OverlayModal(background, menuContent, p.width, p.height)
}
