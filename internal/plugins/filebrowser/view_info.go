package filebrowser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
)

// renderInfoModalContent renders the file info modal.
func (p *Plugin) renderInfoModalContent() string {
	var path string
	var isDir bool

	// Determine target file
	if p.activePane == PanePreview && p.previewFile != "" {
		path = p.previewFile
		// Check if it's a directory (previewFile is usually a file, but let's be safe)
		if info, err := os.Stat(filepath.Join(p.ctx.WorkDir, path)); err == nil {
			isDir = info.IsDir()
		}
	} else {
		node := p.tree.GetNode(p.treeCursor)
		if node != nil {
			path = node.Path
			isDir = node.IsDir
		}
	}

	if path == "" {
		return styles.ModalBox.Render("No file selected")
	}

	// Gather details
	fullPath := filepath.Join(p.ctx.WorkDir, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return styles.ModalBox.Render(styles.StatusDeleted.Render("Error reading file: " + err.Error()))
	}

	// Format fields
	name := info.Name()
	kind := "File"
	if isDir {
		kind = "Directory"
	} else {
		ext := filepath.Ext(name)
		if ext != "" && len(ext) > 1 {
			kind = strings.ToUpper(ext[1:]) + " File"
		}
	}

	size := formatSize(info.Size())
	if isDir {
		size = "--"
	}

	modTime := info.ModTime().Format("Jan 2, 2006 at 15:04")
	perms := info.Mode().String()

	// Build view
	var sb strings.Builder

	// Title area
	icon := "📄"
	if isDir {
		icon = "📁"
	}

	title := styles.ModalTitle.Render(fmt.Sprintf("%s %s", icon, name))
	sb.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(50).Render(title))
	sb.WriteString("\n\n")

	// Key-Value pairs
	labelStyle := styles.Muted.Copy().Width(12).Align(lipgloss.Right).MarginRight(2)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)

		fields := []struct{ label, value string }{

			{"Kind:", kind},

			{"Size:", size},

			{"Where:", filepath.Dir(path)},

			{"Modified:", modTime},

			{"Permissions:", perms},

			{"Git Status:", p.gitStatus},

			{"Commit:", p.gitLastCommit},

		}

	for _, f := range fields {
		line := lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render(f.label),
			valueStyle.Render(f.value),
		)
		sb.WriteString(line + "\n")
	}

	// Footer hint
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(50).Foreground(styles.TextMuted).Render("Press 'esc' to close"))

	return styles.ModalBox.
		Width(60).
		Padding(1, 2).
		Render(sb.String())
}
