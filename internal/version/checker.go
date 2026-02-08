package version

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// UpdateAvailableMsg is sent when a new sidecar version is available.
type UpdateAvailableMsg struct {
	CurrentVersion string
	LatestVersion  string
	UpdateCommand  string
	ReleaseNotes   string
	ReleaseURL     string
	InstallMethod  InstallMethod
}

// TdVersionMsg is sent with td version info (installed or not).
type TdVersionMsg struct {
	Installed      bool
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
}

// updateCommand generates the update command based on install method.
func updateCommand(version string, method InstallMethod) string {
	switch method {
	case InstallMethodHomebrew:
		return "brew upgrade sidecar"
	case InstallMethodBinary:
		return fmt.Sprintf("https://github.com/marcus/sidecar/releases/tag/%s", version)
	default:
		return fmt.Sprintf(
			"go install -ldflags \"-X main.Version=%s\" github.com/marcus/sidecar/cmd/sidecar@%s",
			version, version,
		)
	}
}

// CheckAsync returns a Bubble Tea command that checks for updates in background.
// Compares upstream releases against the last known upstream version (from cache)
// rather than the local build version, so source builds with custom version
// strings don't trigger false "update available" notifications.
func CheckAsync(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		method := DetectInstallMethod()

		// Load existing cache for baseline comparison
		cached, cacheErr := LoadCache()

		// If cache is valid (not expired), use cached result
		if cacheErr == nil && IsCacheValid(cached) {
			if cached.HasUpdate {
				return UpdateAvailableMsg{
					CurrentVersion: currentVersion,
					LatestVersion:  cached.LatestVersion,
					UpdateCommand:  updateCommand(cached.LatestVersion, method),
					InstallMethod:  method,
				}
			}
			return nil // up-to-date, cached
		}

		// Determine if we have a prior baseline (previous upstream version).
		// Without a baseline (first run), we just establish one without notifying.
		hasPriorBaseline := cacheErr == nil && cached != nil && cached.LatestVersion != ""

		// Fetch latest release from GitHub
		result := FetchLatestRelease(repoOwner, repoName)

		if result.Error == nil {
			hasUpdate := false
			if hasPriorBaseline {
				hasUpdate = isNewer(result.LatestVersion, cached.LatestVersion)
			}

			_ = SaveCache(&CacheEntry{
				LatestVersion:  result.LatestVersion,
				CurrentVersion: currentVersion,
				CheckedAt:      time.Now(),
				HasUpdate:      hasUpdate,
			})

			if hasUpdate {
				return UpdateAvailableMsg{
					CurrentVersion: currentVersion,
					LatestVersion:  result.LatestVersion,
					UpdateCommand:  updateCommand(result.LatestVersion, method),
					ReleaseNotes:   result.ReleaseNotes,
					ReleaseURL:     result.UpdateURL,
					InstallMethod:  method,
				}
			}
		}

		return nil
	}
}

// ForceCheckAsync checks for updates, ignoring the cache TTL.
// Still uses cached upstream version as baseline for comparison.
func ForceCheckAsync(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		method := DetectInstallMethod()

		// Use cached upstream version as baseline if available
		cached, cacheErr := LoadCache()
		hasPriorBaseline := cacheErr == nil && cached != nil && cached.LatestVersion != ""

		result := FetchLatestRelease(repoOwner, repoName)
		if result.Error == nil {
			hasUpdate := false
			if hasPriorBaseline {
				hasUpdate = isNewer(result.LatestVersion, cached.LatestVersion)
			}

			_ = SaveCache(&CacheEntry{
				LatestVersion:  result.LatestVersion,
				CurrentVersion: currentVersion,
				CheckedAt:      time.Now(),
				HasUpdate:      hasUpdate,
			})

			if hasUpdate {
				return UpdateAvailableMsg{
					CurrentVersion: currentVersion,
					LatestVersion:  result.LatestVersion,
					UpdateCommand:  updateCommand(result.LatestVersion, method),
					ReleaseNotes:   result.ReleaseNotes,
					ReleaseURL:     result.UpdateURL,
					InstallMethod:  method,
				}
			}
		}
		return nil
	}
}

// tdUpdateCommand generates the update command for td based on install method.
func tdUpdateCommand(version string, method InstallMethod) string {
	switch method {
	case InstallMethodHomebrew:
		return "brew upgrade td"
	default:
		return fmt.Sprintf(
			"go install github.com/marcus/td@%s",
			version,
		)
	}
}

// GetTdVersion returns the installed td version by running `td version --short`.
// Returns empty string if td is not installed or command fails.
func GetTdVersion() string {
	out, err := exec.Command("td", "version", "--short").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CheckTdAsync returns a Bubble Tea command that checks td version in background.
// Returns TdVersionMsg with installation status and version info.
func CheckTdAsync() tea.Cmd {
	return func() tea.Msg {
		tdVersion := GetTdVersion()

		// td not installed
		if tdVersion == "" {
			return TdVersionMsg{Installed: false}
		}

		// Load existing cache for baseline comparison
		cached, cacheErr := LoadTdCache()

		// If cache is valid, use cached result
		if cacheErr == nil && IsCacheValid(cached) {
			return TdVersionMsg{
				Installed:      true,
				CurrentVersion: tdVersion,
				LatestVersion:  cached.LatestVersion,
				HasUpdate:      cached.HasUpdate,
			}
		}

		// Use cached upstream version as baseline if available
		hasPriorBaseline := cacheErr == nil && cached != nil && cached.LatestVersion != ""

		// Cache miss or expired, fetch from GitHub
		result := FetchLatestRelease(tdRepoOwner, tdRepoName)

		hasUpdate := false
		if result.Error == nil {
			if hasPriorBaseline {
				hasUpdate = isNewer(result.LatestVersion, cached.LatestVersion)
			}
			_ = SaveTdCache(&CacheEntry{
				LatestVersion:  result.LatestVersion,
				CurrentVersion: tdVersion,
				CheckedAt:      time.Now(),
				HasUpdate:      hasUpdate,
			})
		}

		return TdVersionMsg{
			Installed:      true,
			CurrentVersion: tdVersion,
			LatestVersion:  result.LatestVersion,
			HasUpdate:      hasUpdate,
		}
	}
}

// ForceCheckTdAsync checks for td updates, ignoring the cache TTL.
func ForceCheckTdAsync() tea.Cmd {
	return func() tea.Msg {
		tdVersion := GetTdVersion()
		if tdVersion == "" {
			return TdVersionMsg{Installed: false}
		}

		// Use cached upstream version as baseline if available
		cached, cacheErr := LoadTdCache()
		hasPriorBaseline := cacheErr == nil && cached != nil && cached.LatestVersion != ""

		result := FetchLatestRelease(tdRepoOwner, tdRepoName)
		hasUpdate := false
		if result.Error == nil {
			if hasPriorBaseline {
				hasUpdate = isNewer(result.LatestVersion, cached.LatestVersion)
			}
			_ = SaveTdCache(&CacheEntry{
				LatestVersion:  result.LatestVersion,
				CurrentVersion: tdVersion,
				CheckedAt:      time.Now(),
				HasUpdate:      hasUpdate,
			})
		}
		return TdVersionMsg{
			Installed:      true,
			CurrentVersion: tdVersion,
			LatestVersion:  result.LatestVersion,
			HasUpdate:      hasUpdate,
		}
	}
}
