package version

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// UpdateAvailableMsg is sent when a new version is available.
type UpdateAvailableMsg struct {
	CurrentVersion string
	LatestVersion  string
	UpdateCommand  string
}

// updateCommand generates the go install command for updating.
func updateCommand(version string) string {
	return fmt.Sprintf(
		"go install -ldflags \"-X main.Version=%s\" github.com/sst/sidecar/cmd/sidecar@%s",
		version, version,
	)
}

// CheckAsync returns a Bubble Tea command that checks for updates in background.
func CheckAsync(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		// Check cache first
		if cached, err := LoadCache(); err == nil && IsCacheValid(cached, currentVersion) {
			if cached.HasUpdate {
				return UpdateAvailableMsg{
					CurrentVersion: currentVersion,
					LatestVersion:  cached.LatestVersion,
					UpdateCommand:  updateCommand(cached.LatestVersion),
				}
			}
			return nil // up-to-date, cached
		}

		// Cache miss or invalid, fetch from GitHub
		result := Check(currentVersion)

		// Save to cache (ignore errors)
		_ = SaveCache(&CacheEntry{
			LatestVersion:  result.LatestVersion,
			CurrentVersion: currentVersion,
			CheckedAt:      time.Now(),
			HasUpdate:      result.HasUpdate,
		})

		if result.HasUpdate {
			return UpdateAvailableMsg{
				CurrentVersion: currentVersion,
				LatestVersion:  result.LatestVersion,
				UpdateCommand:  updateCommand(result.LatestVersion),
			}
		}

		return nil
	}
}
