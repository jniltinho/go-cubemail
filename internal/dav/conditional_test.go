package dav

import (
	"net/http"
	"testing"
)

func TestCheckPreconditions(t *testing.T) {
	const current = "abc123"

	tests := []struct {
		name    string
		headers map[string]string
		exists  bool
		wantErr bool
	}{
		{
			name:    "no conditional headers always passes",
			headers: nil, exists: true, wantErr: false,
		},
		{
			name:    "create-only on a free resource",
			headers: map[string]string{"If-None-Match": "*"}, exists: false, wantErr: false,
		},
		{
			name:    "create-only on a taken resource is refused",
			headers: map[string]string{"If-None-Match": "*"}, exists: true, wantErr: true,
		},
		{
			name:    "update with the current tag",
			headers: map[string]string{"If-Match": `"abc123"`}, exists: true, wantErr: false,
		},
		{
			name:    "update with a stale tag is refused",
			headers: map[string]string{"If-Match": `"stale"`}, exists: true, wantErr: true,
		},
		{
			name:    "update of a deleted resource is refused",
			headers: map[string]string{"If-Match": `"abc123"`}, exists: false, wantErr: true,
		},
		{
			name:    "If-Match wildcard requires existence",
			headers: map[string]string{"If-Match": "*"}, exists: false, wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tc.headers {
				h.Set(k, v)
			}
			err := CheckPreconditions(h, tc.exists, current)
			if tc.wantErr && err == nil {
				t.Fatal("expected the request to be refused with 412")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
		})
	}
}

func TestResourceURIFromHref(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{href: "/dav/user/calendars/default/a1b2.ics", want: "a1b2.ics"},
		{href: "a1b2.ics", want: "a1b2.ics"},
		{href: "http://host/dav/user/contacts/default/x.vcf", want: "x.vcf"},
		{href: "/dav/user/calendars/default/x.ics?rev=2", want: "x.ics"},
		{href: "", want: ""},
		{href: "/", want: ""},
		// A crafted href must not reach outside the collection it was sent to.
		{href: "/dav/user/calendars/default/../../other/secret.ics", want: "secret.ics"},
		{href: "..", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.href, func(t *testing.T) {
			if got := ResourceURIFromHref(tc.href); got != tc.want {
				t.Fatalf("ResourceURIFromHref(%q) = %q, want %q", tc.href, got, tc.want)
			}
		})
	}
}

func TestNewResourceURIIsUniqueAndSuffixed(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uri := NewResourceURI(".ics")
		if seen[uri] {
			t.Fatalf("duplicate resource name generated: %s", uri)
		}
		seen[uri] = true
		if len(uri) < 5 || uri[len(uri)-4:] != ".ics" {
			t.Fatalf("unexpected resource name: %s", uri)
		}
	}
}
