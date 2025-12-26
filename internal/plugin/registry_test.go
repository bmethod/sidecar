package plugin

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockPlugin implements Plugin for testing.
type mockPlugin struct {
	id          string
	initErr     error
	initPanic   bool
	startPanic  bool
	stopPanic   bool
	started     bool
	stopped     bool
}

func (m *mockPlugin) ID() string      { return m.id }
func (m *mockPlugin) Name() string    { return m.id }
func (m *mockPlugin) Icon() string    { return "📦" }
func (m *mockPlugin) IsFocused() bool { return false }
func (m *mockPlugin) SetFocused(bool) {}
func (m *mockPlugin) Commands() []Command { return nil }
func (m *mockPlugin) FocusContext() string { return m.id }
func (m *mockPlugin) View(w, h int) string { return "" }
func (m *mockPlugin) Update(msg tea.Msg) (Plugin, tea.Cmd) { return m, nil }

func (m *mockPlugin) Init(ctx *Context) error {
	if m.initPanic {
		panic("init panic")
	}
	return m.initErr
}

func (m *mockPlugin) Start() tea.Cmd {
	if m.startPanic {
		panic("start panic")
	}
	m.started = true
	return nil
}

func (m *mockPlugin) Stop() {
	if m.stopPanic {
		panic("stop panic")
	}
	m.stopped = true
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry(nil)

	p := &mockPlugin{id: "test"}
	if err := r.Register(p); err != nil {
		t.Errorf("Register failed: %v", err)
	}

	plugins := r.Plugins()
	if len(plugins) != 1 {
		t.Errorf("got %d plugins, want 1", len(plugins))
	}
}

func TestRegistry_RegisterInitError(t *testing.T) {
	r := NewRegistry(nil)

	p := &mockPlugin{id: "failing", initErr: errors.New("init failed")}
	if err := r.Register(p); err != nil {
		t.Errorf("Register should not return error for failed init: %v", err)
	}

	// Plugin should be in unavailable, not in active plugins
	plugins := r.Plugins()
	if len(plugins) != 0 {
		t.Errorf("got %d plugins, want 0", len(plugins))
	}

	unavail := r.Unavailable()
	if _, ok := unavail["failing"]; !ok {
		t.Error("plugin should be in unavailable map")
	}
}

func TestRegistry_RegisterInitPanic(t *testing.T) {
	r := NewRegistry(nil)

	p := &mockPlugin{id: "panicky", initPanic: true}
	if err := r.Register(p); err != nil {
		t.Errorf("Register should not return error for panic: %v", err)
	}

	unavail := r.Unavailable()
	if _, ok := unavail["panicky"]; !ok {
		t.Error("panicked plugin should be in unavailable map")
	}
}

func TestRegistry_StartStop(t *testing.T) {
	r := NewRegistry(nil)

	p1 := &mockPlugin{id: "p1"}
	p2 := &mockPlugin{id: "p2"}
	r.Register(p1)
	r.Register(p2)

	r.Start()
	if !p1.started || !p2.started {
		t.Error("plugins should be started")
	}

	r.Stop()
	if !p1.stopped || !p2.stopped {
		t.Error("plugins should be stopped")
	}
}

func TestRegistry_StartStopPanic(t *testing.T) {
	r := NewRegistry(nil)

	p1 := &mockPlugin{id: "p1", startPanic: true}
	p2 := &mockPlugin{id: "p2", stopPanic: true}
	r.Register(p1)
	r.Register(p2)

	// Should not panic
	r.Start()
	r.Stop()
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(nil)

	p := &mockPlugin{id: "findme"}
	r.Register(p)

	found := r.Get("findme")
	if found == nil {
		t.Error("Get should find registered plugin")
	}

	notFound := r.Get("missing")
	if notFound != nil {
		t.Error("Get should return nil for missing plugin")
	}
}
