package commands

import (
	"testing"

	mailimap "go-cubemail/internal/imap"
)

func TestSelectMailBodyPrefersHTML(t *testing.T) {
	text, native := selectMailBody(&mailimap.ParsedMessage{
		TextPlain: "plain",
		TextHTML:  "<p>html</p>",
	}, bodyTypeHTML)
	if text != "<p>html</p>" || native != bodyTypeHTML {
		t.Fatalf("got text=%q native=%d", text, native)
	}
}

func TestSelectMailBodyPlainFallback(t *testing.T) {
	text, native := selectMailBody(&mailimap.ParsedMessage{TextPlain: "plain only"}, bodyTypeHTML)
	if text != "plain only" || native != bodyTypePlain {
		t.Fatalf("got text=%q native=%d", text, native)
	}
}

func TestResolveItemOpsIDsFromLongId(t *testing.T) {
	cid, sid := resolveItemOpsIDs(easItemOpsFetchRequest{LongID: "mail/abc123+42"})
	if cid != "mail/abc123" || sid != "42" {
		t.Fatalf("cid=%q sid=%q", cid, sid)
	}
}

func TestParseSearchRange(t *testing.T) {
	start, end := parseSearchRange("5-15")
	if start != 5 || end != 15 {
		t.Fatalf("start=%d end=%d", start, end)
	}
}
