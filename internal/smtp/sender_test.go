package smtp

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	msg := &Message{
		From:        "nilton@example.com",
		DisplayName: "Nilton",
		To:          []string{"Recipient <recipient@example.com>", "plain@example.com"},
		Cc:          []string{"cc@example.com"},
		Bcc:         []string{"bcc@example.com"},
		Subject:     "Test Subject with Acentuação",
		TextPlain:   "Hello from plane text!",
		TextHTML:    `<html><body>Hello! <img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" /></body></html>`,
		Attachments: []Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        []byte("Hello attachment!"),
			},
		},
	}

	m, err := buildMessage(msg)
	if err != nil {
		t.Fatalf("buildMessage returned unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if _, err := m.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo returned unexpected error: %v", err)
	}

	raw := buf.String()

	// Verify headers
	if !strings.Contains(raw, `From: "Nilton" <nilton@example.com>`) {
		t.Errorf("Expected From header, got: %s", raw)
	}
	if !strings.Contains(raw, `To: "Recipient" <recipient@example.com>, <plain@example.com>`) {
		t.Errorf("Expected To headers, got: %s", raw)
	}
	if !strings.Contains(raw, "Cc: <cc@example.com>") {
		t.Errorf("Expected Cc header, got: %s", raw)
	}
	if !strings.Contains(raw, "Subject: =?UTF-8?q?Test_Subject_with_Acentua=C3=A7=C3=A3o?=") {
		t.Errorf("Expected Q-encoded Subject header, got: %s", raw)
	}

	// Verify plaintext body
	if !strings.Contains(raw, "Hello from plane text!") {
		t.Errorf("Expected plain text body, got: %s", raw)
	}

	// Verify html body and inline image extraction
	if !strings.Contains(raw, "cid:inline-img-1") {
		t.Errorf("Expected HTML to refer to inline image CID, got: %s", raw)
	}
	if !strings.Contains(raw, "Content-Id: inline-img-1") {
		t.Errorf("Expected inline image attachment part, got: %s", raw)
	}

	// Verify file attachment
	if !strings.Contains(raw, "filename=\"test.txt\"") {
		t.Errorf("Expected test.txt attachment part, got: %s", raw)
	}
	if !strings.Contains(raw, "SGVsbG8gYXR0YWNobWVudCE=") { // base64 of "Hello attachment!"
		t.Errorf("Expected base64 encoded attachment content, got: %s", raw)
	}
}
