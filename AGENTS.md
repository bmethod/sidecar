# Agents: Mandatory: use td usage --new-session to see open work

If you don't have an existing task for the work you're about to start, create one first.

## Build & Versioning

```bash
# Build
go build ./...

# Run tests
go test ./...

# Install with version (use semantic versioning)
go install -ldflags "-X main.Version=v0.1.0" ./cmd/sidecar

# Tag a release
git tag v0.1.0 -m "Release message"
```

Version is set via ldflags at build time. Without it, sidecar shows git revision info.

## Keyboard Shortcut Parity

Maintain consistent keybindings across plugins for familiar UX:

**Navigation shortcuts that should work in all scrollable views:**
- `j`/`down` - scroll down (1 line)
- `k`/`up` - scroll up (1 line)
- `ctrl+d` - page down (~10 lines) [file browser ✓, git diff ✗]
- `ctrl+u` - page up (~10 lines) [file browser ✓, git diff ✗]
- `g` - go to top [file browser ✓, git diff ✓]
- `G` - go to bottom [file browser ✓, git diff ✓]

See td-331dbf19 for diff paging implementation.
