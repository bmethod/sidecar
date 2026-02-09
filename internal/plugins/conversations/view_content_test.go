package conversations

import (
	"testing"

	"github.com/marcus/sidecar/internal/adapter"
)

func TestAdapterAbbrev(t *testing.T) {
	tests := []struct {
		name    string
		session adapter.Session
		want    string
	}{
		{
			name:    "claude-code",
			session: adapter.Session{AdapterID: "claude-code"},
			want:    "CC",
		},
		{
			name:    "codex",
			session: adapter.Session{AdapterID: "codex"},
			want:    "CX",
		},
		{
			name:    "opencode",
			session: adapter.Session{AdapterID: "opencode"},
			want:    "OC",
		},
		{
			name:    "gemini-cli",
			session: adapter.Session{AdapterID: "gemini-cli"},
			want:    "GC",
		},
		{
			name:    "custom adapter with name",
			session: adapter.Session{AdapterID: "mytool", AdapterName: "My Tool"},
			want:    "MY",
		},
		{
			name:    "custom adapter with only ID",
			session: adapter.Session{AdapterID: "warp"},
			want:    "WA",
		},
		{
			name:    "empty ID and name",
			session: adapter.Session{},
			want:    "",
		},
		{
			name:    "short name",
			session: adapter.Session{AdapterID: "x", AdapterName: "X"},
			want:    "X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapterAbbrev(tt.session)
			if got != tt.want {
				t.Errorf("adapterAbbrev() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdapterBadgeText(t *testing.T) {
	tests := []struct {
		name    string
		session adapter.Session
		want    string
	}{
		{
			name:    "with adapter icon",
			session: adapter.Session{AdapterIcon: "🤖"},
			want:    "🤖",
		},
		{
			name:    "no icon with known adapter",
			session: adapter.Session{AdapterID: "claude-code"},
			want:    "●CC",
		},
		{
			name:    "no icon empty ID and name",
			session: adapter.Session{},
			want:    "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapterBadgeText(tt.session)
			if got != tt.want {
				t.Errorf("adapterBadgeText() = %q, want %q", got, tt.want)
			}
		})
	}
}
