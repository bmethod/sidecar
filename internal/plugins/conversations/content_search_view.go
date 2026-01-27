// Package conversations provides the content search modal UI for
// cross-conversation search (searching message content across sessions).
package conversations

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marcus/sidecar/internal/adapter"
	"github.com/marcus/sidecar/internal/modal"
	"github.com/marcus/sidecar/internal/styles"
)

// Content search modal field IDs
const (
	contentSearchInputID = "content-search-input"
)

// renderContentSearchModal renders the content search modal.
// This creates a modal with search input, options, results, and stats sections.
func renderContentSearchModal(state *ContentSearchState, width, height int) string {
	// Calculate modal dimensions
	modalWidth := width - 8
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > 100 {
		modalWidth = 100
	}

	// Build modal using the modal package
	m := modal.New("Search conversations",
		modal.WithWidth(modalWidth),
		modal.WithHints(false),
	).
		AddSection(contentSearchHeaderSection(state, modalWidth-6)).
		AddSection(modal.Spacer()).
		AddSection(contentSearchOptionsSection(state)).
		AddSection(modal.Spacer()).
		AddSection(contentSearchResultsSection(state, height-14, modalWidth-6)).
		AddSection(modal.Spacer()).
		AddSection(contentSearchStatsSection(state))

	return m.Render(width, height, nil)
}

// contentSearchHeaderSection creates the search input header section.
func contentSearchHeaderSection(state *ContentSearchState, contentWidth int) modal.Section {
	return modal.Custom(
		func(cw int, focusID, hoverID string) modal.RenderedSection {
			var sb strings.Builder

			// Search prompt with query and cursor
			sb.WriteString(styles.Subtitle.Render("Search: "))

			query := state.Query
			if len(query) > contentWidth-12 {
				query = query[:contentWidth-15] + "..."
			}
			sb.WriteString(styles.Body.Render(query))
			sb.WriteString(styles.StatusInProgress.Render("\u2588")) // Block cursor

			// Show searching indicator
			if state.IsSearching {
				sb.WriteString("  ")
				sb.WriteString(styles.Muted.Render("Searching..."))
			}

			// Show error if present
			if state.Error != "" {
				sb.WriteString("\n")
				errMsg := state.Error
				if len(errMsg) > contentWidth {
					errMsg = errMsg[:contentWidth-3] + "..."
				}
				sb.WriteString(styles.StatusDeleted.Render(errMsg))
			}

			return modal.RenderedSection{Content: sb.String()}
		},
		nil, // No update handler needed
	)
}

// contentSearchOptionsSection creates the search options toggle section.
func contentSearchOptionsSection(state *ContentSearchState) modal.Section {
	return modal.Custom(
		func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
			var sb strings.Builder

			// Regex toggle
			regexStyle := styles.Muted
			if state.UseRegex {
				regexStyle = styles.StatusInProgress
			}
			sb.WriteString(regexStyle.Render("[.*]"))
			sb.WriteString(styles.Subtle.Render(" regex"))

			sb.WriteString("  ")

			// Case sensitivity toggle
			caseStyle := styles.Muted
			if state.CaseSensitive {
				caseStyle = styles.StatusInProgress
			}
			sb.WriteString(caseStyle.Render("[Aa]"))
			sb.WriteString(styles.Subtle.Render(" case"))

			sb.WriteString("  ")
			sb.WriteString(styles.Subtle.Render("(ctrl+r / alt+c to toggle)"))

			return modal.RenderedSection{Content: sb.String()}
		},
		nil,
	)
}

