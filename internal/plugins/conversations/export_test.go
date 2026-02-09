package conversations

import (
	"strings"
	"testing"
	"time"

	"github.com/marcus/sidecar/internal/adapter"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "my-session", "my-session"},
		{"with spaces", "my session name", "my session name"},
		{"with slashes", "path/to/session", "path-to-session"},
		{"with backslashes", "path\\to\\session", "path-to-session"},
		{"with colons", "session:name", "session-name"},
		{"with asterisks", "session*name", "sessionname"},
		{"with question marks", "session?name", "sessionname"},
		{"with quotes", "session\"name", "sessionname"},
		{"with angle brackets", "session<name>", "sessionname"},
		{"with pipe", "session|name", "sessionname"},
		{"with newlines", "session\nname", "session name"},
		{"with carriage returns", "session\rname", "sessionname"},
		{"mixed invalid chars", "my/session:name*test", "my-session-nametest"},
		{"very long name", strings.Repeat("a", 100), strings.Repeat("a", 50)},
		{"empty string", "", "session"},
		{"only spaces", "   ", "session"},
		{"only dashes", "---", "session"},
		{"spaces and dashes", " - - ", "session"},
		{"unicode characters", "セッション名", "セッション名"},
		{"unicode long", strings.Repeat("日", 60), strings.Repeat("日", 50)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExportSessionAsMarkdown_EmptySession(t *testing.T) {
	result := ExportSessionAsMarkdown(nil, nil)
	
	if !strings.Contains(result, "# Session:") {
		t.Error("expected session header")
	}
	if !strings.Contains(result, "Unknown Session") {
		t.Error("expected 'Unknown Session' for nil session")
	}
}

func TestExportSessionAsMarkdown_WithSessionInfo(t *testing.T) {
	session := &adapter.Session{
		ID:          "ses_123",
		Name:        "Test Session",
		CreatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:    45 * time.Minute,
		TotalTokens: 5000,
		EstCost:     0.25,
	}

	result := ExportSessionAsMarkdown(session, nil)

	if !strings.Contains(result, "# Session: Test Session") {
		t.Error("expected session name in header")
	}
	if !strings.Contains(result, "**Date**: 2024-01-15 10:30") {
		t.Error("expected formatted date")
	}
	if !strings.Contains(result, "**Duration**: 45m") {
		t.Error("expected duration")
	}
	if !strings.Contains(result, "**Tokens**: 5000") {
		t.Error("expected token count")
	}
	if !strings.Contains(result, "**Estimated Cost**: $0.25") {
		t.Error("expected cost")
	}
}

func TestExportSessionAsMarkdown_WithMessages(t *testing.T) {
	session := &adapter.Session{
		ID:   "ses_123",
		Name: "Test",
	}

	messages := []adapter.Message{
		{
			Role:      "user",
			Content:   "Hello, how are you?",
			Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			Role:      "assistant",
			Content:   "I'm doing well, thank you!",
			Timestamp: time.Date(2024, 1, 15, 10, 30, 5, 0, time.UTC),
			Model:     "claude-3-5-sonnet-20241022",
			TokenUsage: adapter.TokenUsage{
				InputTokens:  10,
				OutputTokens: 8,
			},
		},
	}

	result := ExportSessionAsMarkdown(session, messages)

	if !strings.Contains(result, "## User (10:30:00)") {
		t.Error("expected user message header")
	}
	if !strings.Contains(result, "Hello, how are you?") {
		t.Error("expected user message content")
	}
	if !strings.Contains(result, "## Assistant (10:30:05)") {
		t.Error("expected assistant message header")
	}
	if !strings.Contains(result, "I'm doing well, thank you!") {
		t.Error("expected assistant message content")
	}
	if !strings.Contains(result, "*Model:") {
		t.Error("expected model info for assistant message")
	}
	if !strings.Contains(result, "*Tokens: in=10, out=8*") {
		t.Error("expected token info")
	}
}

func TestExportSessionAsMarkdown_WithThinkingBlocks(t *testing.T) {
	session := &adapter.Session{ID: "ses_123", Name: "Test"}

	messages := []adapter.Message{
		{
			Role:      "assistant",
			Content:   "Here's my response",
			Timestamp: time.Now(),
			ThinkingBlocks: []adapter.ThinkingBlock{
				{
					Content:    "Let me think about this...",
					TokenCount: 50,
				},
			},
		},
	}

	result := ExportSessionAsMarkdown(session, messages)

	if !strings.Contains(result, "<details>") {
		t.Error("expected details tag for thinking block")
	}
	if !strings.Contains(result, "<summary>Thinking (50 tokens)</summary>") {
		t.Error("expected thinking block summary")
	}
	if !strings.Contains(result, "Let me think about this...") {
		t.Error("expected thinking block content")
	}
}

func TestExportSessionAsMarkdown_WithToolUses(t *testing.T) {
	session := &adapter.Session{ID: "ses_123", Name: "Test"}

	messages := []adapter.Message{
		{
			Role:      "assistant",
			Content:   "I used some tools",
			Timestamp: time.Now(),
			ToolUses: []adapter.ToolUse{
				{
					Name:  "fs_read",
					Input: `{"path": "/path/to/file.txt"}`,
				},
				{
					Name:  "execute_bash",
					Input: `{"command": "ls -la"}`,
				},
			},
		},
	}

	result := ExportSessionAsMarkdown(session, messages)

	if !strings.Contains(result, "**Tools used:**") {
		t.Error("expected tools used section")
	}
	if !strings.Contains(result, "fs_read") {
		t.Error("expected fs_read tool")
	}
	if !strings.Contains(result, "execute_bash") {
		t.Error("expected execute_bash tool")
	}
}

func TestFormatExportDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"minutes", 15 * time.Minute, "15m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"hours only", 3 * time.Hour, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExportDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("formatExportDuration(%v) = %q, want %q", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestExportSessionAsMarkdown_SessionWithoutName(t *testing.T) {
	session := &adapter.Session{
		ID:        "ses_abc123",
		Name:      "",
		CreatedAt: time.Now(),
	}

	result := ExportSessionAsMarkdown(session, nil)

	if !strings.Contains(result, "# Session: ses_abc123") {
		t.Error("expected session ID as fallback when name is empty")
	}
}

func TestExportSessionAsMarkdown_RoleCapitalization(t *testing.T) {
	session := &adapter.Session{ID: "ses_123", Name: "Test"}

	messages := []adapter.Message{
		{Role: "user", Content: "test", Timestamp: time.Now()},
		{Role: "assistant", Content: "test", Timestamp: time.Now()},
		{Role: "system", Content: "test", Timestamp: time.Now()},
	}

	result := ExportSessionAsMarkdown(session, messages)

	if !strings.Contains(result, "## User") {
		t.Error("expected capitalized 'User'")
	}
	if !strings.Contains(result, "## Assistant") {
		t.Error("expected capitalized 'Assistant'")
	}
	if !strings.Contains(result, "## System") {
		t.Error("expected capitalized 'System'")
	}
}
