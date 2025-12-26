package gitstatus

import (
	"os/exec"
	"strings"
)

// GetDiff returns the diff for a file.
func GetDiff(workDir, path string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// Try to get exit status - git diff returns 1 if there are changes
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return string(output), nil
			}
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetFullDiff returns the diff for all changes.
func GetFullDiff(workDir string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return string(output), nil
			}
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetFileDiffStats returns the +/- counts for a single file.
func GetFileDiffStats(workDir, path string, staged bool) (int, int, error) {
	args := []string{"diff", "--numstat"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	// Parse: <additions>\t<deletions>\t<path>
	line := strings.TrimSpace(string(output))
	if line == "" {
		return 0, 0, nil
	}

	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return 0, 0, nil
	}

	var additions, deletions int
	if parts[0] != "-" {
		_, _ = stringToInt(parts[0], &additions)
	}
	if parts[1] != "-" {
		_, _ = stringToInt(parts[1], &deletions)
	}

	return additions, deletions, nil
}

// stringToInt is a helper to parse int from string.
func stringToInt(s string, result *int) (bool, error) {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		*result = *result*10 + int(c-'0')
	}
	return true, nil
}
