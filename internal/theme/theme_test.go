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
	want := []string{
		"larry", "outlook", "outlook-dark", "apple-mail", "apple-mail-dark",
		"breeze", "breeze-dark", "graphite", "high-contrast",
	}
	got := List()
	if len(got) != len(want) {
		t.Fatalf("List() has %d themes, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("List()[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}

	// Callers must not be able to change the registry through the result.
	got[0].ID = "changed"
	if List()[0].ID != "larry" {
		t.Error("List() exposes the registry's backing array")
	}
}
