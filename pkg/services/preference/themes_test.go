package pref

import "testing"

func TestIsValidThemeID(t *testing.T) {
	t.Run("accepts first-class Cotton Candy theme", func(t *testing.T) {
		if !IsValidThemeID("cottoncandy") {
			t.Fatal("expected cottoncandy to be a valid theme id")
		}
	})

	t.Run("accepts core themes", func(t *testing.T) {
		for _, id := range []string{"light", "dark", "system"} {
			if !IsValidThemeID(id) {
				t.Fatalf("expected %q to be a valid theme id", id)
			}
		}
	})

	t.Run("rejects unknown themes", func(t *testing.T) {
		if IsValidThemeID("not-a-theme") {
			t.Fatal("expected unknown theme id to be invalid")
		}
	})
}

func TestGetThemeByID(t *testing.T) {
	t.Run("returns Cotton Candy as a light theme", func(t *testing.T) {
		theme := GetThemeByID("cottoncandy")
		if theme == nil {
			t.Fatal("expected cottoncandy theme to exist")
		}
		if theme.Type != "light" {
			t.Fatalf("expected cottoncandy theme type light, got %q", theme.Type)
		}
		if theme.IsExtra {
			t.Fatal("expected cottoncandy to be a first-class theme")
		}
	})
}
