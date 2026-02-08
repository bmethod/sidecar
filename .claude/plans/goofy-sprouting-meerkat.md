# Quick-Switch Workspace Keybindings (alt+1 through alt+9)

## Context

When working in interactive mode (typing into a shell), there's no way to quickly jump to a different shell/workspace without manually exiting interactive mode, navigating with j/k, and re-entering. The user wants ctrl+1/2/3 style shortcuts, but **ctrl+number doesn't work reliably in terminals** (ctrl+2=NUL, ctrl+3=ESC, etc.). We'll use `alt+1` through `alt+9` instead, which send reliable ESC+number sequences.

## Behavior

- `alt+1` through `alt+9` jumps to the Nth item in the unified sidebar list (shells first, then worktrees)
- **From interactive mode**: exits current session, switches selection, re-enters interactive mode on new target
- **From list/preview mode**: switches selection directly
- Out-of-range indices are no-ops
- If target has no agent, switches but stays in list mode (can't enter interactive on nothing)

## Changes

### 1. `internal/plugins/workspace/plugin.go` (~line 903, after `moveCursor()`)

Add `switchToIndex(idx int) tea.Cmd`:
- Takes 0-based unified index (0=first shell/worktree, 1=second, etc.)
- Maps index to shell vs worktree based on `len(p.shells)` boundary
- Sets `shellSelected`, `selectedShellIdx`/`selectedIdx`
- Resets preview state (same as `moveCursor`: previewOffset, autoScrollOutput, etc.)
- Returns `loadSelectedContent()`, or nil if already on target or out of range

### 2. `internal/plugins/workspace/keys.go`

Add `parseAltNumber(key string) (int, bool)` helper:
- Parses `"alt+1"` through `"alt+9"` → returns 0-based index
- Used by both interactive and list mode handlers

Add cases in `handleListKeys()` (before `default:` at line 883):
```go
case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7", "alt+8", "alt+9":
    if altIdx, ok := parseAltNumber(msg.String()); ok {
        return p.switchToIndex(altIdx)
    }
```

### 3. `internal/plugins/workspace/interactive.go` (~line 886, after paste handling, before "Update last key time")

Add alt+number interception in `handleInteractiveKeys()`:
```go
if altIdx, ok := parseAltNumber(msg.String()); ok {
    p.exitInteractiveMode()
    if cmd := p.switchToIndex(altIdx); cmd != nil {
        enterCmd := p.enterInteractiveMode()
        if enterCmd != nil {
            return tea.Batch(cmd, enterCmd)
        }
        return cmd
    }
    // Out of range or same target: re-enter on current
    return p.enterInteractiveMode()
}
```

### 4. `internal/keymap/bindings.go`

Add bindings for help palette visibility (NOT footer — too many items):

After line 365 (end of workspace-list bindings):
```go
{Key: "alt+1", Command: "switch-1", Context: "workspace-list"},
...
{Key: "alt+9", Command: "switch-9", Context: "workspace-list"},
```

After line 390 (end of workspace-preview bindings):
```go
{Key: "alt+1", Command: "switch-1", Context: "workspace-preview"},
...
{Key: "alt+9", Command: "switch-9", Context: "workspace-preview"},
```

### 5. `internal/plugins/workspace/plugin.go` (Init, ~line 443)

Register dynamic bindings for interactive context:
```go
for i := 1; i <= 9; i++ {
    ctx.Keymap.RegisterPluginBinding(fmt.Sprintf("alt+%d", i), fmt.Sprintf("switch-%d", i), "workspace-interactive")
}
```

## Verification

1. `go build ./...` — compiles clean
2. `go test ./...` — tests pass
3. Manual: open sidecar, create 2+ shells, enter interactive mode on shell 1, press `alt+2` → should switch to shell 2 in interactive mode
4. Manual: from list mode, press `alt+1` → should select first item
5. Manual: press `alt+9` with <9 items → should be no-op
