package palette

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sst/sidecar/internal/styles"
)

// Palette-specific styles
var (
	paletteBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary).
			Background(styles.BgSecondary).
			Padding(1, 2)

	paletteInput = lipgloss.NewStyle().
			Foreground(styles.TextPrimary).
			Background(styles.BgTertiary).
			Padding(0, 1).
			MarginBottom(1)

	layerHeaderCurrent = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Bold(true).
				MarginTop(1)

	layerHeaderPlugin = lipgloss.NewStyle().
				Foreground(styles.Secondary).
				Bold(true).
				MarginTop(1)

	layerHeaderGlobal = lipgloss.NewStyle().
				Foreground(styles.TextMuted).
				Bold(true).
				MarginTop(1)

	entryNormal = lipgloss.NewStyle().
			Foreground(styles.TextPrimary)

	entrySelected = lipgloss.NewStyle().
			Foreground(styles.TextPrimary).
			Background(styles.BgTertiary)

	entryKey = lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Width(12)

	entryName = lipgloss.NewStyle().
			Foreground(styles.TextPrimary).
			Width(20)

	entryDesc = lipgloss.NewStyle().
			Foreground(styles.TextSecondary)

	matchHighlight = lipgloss.NewStyle().
			Foreground(styles.Primary).
			Bold(true)

	escHint = lipgloss.NewStyle().
		Foreground(styles.TextMuted)
)

// View renders the command palette.
func (m Model) View() string {
	var b strings.Builder

	// Calculate width
	width := min(80, m.width-4)
	if width < 40 {
		width = 40
	}

	// Header with search input
	inputLine := fmt.Sprintf("> %s", m.textInput.View())
	escText := escHint.Render("[esc]")
	inputWidth := width - lipgloss.Width(escText) - 4
	paddedInput := lipgloss.NewStyle().Width(inputWidth).Render(inputLine)
	header := paddedInput + " " + escText
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n")

	// Group entries by layer
	groups := GroupEntriesByLayer(m.filtered)

	// Render each layer
	entryIndex := 0
	layers := []Layer{LayerCurrentMode, LayerPlugin, LayerGlobal}

	for _, layer := range layers {
		entries, ok := groups[layer]
		if !ok || len(entries) == 0 {
			continue
		}

		// Layer header
		header := m.renderLayerHeader(layer, len(entries))
		b.WriteString(header)
		b.WriteString("\n")

		// Render entries
		for _, entry := range entries {
			isSelected := entryIndex == m.cursor
			line := m.renderEntry(entry, isSelected, width-4)
			b.WriteString(line)
			b.WriteString("\n")
			entryIndex++
		}
	}

	// Empty state
	if len(m.filtered) == 0 {
		emptyMsg := styles.Muted.Render("No matching commands")
		b.WriteString("\n")
		b.WriteString(emptyMsg)
		b.WriteString("\n")
	}

	// Wrap in box
	content := strings.TrimRight(b.String(), "\n")
	box := paletteBox.Width(width).Render(content)

	return box
}

// renderLayerHeader renders a layer section header.
func (m Model) renderLayerHeader(layer Layer, count int) string {
	var style lipgloss.Style
	var name string

	switch layer {
	case LayerCurrentMode:
		style = layerHeaderCurrent
		name = strings.ToUpper(m.activeContext)
	case LayerPlugin:
		style = layerHeaderPlugin
		name = strings.ToUpper(m.pluginContext)
	case LayerGlobal:
		style = layerHeaderGlobal
		name = "GLOBAL"
	}

	return style.Render(name)
}

// renderEntry renders a single palette entry.
func (m Model) renderEntry(entry PaletteEntry, selected bool, maxWidth int) string {
	// Key column
	keyStr := entryKey.Render(entry.Key)

	// Name with match highlighting
	nameStr := m.highlightMatches(entry.Name, entry.MatchRanges)
	nameStr = entryName.Render(nameStr)

	// Description (truncate if needed)
	descWidth := maxWidth - 12 - 20 - 4
	desc := entry.Description
	if len(desc) > descWidth {
		desc = desc[:descWidth-3] + "..."
	}
	descStr := entryDesc.Render(desc)

	line := fmt.Sprintf("  %s %s %s", keyStr, nameStr, descStr)

	if selected {
		return entrySelected.Render(line)
	}
	return entryNormal.Render(line)
}

// highlightMatches applies highlighting to matched characters.
func (m Model) highlightMatches(text string, ranges []MatchRange) string {
	if len(ranges) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, r := range ranges {
		// Add non-matched part
		if r.Start > lastEnd {
			result.WriteString(text[lastEnd:r.Start])
		}
		// Add matched part with highlighting
		if r.End <= len(text) {
			result.WriteString(matchHighlight.Render(text[r.Start:r.End]))
		}
		lastEnd = r.End
	}

	// Add remaining text
	if lastEnd < len(text) {
		result.WriteString(text[lastEnd:])
	}

	return result.String()
}
