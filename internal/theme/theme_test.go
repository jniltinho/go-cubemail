package theme

import "testing"

func TestByIDAndValid(t *testing.T) {
	tests := []struct {
		id         string
		wantID     string
		wantLabel  string
		wantScheme string
		wantOK     bool
	}{
		{"larry", "larry", "Larry", "light", true},
		{"outlook", "outlook", "Outlook", "light", true},
		{"outlook-dark", "outlook-dark", "Outlook (dark)", "dark", true},
		{"apple-mail", "apple-mail", "Apple Mail", "light", true},
		{"apple-mail-dark", "apple-mail-dark", "Apple Mail (dark)", "dark", true},
		{"breeze", "breeze", "Breeze", "light", true},
		{"breeze-dark", "breeze-dark", "Breeze (dark)", "dark", true},
		{"graphite", "graphite", "Graphite (dark)", "dark", true},
		{"high-contrast", "high-contrast", "High contrast (dark)", "dark", true},
		// Unknown ids fall back to the first entry.
		{"nope", "larry", "Larry", "light", false},
		{"", "larry", "Larry", "light", false},
		{"Outlook", "larry", "Larry", "light", false},
		{" outlook", "larry", "Larry", "light", false},
	}
	for _, tt := range tests {
		got, ok := ByID(tt.id)
		if ok != tt.wantOK || got.ID != tt.wantID || got.Label != tt.wantLabel || got.Scheme != tt.wantScheme {
			t.Errorf("ByID(%q) = %+v, %v; want {%s %s %s}, %v",
				tt.id, got, ok, tt.wantID, tt.wantLabel, tt.wantScheme, tt.wantOK)
		}
		if Valid(tt.id) != tt.wantOK {
			t.Errorf("Valid(%q) = %v, want %v", tt.id, !tt.wantOK, tt.wantOK)
		}
	}
}

func TestListOrderAndCopy(t *testing.T) {
	// The list is generated (registry_gen.go): check its shape, not its length.
	got := List()
	if len(got) == 0 || got[0].ID != "larry" {
		t.Fatalf("List() must start with larry, got %+v", got)
	}
	seen := map[string]bool{}
	for _, th := range got {
		if seen[th.ID] {
			t.Errorf("theme %q is listed twice", th.ID)
		}
		seen[th.ID] = true
		if th.Label == "" || (th.Scheme != "light" && th.Scheme != "dark") {
			t.Errorf("theme %q has label %q and scheme %q", th.ID, th.Label, th.Scheme)
		}
	}

	// Callers must not be able to change the registry through the result.
	got[0].ID = "changed"
	if List()[0].ID != "larry" {
		t.Error("List() exposes the registry's backing array")
	}
}