// contentSearchResultsSection creates the scrollable results section.
func contentSearchResultsSection(state *ContentSearchState, viewportHeight, contentWidth int) modal.Section {
	return modal.Custom(
		func(cw int, focusID, hoverID string) modal.RenderedSection {
			if viewportHeight < 1 {
				viewportHeight = 1
			}
			if contentWidth < 20 {
				contentWidth = cw
			}

			// Handle empty states
			if len(state.Results) == 0 {
				if state.Query == "" {
					return modal.RenderedSection{Content: styles.Muted.Render("Enter a search query...")}
				}
				if state.IsSearching {
					return modal.RenderedSection{Content: styles.Muted.Render("Searching...")}
				}
				return modal.RenderedSection{Content: styles.Muted.Render("No matches found")}
			}

			// Build all result lines
			var allLines []string
			flatIdx := 0

			for si, sr := range state.Results {
				// Session header row
				selected := flatIdx == state.Cursor
				sessionLine := renderSessionHeader(sr, selected, contentWidth)
				allLines = append(allLines, sessionLine)
				flatIdx++

				// Skip children if collapsed
				if sr.Collapsed {
					continue
				}

				// Message rows
				for mi, msg := range sr.Messages {
					msgSelected := flatIdx == state.Cursor
					msgLine := renderMessageHeader(msg, msgSelected, contentWidth)
					allLines = append(allLines, msgLine)
					flatIdx++

					// Match rows
					for mti, match := range msg.Matches {
						matchSelected := flatIdx == state.Cursor
						matchLine := renderMatchLine(match, state.Query, matchSelected, contentWidth, si, mi, mti)
						allLines = append(allLines, matchLine)
						flatIdx++
					}
				}
			}

			// Apply scroll offset
			maxScroll := len(allLines) - viewportHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			scrollOffset := state.ScrollOffset
			if scrollOffset > maxScroll {
				scrollOffset = maxScroll
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}

			// Slice to viewport
			start := scrollOffset
			end := start + viewportHeight
			if end > len(allLines) {
				end = len(allLines)
			}

			visibleLines := allLines[start:end]

			// Add scroll indicators if needed
			var result strings.Builder
			if scrollOffset > 0 {
				result.WriteString(styles.Muted.Render(fmt.Sprintf("\u2191 %d more above", scrollOffset)))
				result.WriteString("\n")
			}

			for i, line := range visibleLines {
				result.WriteString(line)
				if i < len(visibleLines)-1 {
					result.WriteString("\n")
				}
			}

			remaining := len(allLines) - end
			if remaining > 0 {
				result.WriteString("\n")
				result.WriteString(styles.Muted.Render(fmt.Sprintf("\u2193 %d more below", remaining)))
			}

			return modal.RenderedSection{Content: result.String()}
		},
		nil,
	)
}

// contentSearchStatsSection creates the summary stats section.
func contentSearchStatsSection(state *ContentSearchState) modal.Section {
	return modal.Custom(
		func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
			var sb strings.Builder

			// Stats line
			totalMatches := state.TotalMatches()
			sessionCount := state.SessionCount()

			if totalMatches > 0 {
				statsText := fmt.Sprintf("%d matches in %d sessions", totalMatches, sessionCount)
				sb.WriteString(styles.Subtitle.Render(statsText))
				sb.WriteString("  ")
			}

			// Navigation hints
			hints := "[j/k nav] [enter select] [space expand/collapse] [esc close]"
			if contentWidth < 60 {
				hints = "[j/k] [enter] [space] [esc]"
			}
			sb.WriteString(styles.Muted.Render(hints))

			return modal.RenderedSection{Content: sb.String()}
		},
		nil,
	)
}

// renderSessionHeader renders a session row in the results.
// Format: [chevron] "Session title" (icon adapter) time ago  (count)
func renderSessionHeader(sr SessionSearchResult, selected bool, maxWidth int) string {
	var sb strings.Builder

	// Collapse indicator
	chevron := "\u25bc" // Down-pointing triangle (expanded)
	if sr.Collapsed {
		chevron = "\u25b6" // Right-pointing triangle (collapsed)
	}

	// Session name
	name := sr.Session.Name
	if name == "" {
		name = sr.Session.Slug
	}
	if name == "" && len(sr.Session.ID) > 12 {
		name = sr.Session.ID[:12]
	} else if name == "" {
		name = sr.Session.ID
	}

	// Adapter badge
	adapterBadge := ""
	if sr.Session.AdapterIcon != "" {
		adapterBadge = sr.Session.AdapterIcon
	} else if sr.Session.AdapterID != "" {
		// Use first char or known abbreviation
		switch sr.Session.AdapterID {
		case "claude-code":
			adapterBadge = "\u25c6" // Diamond
		case "codex":
			adapterBadge = "C"
		case "gemini-cli":
			adapterBadge = "G"
		default:
			if len(sr.Session.AdapterID) > 0 {
				adapterBadge = string([]rune(sr.Session.AdapterID)[0])
			}
		}
	}

	// Time ago
	timeAgo := formatTimeAgo(sr.Session.UpdatedAt)

	// Match count
	matchCount := 0
	for _, msg := range sr.Messages {
		matchCount += len(msg.Matches)
	}
	countStr := fmt.Sprintf("(%d)", matchCount)

	// Calculate available width for name
	// chevron(1) + space(1) + quote(1) + name + quote(1) + space(1) + paren(1) + badge + paren(1) + space(1) + time + space(2) + count
	fixedWidth := 1 + 1 + 1 + 1 + 1 + 1 + len(adapterBadge) + 1 + 1 + len(timeAgo) + 2 + len(countStr)
	nameWidth := maxWidth - fixedWidth
	if nameWidth < 10 {
		nameWidth = 10
	}

	// Truncate name if needed (rune-safe)
	if runes := []rune(name); len(runes) > nameWidth {
		name = string(runes[:nameWidth-3]) + "..."
	}

	// Build content
	if selected {
		// Plain text for selected row with background highlight
		content := fmt.Sprintf("%s \"%s\" (%s) %s  %s",
			chevron, name, adapterBadge, timeAgo, countStr)

		// Pad to full width
		if ansi.StringWidth(content) < maxWidth {
			content += strings.Repeat(" ", maxWidth-ansi.StringWidth(content))
		}
		return styles.ListItemSelected.Render(content)
	}

	// Styled content for unselected row
	sb.WriteString(styles.Muted.Render(chevron))
	sb.WriteString(" ")
	sb.WriteString(styles.Title.Render("\"" + name + "\""))
	sb.WriteString(" ")
	sb.WriteString(styles.Code.Render("(" + adapterBadge + ")"))
	sb.WriteString(" ")
	sb.WriteString(styles.Subtle.Render(timeAgo))
	sb.WriteString("  ")
	sb.WriteString(styles.Muted.Render(countStr))

	return sb.String()
}

