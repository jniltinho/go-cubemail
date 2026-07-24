package contacts

import (
	"strings"
	"testing"
	"time"

	"go-cubemail/internal/model"
)

func TestUnfoldJoinsContinuationLines(t *testing.T) {
	// Apple Contacts folds long values; a parser that ignores continuation
	// lines reads the remainder as a bogus property.
	raw := "BEGIN:VCARD\r\n" +
		"NOTE:This note is long enough that the client had to fold it across\r\n" +
		"  several physical lines of the file\r\n" +
		"END:VCARD\r\n"

	lines := Unfold(raw)
	var note string
	for _, l := range lines {
		if strings.HasPrefix(l, "NOTE:") {
			note = l
		}
	}
	if !strings.Contains(note, "several physical lines") {
		t.Fatalf("folded value was not rejoined: %q", note)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 logical lines, got %d: %q", len(lines), lines)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		card      string
		wantFirst string
		wantLast  string
		wantEmail string
		wantPhone string
	}{
		{
			name: "structured name and first e-mail win",
			card: "BEGIN:VCARD\r\nVERSION:3.0\r\n" +
				"N:Ribeiro;Ana;;;\r\nFN:Ana Ribeiro\r\n" +
				"EMAIL;TYPE=INTERNET:ana@example.com\r\n" +
				"EMAIL;TYPE=WORK:ana@work.com\r\nEND:VCARD\r\n",
			wantFirst: "Ana", wantLast: "Ribeiro", wantEmail: "ana@example.com",
		},
		{
			// A card with FN but no N is legal and some clients send exactly that.
			name: "FN without N still yields a name",
			card: "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Bruno Costa\r\n" +
				"EMAIL:bruno@example.com\r\nEND:VCARD\r\n",
			wantFirst: "Bruno", wantLast: "Costa", wantEmail: "bruno@example.com",
		},
		{
			name: "preferred entries beat position",
			card: "BEGIN:VCARD\r\nFN:Carla\r\n" +
				"EMAIL;TYPE=INTERNET:first@example.com\r\n" +
				"EMAIL;TYPE=INTERNET,PREF:preferred@example.com\r\n" +
				"TEL;TYPE=HOME:+551130000000\r\n" +
				"TEL;TYPE=CELL,PREF:+5511900000000\r\nEND:VCARD\r\n",
			wantFirst: "Carla", wantEmail: "preferred@example.com", wantPhone: "+5511900000000",
		},
		{
			name: "grouped properties are recognised",
			card: "BEGIN:VCARD\r\nFN:Dora Lima\r\n" +
				"item1.EMAIL;TYPE=INTERNET:dora@example.com\r\n" +
				"item1.X-ABLabel:_$!<Work>!$_\r\nEND:VCARD\r\n",
			wantFirst: "Dora", wantLast: "Lima", wantEmail: "dora@example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.card)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.FirstName != tc.wantFirst {
				t.Errorf("first name = %q, want %q", got.FirstName, tc.wantFirst)
			}
			if tc.wantLast != "" && got.LastName != tc.wantLast {
				t.Errorf("last name = %q, want %q", got.LastName, tc.wantLast)
			}
			if got.Email != tc.wantEmail {
				t.Errorf("email = %q, want %q", got.Email, tc.wantEmail)
			}
			if tc.wantPhone != "" && got.Phone != tc.wantPhone {
				t.Errorf("phone = %q, want %q", got.Phone, tc.wantPhone)
			}
		})
	}
}

func TestParseRejectsEmptyCard(t *testing.T) {
	if _, err := Parse("BEGIN:VCARD\r\nVERSION:3.0\r\nEND:VCARD\r\n"); err == nil {
		t.Fatal("a card with no name and no e-mail should be rejected")
	}
}

func TestParseHandlesQuotedParameterColons(t *testing.T) {
	card := "BEGIN:VCARD\r\nFN:Eva\r\n" + `TEL;TYPE="work:home":+5511999999999` + "\r\nEND:VCARD\r\n"
	got, err := Parse(card)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phone != "+5511999999999" {
		t.Fatalf("phone = %q — the value was cut at a colon inside a quoted parameter", got.Phone)
	}
}

