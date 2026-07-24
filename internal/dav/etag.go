// Package dav implements the synchronisation primitives shared by the CalDAV
// and CardDAV servers: per-object ETags, per-collection CTags, RFC 6578
// sync-tokens and the transactional changelog that makes deletions visible to
// clients.
//
// These three mechanisms are distinct and all three are required:
//
//	ETag       identifies one version of one resource (conditional requests)
//	CTag       changes when anything in the collection changes (legacy clients)
//	sync-token lets a client ask "what changed since revision N?" (RFC 6578)
package dav

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ComputeETag returns the unquoted entity tag for a resource body: the first
// 16 bytes of its SHA-256 digest, hex encoded. Storing it unquoted keeps the
// database value clean; use Quote when emitting it over HTTP or in XML.
func ComputeETag(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}

// Quote wraps an entity tag in the double quotes that are part of the HTTP
// header value (RFC 9110 §8.8.3). Already-quoted input is returned unchanged.
func Quote(etag string) string {
	if etag == "" {
		return ""
	}
	if strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`) {
		return etag
	}
	return `"` + etag + `"`
}

// Unquote strips the surrounding quotes and any weak-comparison prefix from a
// single entity tag, yielding the opaque value for comparison.
func Unquote(etag string) string {
	s := strings.TrimSpace(etag)
	s = strings.TrimPrefix(s, "W/")
	s = strings.TrimPrefix(s, "w/")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	return s
}

// MatchETag reports whether an If-Match / If-None-Match header value matches
// the resource's current entity tag. The header may hold "*", a single tag, or
// a comma-separated list. An empty header never matches.
func MatchETag(header, current string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	if header == "*" {
		return true
	}
	for _, part := range strings.Split(header, ",") {
		if Unquote(part) == Unquote(current) {
			return true
		}
	}
	return false
}