// renderMessageHeader renders a message row in the results.
// Format:     [Role] HH:MM "Preview text..."
func renderMessageHeader(msg adapter.MessageMatch, selected bool, maxWidth int) string {
	var sb strings.Builder

	indent := "    " // 4 spaces for messages under sessions

	// Role badge
	role := msg.Role
	if len(role) > 8 {
		role = role[:8]
	}
	roleBadge := fmt.Sprintf("[%s]", strings.Title(role))

	// Timestamp
	timestamp := msg.Timestamp.Local().Format("15:04")

	// Preview text from first match
	preview := ""
	if len(msg.Matches) > 0 {
		preview = msg.Matches[0].LineText
		preview = strings.TrimSpace(preview)
		preview = strings.ReplaceAll(preview, "\n", " ")
	}

	// Calculate available width for preview
	// indent(4) + role(~10) + space(1) + timestamp(5) + space(1) + quote(2) + preview
	fixedWidth := len(indent) + len(roleBadge) + 1 + len(timestamp) + 1 + 2
	previewWidth := maxWidth - fixedWidth
	if previewWidth < 10 {
		previewWidth = 10
	}

	// Truncate preview if needed (rune-safe)
	if runes := []rune(preview); len(runes) > previewWidth {
		preview = string(runes[:previewWidth-3]) + "..."
	}

	if selected {
		// Plain text for selected row
		content := fmt.Sprintf("%s%s %s \"%s\"",
			indent, roleBadge, timestamp, preview)

		// Pad to full width
		if ansi.StringWidth(content) < maxWidth {
			content += strings.Repeat(" ", maxWidth-ansi.StringWidth(content))
		}
		return styles.ListItemSelected.Render(content)
	}

	// Styled content
	sb.WriteString(indent)

	// Role with color based on type
	roleStyle := styles.StatusStaged // Default for assistant
	if msg.Role == "user" {
		roleStyle = styles.StatusInProgress
	}
	sb.WriteString(roleStyle.Render(roleBadge))
	sb.WriteString(" ")
	sb.WriteString(styles.Muted.Render(timestamp))
	sb.WriteString(" ")
	sb.WriteString(styles.Body.Render("\"" + preview + "\""))

	return sb.String()
}