// The core guarantee: editing the fields the UI knows about must not destroy
// the rest of the card.
func TestApplyToVCardPreservesUnknownProperties(t *testing.T) {
	original := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"UID:abc@test\r\n" +
		"FN:Ana Ribeiro\r\n" +
		"N:Ribeiro;Ana;;;\r\n" +
		"EMAIL;TYPE=INTERNET:ana@example.com\r\n" +
		"EMAIL;TYPE=WORK:ana@work.com\r\n" +
		"TEL;TYPE=CELL:+5511900000000\r\n" +
		"ADR;TYPE=HOME:;;Rua das Flores 100;Sao Paulo;SP;01000-000;Brazil\r\n" +
		"PHOTO;ENCODING=b;TYPE=JPEG:/9j/4AAQSkZJRg==\r\n" +
		"BDAY:19850312\r\n" +
		"X-CUSTOM:keep-me\r\n" +
		"END:VCARD\r\n"

	edited := model.Contact{
		FirstName: "Ana", LastName: "Ribeiro",
		Email: "ana.new@example.com", Phone: "+5511900000000",
		Title: "Engineer", UpdatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	out := ApplyToVCard(original, edited)

	for _, keep := range []string{
		"UID:abc@test", "ADR;TYPE=HOME", "PHOTO;ENCODING=b", "BDAY:19850312",
		"X-CUSTOM:keep-me", "ana@work.com", "VERSION:3.0",
	} {
		if !strings.Contains(out, keep) {
			t.Errorf("lost %q from the stored card:\n%s", keep, out)
		}
	}
	if !strings.Contains(out, "ana.new@example.com") {
		t.Errorf("the edited e-mail was not applied:\n%s", out)
	}
	if strings.Contains(out, "EMAIL;TYPE=INTERNET:ana@example.com") {
		t.Errorf("the old primary e-mail should have been replaced:\n%s", out)
	}
	// A property the original lacked is appended.
	if !strings.Contains(out, "TITLE:Engineer") {
		t.Errorf("new property was not added:\n%s", out)
	}
	if !strings.HasSuffix(out, "END:VCARD\r\n") {
		t.Errorf("card must end with END:VCARD:\n%s", out)
	}
	if strings.Count(out, "END:VCARD") != 1 {
		t.Errorf("card has a duplicated END:VCARD:\n%s", out)
	}
}

func TestApplyToVCardBuildsFromScratchWhenEmpty(t *testing.T) {
	out := ApplyToVCard("", model.Contact{FirstName: "Novo", LastName: "Contato",
		Email: "novo@example.com"})
	if !strings.HasPrefix(out, "BEGIN:VCARD") || !strings.Contains(out, "FN:Novo Contato") {
		t.Fatalf("a contact without a stored card must get a fresh one:\n%s", out)
	}
}

func TestApplyToVCardDropsClearedFields(t *testing.T) {
	original := "BEGIN:VCARD\r\nFN:Ana\r\nTITLE:Engineer\r\nEMAIL:ana@example.com\r\nEND:VCARD\r\n"
	out := ApplyToVCard(original, model.Contact{FirstName: "Ana", Email: "ana@example.com"})
	if strings.Contains(out, "TITLE:") {
		t.Fatalf("a field cleared in the UI must be removed from the card:\n%s", out)
	}
}

func TestFoldLineRespectsTheOctetLimit(t *testing.T) {
	long := "NOTE:" + strings.Repeat("a", 200)
	folded := FoldLine(long)
	for _, line := range strings.Split(strings.TrimRight(folded, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Fatalf("line exceeds 75 octets (%d): %q", len(line), line)
		}
	}
	// Continuation lines must start with a single space.
	parts := strings.Split(strings.TrimRight(folded, "\r\n"), "\r\n")
	for _, line := range parts[1:] {
		if !strings.HasPrefix(line, " ") {
			t.Fatalf("continuation line lacks the leading space: %q", line)
		}
	}
	// Unfolding must recover the original content.
	rejoined := Unfold(folded)
	if len(rejoined) != 1 || rejoined[0] != long {
		t.Fatalf("fold/unfold round trip failed:\n%q", rejoined)
	}
}

func TestFoldLineNeverSplitsUTF8(t *testing.T) {
	long := "NOTE:" + strings.Repeat("ação", 40)
	folded := FoldLine(long)
	rejoined := Unfold(folded)
	if len(rejoined) != 1 || rejoined[0] != long {
		t.Fatalf("multi-byte characters were corrupted by folding:\n%q", rejoined)
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	for _, value := range []string{
		`Simple`, `With, comma`, `With; semicolon`, "With\nnewline", `Back\slash`,
		`All, of; them\and` + "\nmore",
	} {
		if got := Unescape(Escape(value)); got != value {
			t.Fatalf("round trip of %q gave %q", value, got)
		}
	}
}
