// Package conversations provides content search keyboard handling for
// cross-conversation search (td-6ac70a).
package conversations

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/app"
	"github.com/marcus/sidecar/internal/plugin"
)

// handleContentSearchKey handles keyboard input when in content search mode.
func (p *Plugin) handleContentSearchKey(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		p.contentSearchMode = false
		p.contentSearchState = nil
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return p, p.jumpToSearchResult()

	case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
		if p.contentSearchState != nil {
			p.contentSearchState.Cursor++
			p.contentSearchState.ClampCursor()
			p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
		if p.contentSearchState != nil {
			p.contentSearchState.Cursor--
			p.contentSearchState.ClampCursor()
			p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		// Jump to next match (skips session and message rows)
		if p.contentSearchState != nil {
			nextIdx := p.contentSearchState.NextMatchIndex(p.contentSearchState.Cursor)
			if nextIdx >= 0 {
				p.contentSearchState.Cursor = nextIdx
				p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
			}
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("N"))):
		// Jump to previous match (skips session and message rows)
		if p.contentSearchState != nil {
			prevIdx := p.contentSearchState.PrevMatchIndex(p.contentSearchState.Cursor)
			if prevIdx >= 0 {
				p.contentSearchState.Cursor = prevIdx
				p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
			}
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		// Move to session (navigate up in hierarchy)
		if p.contentSearchState != nil {
			p.contentSearchState.MoveToSession()
			p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", " "))):
		// Toggle collapse/expand for session
		if p.contentSearchState != nil {
			p.contentSearchState.ToggleCollapse()
			p.contentSearchState.ClampCursor()
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+r", "alt+r"))):
		// Toggle regex mode
		if p.contentSearchState != nil {
			p.contentSearchState.UseRegex = !p.contentSearchState.UseRegex
			return p, p.triggerContentSearch()
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("alt+c"))):
		// Toggle case sensitivity
		if p.contentSearchState != nil {
			p.contentSearchState.CaseSensitive = !p.contentSearchState.CaseSensitive
			return p, p.triggerContentSearch()
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
		// Page down
		if p.contentSearchState != nil {
			p.contentSearchState.Cursor += 10
			p.contentSearchState.ClampCursor()
			p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
		// Page up
		if p.contentSearchState != nil {
			p.contentSearchState.Cursor -= 10
			p.contentSearchState.ClampCursor()
			p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		// Go to top
		if p.contentSearchState != nil {
			p.contentSearchState.Cursor = 0
			p.contentSearchState.ScrollOffset = 0
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
		// Go to bottom
		if p.contentSearchState != nil {
			flatLen := p.contentSearchState.FlatLen()
			if flatLen > 0 {
				p.contentSearchState.Cursor = flatLen - 1
				p.contentSearchState.EnsureCursorVisible(p.contentSearchViewportHeight())
			}
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("E"))):
		// Expand all sessions
		if p.contentSearchState != nil {
			p.contentSearchState.ExpandAll()
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("C"))):
		// Collapse all sessions
		if p.contentSearchState != nil {
			p.contentSearchState.CollapseAll()
			p.contentSearchState.ClampCursor()
		}
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
		// Delete character from query
		if p.contentSearchState != nil && len(p.contentSearchState.Query) > 0 {
			runes := []rune(p.contentSearchState.Query)
			p.contentSearchState.Query = string(runes[:len(runes)-1])
			return p, p.triggerContentSearch()
		}
		return p, nil

	case msg.Type == tea.KeyRunes:
		// Add character to query
		if p.contentSearchState != nil {
			p.contentSearchState.Query += string(msg.Runes)
			return p, p.triggerContentSearch()
		}
		return p, nil
	}

	return p, nil
}

// openContentSearch opens the content search modal.
func (p *Plugin) openContentSearch() (plugin.Plugin, tea.Cmd) {
	p.contentSearchMode = true
	p.contentSearchState = NewContentSearchState()
	p.hitRegionsDirty = true
	return p, nil
}

// triggerContentSearch initiates a debounced search.
func (p *Plugin) triggerContentSearch() tea.Cmd {
	if p.contentSearchState == nil {
		return nil
	}
	p.contentSearchState.DebounceVersion++
	p.contentSearchState.IsSearching = true
	p.contentSearchState.Error = ""
	return scheduleContentSearch(p.contentSearchState.Query, p.contentSearchState.DebounceVersion)
}

// contentSearchViewportHeight returns the viewport height for the content search results.
func (p *Plugin) contentSearchViewportHeight() int {
	// Modal height minus header, options, stats sections
	return p.height - 14
}

// scrollToMessageMsg is used to scroll to a specific message after loading.
type scrollToMessageMsg struct {
	MessageIdx int
}

// jumpToSearchResult selects the session and message from the current search result.
func (p *Plugin) jumpToSearchResult() tea.Cmd {
	if p.contentSearchState == nil {
		return nil
	}

	session, msgMatch, _ := p.contentSearchState.GetSelectedResult()
	if session == nil {
		return nil
	}

	// Close search modal
	p.contentSearchMode = false

	// Find the session in our list and select it
	found := false
	for i := range p.sessions {
		if p.sessions[i].ID == session.ID {
			p.cursor = i
			p.ensureCursorVisible()
			found = true
			break
		}
	}

	if !found {
		// Session not in current list (shouldn't happen, but handle gracefully)
		p.contentSearchState = nil
		return func() tea.Msg {
			return app.ToastMsg{Message: "Session not found", Duration: 2 * time.Second, IsError: true}
		}
	}

	// Set selected session and switch to messages pane
	p.setSelectedSession(session.ID)
	p.activePane = PaneMessages
	p.contentSearchState = nil

	// Build commands to load messages
	cmds := []tea.Cmd{
		p.loadMessages(session.ID),
		p.loadUsage(session.ID),
	}

	// If we have a specific message to jump to, scroll to it after loading
	if msgMatch != nil {
		targetMsgIdx := msgMatch.MessageIdx
		cmds = append(cmds, func() tea.Msg {
			return scrollToMessageMsg{MessageIdx: targetMsgIdx}
		})
	}

	return tea.Batch(cmds...)
}
