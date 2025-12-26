# Sidecar

Terminal UI for viewing AI coding agent sessions. Monitor Claude Code conversations, git status, and task progress in a unified interface.

## Requirements

- Go 1.23+

## Installation

```bash
# Install globally (adds 'sidecar' to your PATH)
go install github.com/sst/sidecar/cmd/sidecar@latest

# Or clone and install locally
git clone https://github.com/sst/sidecar
cd sidecar
go install ./cmd/sidecar
```

## Usage

```bash
# Run from any project directory
sidecar

# Specify project root
sidecar --project /path/to/project

# Use custom config
sidecar --config ~/.config/sidecar/config.json

# Enable debug logging
sidecar --debug
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q`, `ctrl+c` | Quit |
| `tab` | Next plugin |
| `shift+tab` | Previous plugin |
| `1-9` | Focus plugin by number |
| `?` | Toggle help |
| `!` | Toggle diagnostics |
| `ctrl+h` | Toggle footer |
| `r` | Refresh |
| `j/k` or `↓/↑` | Navigate |
| `enter` | Select |
| `esc` | Back/close |

## Configuration

Config file: `~/.config/sidecar/config.json`

```json
{
  "plugins": {
    "git-status": { "enabled": true, "refreshInterval": "1s" },
    "td-monitor": { "enabled": true, "refreshInterval": "2s" },
    "conversations": { "enabled": true }
  },
  "ui": {
    "showFooter": true,
    "showClock": true
  }
}
```

## Development

```bash
# Build
go build ./cmd/sidecar

# Run directly
go run ./cmd/sidecar

# Run tests
go test ./...
```

## Status

Early development. Core shell implemented, plugins in progress.
