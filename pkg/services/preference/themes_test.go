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
		{id: "ember", valid: true},
		{id: "gloom", valid: true},
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

func TestGetThemeByID_Ember(t *testing.T) {
	theme := GetThemeByID("ember")
	if theme == nil {
		t.Fatal("expected ember theme to be registered")
	}
	if theme.Type != "dark" {
		t.Fatalf("expected ember theme type dark, got %q", theme.Type)
	}
	if !theme.IsExtra {
		t.Fatal("expected ember theme to be marked as extra")
	}
}
