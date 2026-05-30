package commands

import (
	"testing"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
)

func TestProvisionHandler(t *testing.T) {
	h := &ProvisionHandler{}
	out, err := h.Handle(&Context{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected WBXML output")
	}

	var resp eas.ProvisionResponse
	if err := wbxml.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != eas.StatusSuccess {
		t.Fatalf("status=%d", resp.Status)
	}
	if len(resp.Policies.Policy) != 1 || resp.Policies.Policy[0].Status != 2 {
		t.Fatalf("unexpected policy response: %+v", resp.Policies.Policy)
	}
}

func TestPingHandler(t *testing.T) {
	h := NewPingHandler(nil, nil, nil, nil, nil)
	out, err := h.Handle(&Context{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var resp eas.PingResponse
	if err := wbxml.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != 1 {
		t.Fatalf("status=%d", resp.Status)
	}
}
