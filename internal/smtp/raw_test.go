package smtp

import (
	"bytes"
	netmail "net/mail"
	"strings"
	"testing"
)

func TestCollectRecipients(t *testing.T) {
	raw := []byte("" +
		"From: alice@example.com\r\n" +
		"To: Bob <bob@example.com>, cc@example.com\r\n" +
		"Cc: Carol <carol@example.com>\r\n" +
		"Bcc: hidden@example.com\r\n" +
		"\r\nBody")
	msg, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	rcpts := collectRecipients(msg)
	if len(rcpts) != 4 {
		t.Fatalf("expected 4 recipients, got %d: %v", len(rcpts), rcpts)
	}
}

func TestFirstFromAddress(t *testing.T) {
	raw := []byte("From: Alice <alice@example.com>\r\nTo: bob@example.com\r\n\r\nHi")
	msg, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	from, err := firstFromAddress(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(from, "alice@example.com") {
		t.Fatalf("from=%q", from)
	}
}