// renderMatchLine renders a single match line within a message.
// Format:       |  Line N: ...text with **highlighted** match...
func renderMatchLine(match adapter.ContentMatch, _ string, selected bool, maxWidth int, _, _, _ int) string {
	var sb strings.Builder

	indent := "      " // 6 spaces for matches under messages
	linePrefix := fmt.Sprintf("\u2502  Line %d: ", match.LineNo)

	// Get the line text and highlight the match
	lineText := match.LineText
	lineText = strings.TrimSpace(lineText)
	lineText = strings.ReplaceAll(lineText, "\n", " ")

	// Calculate available width for content
	fixedWidth := len(indent) + ansi.StringWidth(linePrefix)
	contentWidth := maxWidth - fixedWidth
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Convert byte indices to rune indices for proper UTF-8 handling
	// match.ColStart and match.ColEnd are byte indices from regexp
	runes := []rune(lineText)
	runeLen := len(runes)

	colStart := byteToRuneIndex(lineText, match.ColStart)
	colEnd := byteToRuneIndex(lineText, match.ColEnd)

	// Apply context window around match
	displayRunes := runes
	displayText := lineText

	// If line is too long, show context around the match
	if runeLen > contentWidth {
		// Calculate context window
		contextBefore := 15
		matchLen := colEnd - colStart
		contextAfter := contentWidth - matchLen - contextBefore - 6 // 6 for "..."
		if contextAfter < 10 {
			contextAfter = 10
		}

		start := colStart - contextBefore
		end := colEnd + contextAfter

		prefix := ""
		suffix := ""

		if start < 0 {
			start = 0
		} else {
			prefix = "..."
		}

		if end > runeLen {
			end = runeLen
		} else {
			suffix = "..."
		}

		// Adjust column positions for the new substring
		newColStart := colStart - start + utf8.RuneCountInString(prefix)
		newColEnd := colEnd - start + utf8.RuneCountInString(prefix)

		displayRunes = runes[start:end]
		displayText = prefix + string(displayRunes) + suffix
		colStart = newColStart
		colEnd = newColEnd

		// Clamp to valid range
		if colStart < 0 {
			colStart = 0
		}
		displayRuneLen := utf8.RuneCountInString(displayText)
		if colEnd > displayRuneLen {
			colEnd = displayRuneLen
		}
	}

	// Final truncation if still too long
	displayRuneLen := utf8.RuneCountInString(displayText)
	if displayRuneLen > contentWidth {
		displayRunes := []rune(displayText)
		displayText = string(displayRunes[:contentWidth-3]) + "..."
		displayRuneLen = utf8.RuneCountInString(displayText)
		if colEnd > displayRuneLen {
			colEnd = displayRuneLen
		}
	}

	if selected {
		// Plain text for selected row
		content := fmt.Sprintf("%s%s%s", indent, linePrefix, displayText)

		// Pad to full width
		if ansi.StringWidth(content) < maxWidth {
			content += strings.Repeat(" ", maxWidth-ansi.StringWidth(content))
		}
		return styles.ListItemSelected.Render(content)
	}

	// Styled content with highlighted match
	sb.WriteString(indent)
	sb.WriteString(styles.Muted.Render(linePrefix))

	// Highlight the matched portion (using rune indices)
	highlightedText := highlightMatchRunes(displayText, colStart, colEnd)
	sb.WriteString(highlightedText)

	return sb.String()
}

// highlightMatchRunes adds styling to the matched portion of text using rune indices.
// This is safe for multi-byte UTF-8 characters (emojis, CJK, etc.).
// colStart and colEnd are rune indices, not byte indices.
func highlightMatchRunes(text string, colStart, colEnd int) string {
	runes := []rune(text)
	runeLen := len(runes)

	if colStart < 0 || colEnd < 0 || colStart >= runeLen || colEnd > runeLen || colStart >= colEnd {
		// Invalid range, return text with muted styling
		return styles.Muted.Render(text)
	}

	var sb strings.Builder

	// Before match
	if colStart > 0 {
		sb.WriteString(styles.Muted.Render(string(runes[:colStart])))
	}

	// Matched portion with highlight
	matchStyle := lipgloss.NewStyle().
		Background(styles.Warning).   // Yellow/amber background
		Foreground(styles.BgPrimary). // Dark text for contrast
		Bold(true)
	sb.WriteString(matchStyle.Render(string(runes[colStart:colEnd])))

	// After match
	if colEnd < runeLen {
		sb.WriteString(styles.Muted.Render(string(runes[colEnd:])))
	}

	return sb.String()
}

// formatTimeAgo formats a time as a human-readable "X ago" string.
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	}
	if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%dd ago", days)
	}
	if d < 30*24*time.Hour {
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	}

	// Older than a month, show date
	return t.Local().Format("Jan 02")
}

// byteToRuneIndex converts a byte index to a rune index in a string.
// If the byte index is invalid or points to the middle of a multi-byte character,
// it returns the nearest valid rune index.
func byteToRuneIndex(s string, byteIdx int) int {
	if byteIdx <= 0 {
		return 0
	}
	if byteIdx >= len(s) {
		return utf8.RuneCountInString(s)
	}
	return utf8.RuneCountInString(s[:byteIdx])
}

