package plugin

import tea "github.com/charmbracelet/bubbletea"

// Plugin defines the interface for all sidecar plugins.
type Plugin interface {
	ID() string
	Name() string
	Icon() string
	Init(ctx *Context) error
	Start() tea.Cmd
	Stop()
	Update(msg tea.Msg) (Plugin, tea.Cmd)
	View(width, height int) string
	IsFocused() bool
	SetFocused(bool)
	Commands() []Command
	FocusContext() string
}

// Category represents a logical grouping of commands for the command palette.
type Category string

const (
	CategoryNavigation Category = "Navigation"
	CategoryActions    Category = "Actions"
	CategoryView       Category = "View"
	CategorySearch     Category = "Search"
	CategoryEdit       Category = "Edit"
	CategoryGit        Category = "Git"
	CategorySystem     Category = "System"
)

// Command represents a keybinding command exposed by a plugin.
type Command struct {
	ID          string         // Unique identifier (e.g., "stage-file")
	Name        string         // Short name for footer (e.g., "Stage")
	Description string         // Full description for palette
	Category    Category       // Logical grouping for palette display
	Handler     func() tea.Cmd // Action to execute (optional)
	Context     string         // Activation context
}

// DiagnosticProvider is implemented by plugins that expose diagnostics.
type DiagnosticProvider interface {
	Diagnostics() []Diagnostic
}

// Diagnostic represents a health/status check result.
type Diagnostic struct {
	ID     string
	Status string
	Detail string
}
