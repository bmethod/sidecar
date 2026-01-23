package gitstatus

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/plugin"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// updateBranchPicker handles key events in the branch picker modal.
func (p *Plugin) updateBranchPicker(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Close picker
		p.viewMode = p.branchReturnMode
		p.branches = nil
		return p, nil

	case "j", "down":
		if len(p.branches) > 0 && p.branchCursor < len(p.branches)-1 {
			p.branchCursor++
		}
		return p, nil

	case "k", "up":
		if p.branchCursor > 0 {
			p.branchCursor--
		}
		return p, nil

	case "g":
		p.branchCursor = 0
		return p, nil

	case "G":
		if len(p.branches) > 0 {
			p.branchCursor = len(p.branches) - 1
		}
		return p, nil

	case "enter":
		// Switch to selected branch
		if len(p.branches) > 0 && p.branchCursor < len(p.branches) {
			branch := p.branches[p.branchCursor]
			if !branch.IsCurrent {
				return p, p.doSwitchBranch(branch.Name)
			}
		}
		return p, nil
	}

	return p, nil
}

// doSwitchBranch switches to a different branch.
func (p *Plugin) doSwitchBranch(branchName string) tea.Cmd {
	workDir := p.repoRoot
	return func() tea.Msg {
		err := CheckoutBranch(workDir, branchName)
		if err != nil {
			return BranchErrorMsg{Err: err}
		}
		return BranchSwitchSuccessMsg{Branch: branchName}
	}
}

// loadBranches loads the branch list.
func (p *Plugin) loadBranches() tea.Cmd {
	workDir := p.repoRoot
	return func() tea.Msg {
		branches, err := GetBranches(workDir)
		if err != nil {
			return BranchErrorMsg{Err: err}
		}
		return BranchListLoadedMsg{Branches: branches}
	}
}

// renderBranchPicker renders the branch picker modal.
func (p *Plugin) renderBranchPicker() string {
	// Render the background (status view dimmed)
	background := p.renderThreePaneView()

	// Clear previous hit regions for branch items
	p.mouseHandler.HitMap.Clear()

	var sb strings.Builder

	// Title
	title := styles.Title.Render(" Branches ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Calculate modal width first (needed for hit regions)
	modalWidth := 50
	for _, b := range p.branches {
		lineLen := len(b.Name) + 10
		if lineLen > modalWidth {
			modalWidth = lineLen
		}
	}
	if modalWidth > p.width-10 {
		modalWidth = p.width - 10
	}

	// Track visible branches for hit regions
	var visibleStart, visibleEnd int

	if len(p.branches) == 0 {
		sb.WriteString(styles.Muted.Render("  Loading branches..."))
	} else {
		// Calculate visible range (max 15 branches visible)
		maxVisible := 15
		if p.height-10 < maxVisible {
			maxVisible = p.height - 10
		}
		if maxVisible < 5 {
			maxVisible = 5
		}

		start := 0
		if p.branchCursor >= maxVisible {
			start = p.branchCursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(p.branches) {
			end = len(p.branches)
		}
		visibleStart = start
		visibleEnd = end

		for i := start; i < end; i++ {
			branch := p.branches[i]
			selected := i == p.branchCursor
			hovered := i == p.branchPickerHover

			line := p.renderBranchLine(branch, selected, hovered)
			sb.WriteString(line)
			if i < end-1 {
				sb.WriteString("\n")
			}
		}

		// Scroll indicator
		if len(p.branches) > maxVisible {
			sb.WriteString("\n\n")
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("  %d/%d branches", p.branchCursor+1, len(p.branches))))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(styles.Muted.Render("  Enter to switch, j/k to navigate, Esc to cancel"))

	modalContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(modalWidth).
		Render(sb.String())

	// Register hit regions for visible branches
	// Modal adds: border(1) + padding(1) on each side vertically = 4 total height added
	// Title + blank line = 2 lines before branches
	// Modal position is centered
	actualModalWidth := modalWidth + 6 // border(1) + padding(2) on each side
	modalHeight := 4 + 2 + (visibleEnd - visibleStart) + 4 // approx: borders + title + branches + footer
	startX := (p.width - actualModalWidth) / 2
	startY := (p.height - modalHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Branch items start at: startY + border(1) + padding(1) + title(1) + blank(1) = startY + 4
	branchStartY := startY + 4
	for i := visibleStart; i < visibleEnd; i++ {
		lineY := branchStartY + (i - visibleStart)
		p.mouseHandler.HitMap.AddRect(regionBranchItem, startX, lineY, actualModalWidth, 1, i)
	}

	return ui.OverlayModal(background, modalContent, p.width, p.height)
}

// renderBranchLine renders a single branch line.
func (p *Plugin) renderBranchLine(branch *Branch, selected, hovered bool) string {
	// Current branch indicator
	indicator := "  "
	if branch.IsCurrent {
		indicator = "* "
	}

	// Branch name
	name := branch.Name

	// Tracking info
	trackingInfo := branch.FormatTrackingInfo()
	trackingInfoPlain := trackingInfo
	if trackingInfo != "" {
		trackingInfo = " " + styles.StatusModified.Render(trackingInfo)
	}

	// Upstream indicator
	upstream := ""
	if branch.Upstream != "" {
		upstream = styles.Muted.Render(" → " + branch.Upstream)
	}

	// Build plain line for selected/hovered states (need consistent width)
	buildPlainLine := func() string {
		line := fmt.Sprintf("%s%s", indicator, name)
		if trackingInfoPlain != "" {
			line += " " + trackingInfoPlain
		}
		maxWidth := 45
		if len(line) < maxWidth {
			line += strings.Repeat(" ", maxWidth-len(line))
		}
		return line
	}

	if selected {
		return styles.ListItemSelected.Render(buildPlainLine())
	}

	if hovered {
		// Use a hover style - slightly highlighted background
		return styles.ListItemSelected.Render(buildPlainLine())
	}

	// Style based on current branch
	nameStyle := styles.Body
	if branch.IsCurrent {
		nameStyle = styles.StatusStaged
	}

	return styles.ListItemNormal.Render(fmt.Sprintf("%s%s%s%s", indicator, nameStyle.Render(name), trackingInfo, upstream))
}
