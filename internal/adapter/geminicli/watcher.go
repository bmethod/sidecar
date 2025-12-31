package geminicli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/marcus/sidecar/internal/adapter"
)

// NewWatcher creates a watcher for Gemini CLI session changes.
func NewWatcher(chatsDir string) (<-chan adapter.Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(chatsDir); err != nil {
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

				// Only watch session-*.json files
				name := filepath.Base(event.Name)
				if !strings.HasPrefix(name, "session-") || !strings.HasSuffix(name, ".json") {
					continue
				}

				lastEvent = event

				// Debounce rapid events
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					sessionID := extractSessionID(lastEvent.Name)
					if sessionID == "" {
						return
					}

					var eventType adapter.EventType
					switch {
					case lastEvent.Op&fsnotify.Create != 0:
						eventType = adapter.EventSessionCreated
					case lastEvent.Op&fsnotify.Write != 0:
						eventType = adapter.EventMessageAdded
					case lastEvent.Op&fsnotify.Remove != 0:
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
			}
		}
	}()

	return events, nil
}

// extractSessionID reads the session file and extracts the sessionId field.
func extractSessionID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var session struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return ""
	}

	return session.SessionID
}
