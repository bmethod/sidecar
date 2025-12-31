package claudecode

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/marcus/sidecar/internal/adapter"
)

// NewWatcher creates a watcher for Claude Code session changes.
func NewWatcher(projectDir string) (<-chan adapter.Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(projectDir); err != nil {
		watcher.Close()
		return nil, err
	}

	events := make(chan adapter.Event, 32)

	go func() {
		defer watcher.Close()
		defer close(events)

		// Debounce timer
		var debounceTimer *time.Timer
		var lastEvent fsnotify.Event
		debounceDelay := 100 * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only watch .jsonl files
				if !strings.HasSuffix(event.Name, ".jsonl") {
					continue
				}

				lastEvent = event

				// Debounce rapid events
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					sessionID := strings.TrimSuffix(filepath.Base(lastEvent.Name), ".jsonl")

					var eventType adapter.EventType
					switch {
					case lastEvent.Op&fsnotify.Create != 0:
						eventType = adapter.EventSessionCreated
					case lastEvent.Op&fsnotify.Write != 0:
						eventType = adapter.EventMessageAdded
					case lastEvent.Op&fsnotify.Remove != 0:
						// Session deleted, could add EventSessionDeleted
						return
					default:
						eventType = adapter.EventSessionUpdated
					}

					select {
					case events <- adapter.Event{
						Type:      eventType,
						SessionID: sessionID,
					}:
					default:
						// Channel full, drop event
					}
				})

			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Log error but continue watching
			}
		}
	}()

	return events, nil
}
