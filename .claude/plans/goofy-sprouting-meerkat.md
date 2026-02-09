# Quick-Switch Workspace Keybindings (ctrl+1 through ctrl+9)

## Context

When working in interactive mode (typing into a shell), there's no way to quickly jump to a different shell/workspace without manually exiting interactive mode, navigating with j/k, and re-entering. We'll use `ctrl+1` through `ctrl+9` to switch directly by index.

**Note:** `ctrl+number` may not produce unique escape sequences in all terminals. If Bubble Tea doesn't recognize them as distinct key events, we may need to check what `msg.String()` actually returns for these keys and adjust the matching accordingly. Investigate during implementation.

## Behavior

- `ctrl+1` through `ctrl+9` jumps to the Nth item in the unified sidebar list (shells first, then worktrees)
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

Add `parseCtrlNumber(key string) (int, bool)` helper:
- Parses `"ctrl+1"` through `"ctrl+9"` → returns 0-based index
- Used by both interactive and list mode handlers

Add cases in `handleListKeys()` (before `default:` at line 883):
```go
case "ctrl+1", "ctrl+2", "ctrl+3", "ctrl+4", "ctrl+5", "ctrl+6", "ctrl+7", "ctrl+8", "ctrl+9":
    if idx, ok := parseCtrlNumber(msg.String()); ok {
        return p.switchToIndex(idx)
    }
```

### 3. `internal/plugins/workspace/interactive.go` (~line 886, after paste handling, before "Update last key time")

Add ctrl+number interception in `handleInteractiveKeys()`:
```go
if idx, ok := parseCtrlNumber(msg.String()); ok {
    p.exitInteractiveMode()
    if cmd := p.switchToIndex(idx); cmd != nil {
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
{Key: "ctrl+1", Command: "switch-1", Context: "workspace-list"},
...
{Key: "ctrl+9", Command: "switch-9", Context: "workspace-list"},
```

After line 390 (end of workspace-preview bindings):
```go
{Key: "ctrl+1", Command: "switch-1", Context: "workspace-preview"},
...
{Key: "ctrl+9", Command: "switch-9", Context: "workspace-preview"},
```

### 5. `internal/plugins/workspace/plugin.go` (Init, ~line 443)

Register dynamic bindings for interactive context:
```go
for i := 1; i <= 9; i++ {
    ctx.Keymap.RegisterPluginBinding(fmt.Sprintf("ctrl+%d", i), fmt.Sprintf("switch-%d", i), "workspace-interactive")
}
```

## Verification

1. `go build ./...` — compiles clean
2. `go test ./...` — tests pass
3. Manual: open sidecar, create 2+ shells, enter interactive mode on shell 1, press `ctrl+2` → should switch to shell 2 in interactive mode
4. Manual: from list mode, press `ctrl+1` → should select first item
5. Manual: press `ctrl+9` with <9 items → should be no-op
6. **Terminal compatibility**: test in kitty/alacritty/wezterm to verify ctrl+number produces recognizable key events
