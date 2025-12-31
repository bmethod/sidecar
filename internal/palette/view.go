package palette

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
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

// renderItem represents a single line in the palette (header or entry).
type renderItem struct {
	isHeader   bool
	layer      Layer
	entry      *PaletteEntry
	entryIndex int // index in filtered entries (for cursor matching)
}

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

	// Mode indicator
	var modeText string
	if m.showAllContexts {
		modeText = "[All Contexts]"
	} else {
		modeText = fmt.Sprintf("[%s]", m.activeContext)
	}
	toggleHint := styles.Muted.Render("tab to toggle")
	b.WriteString(fmt.Sprintf("%s  %s", styles.Muted.Render(modeText), toggleHint))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n")

	// Build flat list of render items
	items := m.buildRenderItems()
	totalEntries := len(m.filtered)

	// Calculate visible range based on entry indices
	visibleStart := m.offset
	visibleEnd := m.offset + m.maxVisible
	if visibleEnd > totalEntries {
		visibleEnd = totalEntries
	}

	// Show scroll-up indicator if content above
	if m.offset > 0 {
		b.WriteString(styles.Muted.Render(fmt.Sprintf("  ↑ %d more above", m.offset)))
		b.WriteString("\n")
	}

	// Render only visible items
	for _, item := range items {
		if item.isHeader {
			// Show header only if it has visible entries
			if m.layerHasVisibleEntries(item.layer, visibleStart, visibleEnd) {
				b.WriteString(m.renderLayerHeader(item.layer, m.countEntriesInLayer(item.layer)))
				b.WriteString("\n")
			}
		} else {
			// Only render entries within visible range
			if item.entryIndex >= visibleStart && item.entryIndex < visibleEnd {
				isSelected := item.entryIndex == m.cursor
				line := m.renderEntry(*item.entry, isSelected, width-4)
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
	}

	// Show scroll-down indicator if content below
	if visibleEnd < totalEntries {
		remaining := totalEntries - visibleEnd
		b.WriteString(styles.Muted.Render(fmt.Sprintf("  ↓ %d more below", remaining)))
		b.WriteString("\n")
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

// buildRenderItems creates a flat list of headers and entries for rendering.
func (m Model) buildRenderItems() []renderItem {
	groups := GroupEntriesByLayer(m.filtered)
	layers := []Layer{LayerCurrentMode, LayerPlugin, LayerGlobal}

	var items []renderItem
	entryIndex := 0

	for _, layer := range layers {
		entries, ok := groups[layer]
		if !ok || len(entries) == 0 {
			continue
		}

		// Add layer header
		items = append(items, renderItem{isHeader: true, layer: layer})

		// Add entries
		for i := range entries {
			items = append(items, renderItem{
				entry:      &entries[i],
				entryIndex: entryIndex,
			})
			entryIndex++
		}
	}

	return items
}

// layerHasVisibleEntries checks if a layer has any entries in the visible range.
func (m Model) layerHasVisibleEntries(layer Layer, visibleStart, visibleEnd int) bool {
	groups := GroupEntriesByLayer(m.filtered)
	layers := []Layer{LayerCurrentMode, LayerPlugin, LayerGlobal}

	entryIndex := 0
	for _, l := range layers {
		entries := groups[l]
		layerStart := entryIndex
		layerEnd := entryIndex + len(entries)

		if l == layer {
			// Check if any entries in this layer fall within visible range
			return layerStart < visibleEnd && layerEnd > visibleStart
		}
		entryIndex = layerEnd
	}
	return false
}

// countEntriesInLayer returns the count of entries in a specific layer.
func (m Model) countEntriesInLayer(layer Layer) int {
	groups := GroupEntriesByLayer(m.filtered)
	return len(groups[layer])
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

	// Show context count if command appears in multiple contexts
	if entry.ContextCount > 1 {
		desc = fmt.Sprintf("%s (%d contexts)", desc, entry.ContextCount)
	}

	if len(desc) > descWidth && descWidth > 3 {
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
