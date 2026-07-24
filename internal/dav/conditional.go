package dav

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"path"
	"strings"
)

// ErrPreconditionFailed signals that an If-Match / If-None-Match header did not
// hold and the request must be answered with 412 Precondition Failed.
// Honouring it is what stops two clients from silently overwriting each other.
var ErrPreconditionFailed = errors.New("dav: precondition failed")

// CheckPreconditions evaluates the conditional headers of a PUT or DELETE
// against the current state of the target resource.
//
//	If-None-Match: *          create only — fail when the resource exists
//	If-Match: "etag"          update only — fail when it is gone or changed
//	If-Match: *               fail when the resource does not exist
func CheckPreconditions(h http.Header, exists bool, currentETag string) error {
	// MatchETag treats "*" as matching anything, so the wildcard create-only
	// case is covered by the same check as a specific tag.
	if inm := h.Get("If-None-Match"); inm != "" {
		if exists && MatchETag(inm, currentETag) {
			return ErrPreconditionFailed
		}
	}
	if im := h.Get("If-Match"); im != "" {
		if !exists {
			return ErrPreconditionFailed
		}
		if !MatchETag(im, currentETag) {
			return ErrPreconditionFailed
		}
	}
	return nil
}

// NewResourceURI mints a random resource name with the given extension, used
// when a client creates an object through the REST API rather than a DAV PUT.
func NewResourceURI(ext string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) + ext
}

// ResourceURIFromHref extracts the last path segment of a DAV href, which is
// the resource name inside its collection. Any directory component is dropped,
// so a crafted href cannot escape the collection it was sent to.
func ResourceURIFromHref(href string) string {
	h := strings.TrimSpace(href)
	if h == "" {
		return ""
	}
	if i := strings.IndexAny(h, "?#"); i >= 0 {
		h = h[:i]
	}
	h = strings.TrimSuffix(h, "/")
	name := path.Base(h)
	if name == "." || name == "/" || name == ".." {
		return ""
	}
	return name
}
