# Mouse Support for Sidecar Plugins

This guide explains how to add mouse support to sidecar plugins using the `internal/mouse` package.

## Architecture

### Coordinate System

Sidecar has a 2-line header that's always visible. Mouse events use screen coordinates, so sidecar offsets Y by 2 before forwarding `tea.MouseMsg` to plugins. Plugins receive coordinates in their own content space starting at Y=0.

```
Screen Y=0  ┌─────────────────┐  ← App header (2 lines)
Screen Y=2  │                 │  ← Plugin content starts (Y=0 in plugin space)
            │  Plugin View    │
            │                 │
            └─────────────────┘
```

### Core Types

The `internal/mouse` package provides:

- **Rect** - Rectangle with `Contains(x, y)` hit testing
- **Region** - Named rect with associated data (e.g., file index)
- **HitMap** - Collection of regions with `Clear()`, `Add()`, `Test()`
- **Handler** - Combines HitMap with click/drag/double-click tracking

## Adding Mouse Support to a Plugin

### 1. Add Handler Field

```go
import "github.com/sst/sidecar/internal/mouse"

type Plugin struct {
    // ... other fields
    mouseHandler *mouse.Handler
}

func New() *Plugin {
    return &Plugin{
        mouseHandler: mouse.NewHandler(),
    }
}
```

### 2. Handle MouseMsg in Update

```go
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        return p.handleMouse(msg)
    // ... other cases
    }
    return p, nil
}
```

### 3. Register Hit Regions During Render

Hit regions must be updated each render to match the current layout:

```go
func (p *Plugin) View(width, height int) string {
    // Clear old regions at start of render
    p.mouseHandler.Clear()

    // Register regions as you render elements
    p.mouseHandler.HitMap.AddRect("sidebar", 0, 0, sidebarWidth, height, nil)
    p.mouseHandler.HitMap.AddRect("file-0", 2, 5, width-4, 1, 0) // data: file index

    // ... render content
}
```

### 4. Process Mouse Actions

```go
func (p *Plugin) handleMouse(msg tea.MouseMsg) (*Plugin, tea.Cmd) {
    action := p.mouseHandler.HandleMouse(msg)

    switch action.Type {
    case mouse.ActionClick:
        if action.Region != nil {
            switch action.Region.ID {
            case "sidebar":
                p.focusSidebar()
            case "file-0":
                idx := action.Region.Data.(int)
                p.selectFile(idx)
            }
        }

    case mouse.ActionDoubleClick:
        if action.Region != nil && action.Region.ID == "file-0" {
            idx := action.Region.Data.(int)
            return p, p.openFile(idx)
        }

    case mouse.ActionScrollUp, mouse.ActionScrollDown:
        if action.Region != nil {
            p.scroll(action.Delta)
        }

    case mouse.ActionDrag:
        // Handle drag (e.g., pane resizing)
        dx, dy := action.DragDX, action.DragDY
    }

    return p, nil
}
```

## Common Patterns

### Click to Select

```go
case mouse.ActionClick:
    if action.Region != nil && action.Region.ID == "item" {
        if idx, ok := action.Region.Data.(int); ok {
            p.cursor = idx
            p.ensureCursorVisible()
        }
    }
```

### Scroll Wheel

```go
case mouse.ActionScrollUp, mouse.ActionScrollDown:
    // Scroll content by delta (typically 3 lines)
    p.scrollOffset += action.Delta
    p.clampScroll()
```

### Double-Click Actions

```go
case mouse.ActionDoubleClick:
    if action.Region != nil {
        switch action.Region.ID {
        case "file":
            return p, p.openFile(action.Region.Data.(int))
        case "folder":
            p.toggleFolder(action.Region.Data.(int))
        }
    }
```

### Drag to Resize

```go
case mouse.ActionClick:
    if action.Region != nil && action.Region.ID == "divider" {
        // Start drag with current width as initial value
        p.mouseHandler.StartDrag(action.X, action.Y, "divider", p.sidebarWidth)
    }

case mouse.ActionDrag:
    if p.mouseHandler.DragRegion() == "divider" {
        // Calculate new width from drag delta
        newWidth := p.mouseHandler.DragStartValue() + action.DragDX
        p.sidebarWidth = clamp(newWidth, minWidth, maxWidth)
    }
```

## Hit Region Best Practices

1. **Clear regions each render** - The layout may change, so rebuild the hit map
2. **Register from bottom to top** - Later regions take priority (for overlapping)
3. **Use meaningful IDs** - `"file"`, `"commit"`, `"divider"` not `"region1"`
4. **Store indices in Data** - Use `Data` field for item indices, not string parsing
5. **Account for borders** - Subtract padding/borders from click regions

## Example: Git Plugin

The git plugin demonstrates full mouse support:

- `internal/plugins/gitstatus/mouse.go` - Mouse event handlers
- `internal/plugins/gitstatus/sidebar_view.go` - Hit region registration

Key features:
- Click to select files/commits
- Scroll wheel navigation
- Double-click to open files or toggle folders
- Drag pane divider to resize

## Testing Mouse Support

Since mouse events require terminal interaction, test by:

1. Running `sidecar` in a terminal with mouse support
2. Clicking on UI elements to verify selection
3. Using scroll wheel to navigate lists
4. Double-clicking to trigger actions
5. Dragging dividers to resize panes
