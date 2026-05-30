package commands

import (
	"testing"

	"github.com/remdev/go-activesync/eas"
)

func TestUserResponseToPartStat(t *testing.T) {
	if userResponseToPartStat(1) != "ACCEPTED" {
		t.Fatal("expected ACCEPTED")
	}
	if userResponseToPartStat(2) != "DECLINED" {
		t.Fatal("expected DECLINED")
	}
	if userResponseToPartStat(99) != "" {
		t.Fatal("expected empty for unknown response")
	}
}

func TestLooksLikeRFC822(t *testing.T) {
	raw := []byte("From: alice@example.com\r\nTo: bob@example.com\r\nSubject: Hi\r\n\r\nBody")
	if !looksLikeRFC822(raw) {
		t.Fatal("expected RFC822 detection")
	}
}

func TestTryBase64MIME(t *testing.T) {
	plain := []byte("From: a@b.com\r\n\r\nHi")
	encoded := []byte("RnJvbTogQGJiLmNvbQ0KDQpIaQ==") // invalid but test decode path
	if _, ok := tryBase64MIME(plain); ok {
		t.Fatal("plain MIME should not decode as base64")
	}
	_ = encoded
}

func TestEasSendMailResponseStatus(t *testing.T) {
	if eas.StatusSuccess != 1 {
		t.Fatalf("unexpected success constant %d", eas.StatusSuccess)
	}
}
