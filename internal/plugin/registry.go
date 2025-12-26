package plugin

import (
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Registry manages plugin registration and lifecycle.
type Registry struct {
	plugins     []Plugin
	unavailable map[string]string // pluginID -> error reason
	ctx         *Context
	mu          sync.RWMutex
}

// NewRegistry creates a new plugin registry with the given context.
func NewRegistry(ctx *Context) *Registry {
	return &Registry{
		plugins:     make([]Plugin, 0),
		unavailable: make(map[string]string),
		ctx:         ctx,
	}
}

// Register adds a plugin to the registry.
// If Init fails, the plugin is marked unavailable (silent degradation).
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.safeInit(p); err != nil {
		r.unavailable[p.ID()] = err.Error()
		if r.ctx != nil && r.ctx.Logger != nil {
			r.ctx.Logger.Warn("plugin init failed", "id", p.ID(), "error", err)
		}
		return nil // Silent degradation - not an error
	}

	r.plugins = append(r.plugins, p)
	return nil
}

// safeInit calls Init with panic recovery.
func (r *Registry) safeInit(p Plugin) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic: %v", rec)
		}
	}()
	return p.Init(r.ctx)
}

// Start starts all registered plugins and returns their initial commands.
func (r *Registry) Start() []tea.Cmd {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]tea.Cmd, 0, len(r.plugins))
	for _, p := range r.plugins {
		if cmd := r.safeStart(p); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// safeStart calls Start with panic recovery.
func (r *Registry) safeStart(p Plugin) (cmd tea.Cmd) {
	defer func() {
		if rec := recover(); rec != nil {
			if r.ctx != nil && r.ctx.Logger != nil {
				r.ctx.Logger.Error("plugin start panic", "id", p.ID(), "error", rec)
			}
			cmd = nil
		}
	}()
	return p.Start()
}

// Stop stops all registered plugins in reverse order.
func (r *Registry) Stop() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.plugins) - 1; i >= 0; i-- {
		r.safeStop(r.plugins[i])
	}
}

// safeStop calls Stop with panic recovery.
func (r *Registry) safeStop(p Plugin) {
	defer func() {
		if rec := recover(); rec != nil {
			if r.ctx != nil && r.ctx.Logger != nil {
				r.ctx.Logger.Error("plugin stop panic", "id", p.ID(), "error", rec)
			}
		}
	}()
	p.Stop()
}

// Plugins returns all active plugins.
func (r *Registry) Plugins() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, len(r.plugins))
	copy(result, r.plugins)
	return result
}

// Get returns a plugin by ID, or nil if not found.
func (r *Registry) Get(id string) Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// Unavailable returns a map of plugin IDs to their failure reasons.
func (r *Registry) Unavailable() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.unavailable))
	for k, v := range r.unavailable {
		result[k] = v
	}
	return result
}
