package warp

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sst/sidecar/internal/adapter"
)

// NewWatcher creates a watcher for Warp SQLite changes.
// Watches the WAL file for modifications since Warp uses WAL mode.
func NewWatcher(dbPath string) (<-chan adapter.Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the directory containing the SQLite files
	dbDir := filepath.Dir(dbPath)
	if err := watcher.Add(dbDir); err != nil {
		watcher.Close()
		return nil, err
	}

	walFile := dbPath + "-wal"
	events := make(chan adapter.Event, 32)

	go func() {
		defer watcher.Close()
		defer close(events)

		var debounceTimer *time.Timer
		debounceDelay := 100 * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only watch for writes to the WAL file or main DB
				if event.Name != walFile && event.Name != dbPath {
					continue
				}

				if event.Op&fsnotify.Write == 0 {
					continue
				}

				// Debounce rapid writes
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					select {
					case events <- adapter.Event{
						Type: adapter.EventSessionUpdated,
					}:
					default:
						// Channel full, skip
					}
				})

			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Ignore errors, just keep watching
			}
		}
	}()

	return events, nil
}
