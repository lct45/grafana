package pref

import "testing"

func TestIsValidThemeID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{id: "dark", valid: true},
		{id: "light", valid: true},
		{id: "system", valid: true},
		{id: "harbor", valid: true},
		{id: "not-a-real-theme", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := IsValidThemeID(tt.id); got != tt.valid {
				t.Fatalf("IsValidThemeID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

func TestGetThemeByID_Harbor(t *testing.T) {
	theme := GetThemeByID("harbor")
	if theme == nil {
		t.Fatal("expected harbor theme to be registered")
	}

	if theme.ID != "harbor" {
		t.Fatalf("theme ID = %q, want harbor", theme.ID)
	}

	if theme.Type != "dark" {
		t.Fatalf("theme type = %q, want dark", theme.Type)
	}

	if !theme.IsExtra {
		t.Fatal("expected harbor to be marked as an extra theme")
	}
}
