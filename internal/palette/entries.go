package palette

import (
	"strings"

	"github.com/sst/sidecar/internal/keymap"
	"github.com/sst/sidecar/internal/plugin"
)

// Layer represents the contextual hierarchy of a command.
type Layer int

const (
	LayerCurrentMode Layer = iota // e.g., "git-diff" specific
	LayerPlugin                   // e.g., "git-status" base
	LayerGlobal                   // Global shortcuts
)

// LayerName returns a display name for the layer.
func (l Layer) Name() string {
	switch l {
	case LayerCurrentMode:
		return "Current"
	case LayerPlugin:
		return "Plugin"
	case LayerGlobal:
		return "Global"
	default:
		return "Unknown"
	}
}

// PaletteEntry represents a single searchable entry in the command palette.
type PaletteEntry struct {
	Key         string          // Display key(s): "s" or "ctrl+s"
	CommandID   string          // Command ID
	Name        string          // Short name
	Description string          // Full description
	Category    plugin.Category // Category for grouping
	Context     string          // Source context
	Layer       Layer           // Which layer: CurrentMode, Plugin, Global
	Score       int             // Fuzzy match score (computed during search)
	MatchRanges []MatchRange    // For highlighting matches in name
}

// BuildEntries aggregates commands from keymap bindings and plugin commands.
// activeContext is the current focus context (e.g., "git-diff").
// pluginContext is the base plugin context (e.g., "git-status").
func BuildEntries(km *keymap.Registry, plugins []plugin.Plugin, activeContext, pluginContext string) []PaletteEntry {
	// Build a map of command metadata from plugins
	cmdMeta := make(map[string]plugin.Command)
	for _, p := range plugins {
		for _, cmd := range p.Commands() {
			cmdMeta[cmd.ID] = cmd
		}
	}

	// Collect all unique contexts
	contexts := km.AllContexts()

	// Map to deduplicate entries by command ID per context
	seen := make(map[string]bool)
	var entries []PaletteEntry

	for _, ctx := range contexts {
		bindings := km.BindingsForContext(ctx)
		for _, b := range bindings {
			key := b.Command + ":" + b.Context
			if seen[key] {
				continue
			}
			seen[key] = true

			entry := bindingToEntry(b, cmdMeta, activeContext, pluginContext)
			entries = append(entries, entry)
		}
	}

	return entries
}

// bindingToEntry converts a keymap binding to a palette entry.
func bindingToEntry(b keymap.Binding, cmdMeta map[string]plugin.Command, activeContext, pluginContext string) PaletteEntry {
	entry := PaletteEntry{
		Key:       b.Key,
		CommandID: b.Command,
		Context:   b.Context,
		Layer:     determineLayer(b.Context, activeContext, pluginContext),
	}

	// Try to get metadata from plugin command
	if cmd, ok := cmdMeta[b.Command]; ok {
		entry.Name = cmd.Name
		entry.Description = cmd.Description
		entry.Category = cmd.Category
	}

	// Fallback: generate name from command ID
	if entry.Name == "" {
		entry.Name = formatCommandID(b.Command)
	}

	// Fallback: generate description from command ID
	if entry.Description == "" {
		entry.Description = formatCommandID(b.Command)
	}

	// Fallback: infer category from command ID
	if entry.Category == "" {
		entry.Category = inferCategory(b.Command)
	}

	return entry
}

// determineLayer determines which layer a binding belongs to.
func determineLayer(bindingContext, activeContext, pluginContext string) Layer {
	if bindingContext == activeContext {
		return LayerCurrentMode
	}
	if bindingContext == pluginContext || strings.HasPrefix(activeContext, bindingContext+"-") {
		return LayerPlugin
	}
	if bindingContext == "global" {
		return LayerGlobal
	}
	// Default to plugin layer for non-global, non-current contexts
	return LayerPlugin
}

// formatCommandID converts a command ID to a readable name.
// "stage-file" -> "Stage file"
func formatCommandID(id string) string {
	words := strings.Split(id, "-")
	if len(words) == 0 {
		return id
	}
	// Capitalize first word
	words[0] = strings.Title(words[0])
	return strings.Join(words, " ")
}

// inferCategory infers a category from a command ID.
func inferCategory(cmdID string) plugin.Category {
	lower := strings.ToLower(cmdID)

	switch {
	case strings.Contains(lower, "scroll") ||
		strings.Contains(lower, "cursor") ||
		strings.Contains(lower, "next") ||
		strings.Contains(lower, "prev") ||
		strings.Contains(lower, "top") ||
		strings.Contains(lower, "bottom") ||
		strings.Contains(lower, "focus"):
		return plugin.CategoryNavigation

	case strings.Contains(lower, "search") ||
		strings.Contains(lower, "find"):
		return plugin.CategorySearch

	case strings.Contains(lower, "view") ||
		strings.Contains(lower, "show") ||
		strings.Contains(lower, "toggle") ||
		strings.Contains(lower, "diff"):
		return plugin.CategoryView

	case strings.Contains(lower, "stage") ||
		strings.Contains(lower, "unstage") ||
		strings.Contains(lower, "commit") ||
		strings.Contains(lower, "push") ||
		strings.Contains(lower, "pull") ||
		strings.Contains(lower, "history"):
		return plugin.CategoryGit

	case strings.Contains(lower, "edit") ||
		strings.Contains(lower, "delete") ||
		strings.Contains(lower, "add") ||
		strings.Contains(lower, "remove"):
		return plugin.CategoryEdit

	case strings.Contains(lower, "quit") ||
		strings.Contains(lower, "refresh") ||
		strings.Contains(lower, "help"):
		return plugin.CategorySystem

	default:
		return plugin.CategoryActions
	}
}

// GroupEntriesByLayer groups entries by their layer for display.
func GroupEntriesByLayer(entries []PaletteEntry) map[Layer][]PaletteEntry {
	groups := make(map[Layer][]PaletteEntry)
	for _, e := range entries {
		groups[e.Layer] = append(groups[e.Layer], e)
	}
	return groups
}
