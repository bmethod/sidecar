package styles

import "testing"

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid 6-char hex colors
		{"valid uppercase", "#FF5500", true},
		{"valid lowercase", "#aabbcc", true},
		{"valid mixed case", "#AbCdEf", true},
		{"valid all zeros", "#000000", true},
		{"valid all Fs", "#FFFFFF", true},
		
		// Valid 8-char hex colors with alpha
		{"valid with alpha 80", "#00000080", true},
		{"valid with alpha FF", "#FF5500FF", true},
		{"valid with alpha 00", "#aabbcc00", true},
		
		// Invalid formats - wrong length
		{"invalid 3-char", "#FFF", false},
		{"invalid 4-char", "#FFFF", false},
		{"invalid 5-char", "#FF550", false},
		{"invalid 7-char", "#FF55001", false},
		{"invalid 9-char", "#FF5500801", false},
		
		// Invalid formats - no hash
		{"no hash 6-char", "FF5500", false},
		{"no hash 8-char", "FF550080", false},
		
		// Invalid formats - invalid characters
		{"invalid char G", "#GGGGGG", false},
		{"invalid char Z", "#ZZZZZZ", false},
		{"invalid char space", "#FF 550", false},
		{"invalid char dash", "#FF-550", false},
		
		// Edge cases
		{"empty string", "", false},
		{"just hash", "#", false},
		{"very long", "#FF5500FF5500FF5500", false},
		{"hash only no digits", "#XXXXXX", false},
		
		// Boundary cases
		{"exactly 6 hex digits", "#123456", true},
		{"exactly 8 hex digits", "#12345678", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidHexColor(tt.input)
			if got != tt.valid {
				t.Errorf("IsValidHexColor(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
