package filebrowser

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchGitInfo retrieves git status and last commit for a file.
func (p *Plugin) fetchGitInfo(path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return GitInfoMsg{}
		}

		// Check status
		statusCmd := exec.Command("git", "status", "--porcelain", path)
		statusCmd.Dir = p.ctx.WorkDir
		statusOut, _ := statusCmd.Output()
		status := strings.TrimSpace(string(statusOut))
		if status == "" {
			status = "Clean"
		} else {
			// Extract status code (e.g. "M ", "??")
			if len(status) >= 2 {
				status = status[:2]
			}
		}

		// Check last commit
		logCmd := exec.Command("git", "log", "-1", "--format=%h - %s (%cr)", path)
		logCmd.Dir = p.ctx.WorkDir
		logOut, _ := logCmd.Output()
		lastCommit := strings.TrimSpace(string(logOut))
		if lastCommit == "" {
			lastCommit = "Not committed"
		}

		return GitInfoMsg{Status: status, LastCommit: lastCommit}
	}
}
