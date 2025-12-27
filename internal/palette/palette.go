package palette

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sst/sidecar/internal/keymap"
	"github.com/sst/sidecar/internal/plugin"
)

// CommandSelectedMsg is sent when a command is selected from the palette.
type CommandSelectedMsg struct {
	CommandID string
	Context   string
}

// Model is the command palette state.
type Model struct {
	// Input state
	textInput textinput.Model

	// Entries
	allEntries []PaletteEntry
	filtered   []PaletteEntry
	cursor     int
	offset     int // scroll offset for virtual scrolling

	// Display
	width      int
	height     int
	maxVisible int

	// Context
	activeContext string
	pluginContext string

	// Dependencies
	keymap  *keymap.Registry
	plugins []plugin.Plugin
}

// New creates a new command palette model.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Search commands..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	return Model{
		textInput:  ti,
		maxVisible: 15,
	}
}

// Init initializes the palette (no-op for now).
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize updates the palette dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve space for input, borders, layer headers
	m.maxVisible = max(5, (height-10)/2)
	m.textInput.Width = min(50, width-10)
}

// Open prepares the palette for display.
// Rebuilds entries based on current context.
func (m *Model) Open(km *keymap.Registry, plugins []plugin.Plugin, activeContext, pluginContext string) {
	m.keymap = km
	m.plugins = plugins
	m.activeContext = activeContext
	m.pluginContext = pluginContext

	// Rebuild entries
	m.allEntries = BuildEntries(km, plugins, activeContext, pluginContext)
	m.filtered = FilterEntries(m.allEntries, "")

	// Reset state
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.cursor = 0
	m.offset = 0
}

// Query returns the current search query.
func (m Model) Query() string {
	return m.textInput.Value()
}

// Filtered returns the currently filtered entries.
func (m Model) Filtered() []PaletteEntry {
	return m.filtered
}

// Cursor returns the current cursor position.
func (m Model) Cursor() int {
	return m.cursor
}

// Offset returns the scroll offset.
func (m Model) Offset() int {
	return m.offset
}

// MaxVisible returns max visible entries.
func (m Model) MaxVisible() int {
	return m.maxVisible
}

// SelectedEntry returns the currently selected entry, if any.
func (m Model) SelectedEntry() *PaletteEntry {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return &m.filtered[m.cursor]
	}
	return nil
}

// Update handles messages for the palette.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			// Close without selecting - handled by parent
			return m, nil

		case tea.KeyEnter:
			// Select current entry
			if entry := m.SelectedEntry(); entry != nil {
				return m, func() tea.Msg {
					return CommandSelectedMsg{
						CommandID: entry.CommandID,
						Context:   entry.Context,
					}
				}
			}
			return m, nil

		case tea.KeyUp, tea.KeyCtrlP:
			m.moveCursor(-1)
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN:
			m.moveCursor(1)
			return m, nil

		case tea.KeyCtrlU:
			// Page up
			m.moveCursor(-m.maxVisible)
			return m, nil

		case tea.KeyCtrlD:
			// Page down
			m.moveCursor(m.maxVisible)
			return m, nil

		default:
			// Pass to text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)

			// Re-filter on query change
			m.filtered = FilterEntries(m.allEntries, m.textInput.Value())
			m.cursor = 0
			m.offset = 0

			return m, tea.Batch(cmds...)
		}

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// moveCursor moves the cursor by delta, clamping to valid range.
func (m *Model) moveCursor(delta int) {
	m.cursor += delta

	// Clamp to valid range
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Adjust scroll offset to keep cursor visible
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.maxVisible {
		m.offset = m.cursor - m.maxVisible + 1
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
