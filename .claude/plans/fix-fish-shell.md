Fix fish shell incompatibility in workspace environment setup.

## Problem

Sidecar emits POSIX shell syntax (`export`, `unset`) into tmux sessions, which fails when the user's shell is fish. Fish uses `set -gx FOO 'bar'` and `set -e FOO` instead.

## Affected Files

### Critical — env.go (`internal/plugins/workspace/env.go`)
- `GenerateExportCommands()` (lines ~101-114): Generates `export KEY='val'` and `unset KEY`
- `GenerateSingleEnvCommand()` (lines ~116-136): Same, concatenated with `;`
- These need a shell parameter or should detect the shell and emit the correct syntax

### Critical — agent.go (`internal/plugins/workspace/agent.go`)
- Line ~374 (`StartAgent`): `fmt.Sprintf("export TD_SESSION_ID=%s", ...)`
- Line ~572 (`StartAgentWithOptions`): Same hardcoded export
- Lines ~377-381, ~575-579: Call `GenerateSingleEnvCommand()` without shell awareness

### Critical — shell.go (`internal/plugins/workspace/shell.go`)
- Line ~1043 (`startAgentWithResumeCmd`): Same hardcoded `export TD_SESSION_ID`
- Lines ~1046-1050: Call `GenerateSingleEnvCommand()` without shell awareness

### High — agent.go launcher script (~lines 464-525)
- `writeAgentLauncher()` uses `#!/bin/bash`, `export`, `[ -s ... ]`, `source`
- Does not account for fish at all

## Fix Approach

1. Add shell detection helper (check `$SHELL` env var for `/fish`):
```go
func detectShell() string {
    shell := os.Getenv("SHELL")
    if strings.Contains(shell, "/fish") {
        return "fish"
    }
    return "posix"
}
```

2. Update `GenerateExportCommands()` and `GenerateSingleEnvCommand()` to accept a shell type parameter and emit the correct syntax:
   - posix: `export KEY='val'` / `unset KEY` / joined with `; `
   - fish: `set -gx KEY 'val'` / `set -e KEY` / joined with `; `

3. Update all three call sites (StartAgent, StartAgentWithOptions, startAgentWithResumeCmd) to use detected shell for both TD_SESSION_ID and env overrides.

4. For the launcher script in `writeAgentLauncher()`, detect shell and either:
   - Use `#!/usr/bin/env fish` with fish syntax, OR
   - Keep bash launcher scripts (they run independently, not in user's shell) — only the `send-keys` commands need fish syntax since those execute in the tmux pane's shell.

## Important Distinction

The `send-keys` commands run inside the user's tmux shell (fish) — these MUST use fish syntax.
The launcher scripts (`writeAgentLauncher`) run as standalone bash scripts — these can stay as bash since they have their own shebang.

## Tests

Update `env_test.go` (`TestGenerateExportCommands`) to cover both posix and fish output.
