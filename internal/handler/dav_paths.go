package handler

// DAV URL construction and parsing.
//
// Every DAV path is parsed in exactly one place. Path handling spread across
// handlers is where traversal bugs and cross-user leaks come from, so the rules
// live here: segments are percent-decoded once, a decoded segment may never
// contain a slash or "..", and the user segment is compared against the
// authenticated user by the caller.

import (
	"net/url"
	"strings"
)

// davResourceType identifies which DAV resource a path addresses.
type davResourceType int

const (
	davUnknown davResourceType = iota
	davRoot                    // /dav/
	davPrincipal               // /dav/{user}/
	davCalendarHome            // /dav/{user}/calendars/
	davCalendar                // /dav/{user}/calendars/{uri}/
	davCalendarObject          // /dav/{user}/calendars/{uri}/{resource}
	davAddressBookHome         // /dav/{user}/contacts/
	davAddressBook             // /dav/{user}/contacts/{uri}/
	davAddressObject           // /dav/{user}/contacts/{uri}/{resource}
)

// davPath is a parsed DAV request path.
type davPath struct {
	Type       davResourceType
	User       string
	Collection string
	Resource   string
}

// Path segment names for the two collection roots.
const (
	calendarsSegment = "calendars"
	contactsSegment  = "contacts"
)

// parseDAVPath decodes a request path below /dav into its parts. Anything that
// does not match the documented layout is reported as davUnknown so callers
// answer 404 instead of guessing.
func parseDAVPath(p string) davPath {
	trimmed := strings.TrimPrefix(p, "/dav")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return davPath{Type: davRoot}
	}

	raw := strings.Split(trimmed, "/")
	segs := make([]string, 0, len(raw))
	for _, s := range raw {
		dec, err := url.PathUnescape(s)
		if err != nil || dec == "" || dec == "." || dec == ".." || strings.ContainsAny(dec, "/\\") {
			return davPath{Type: davUnknown}
		}
		segs = append(segs, dec)
	}

	out := davPath{User: segs[0]}
	switch len(segs) {
	case 1:
		out.Type = davPrincipal
	case 2:
		switch segs[1] {
		case calendarsSegment:
			out.Type = davCalendarHome
		case contactsSegment:
			out.Type = davAddressBookHome
		default:
			out.Type = davUnknown
		}
	case 3:
		out.Collection = segs[2]
		switch segs[1] {
		case calendarsSegment:
			out.Type = davCalendar
		case contactsSegment:
			out.Type = davAddressBook
		default:
			out.Type = davUnknown
		}
	case 4:
		out.Collection = segs[2]
		out.Resource = segs[3]
		switch segs[1] {
		case calendarsSegment:
			out.Type = davCalendarObject
		case contactsSegment:
			out.Type = davAddressObject
		default:
			out.Type = davUnknown
		}
	default:
		out.Type = davUnknown
	}
	return out
}

// isCalendarScope reports whether the path addresses CalDAV resources.
func (p davPath) isCalendarScope() bool {
	switch p.Type {
	case davCalendarHome, davCalendar, davCalendarObject:
		return true
	}
	return false
}

// isAddressBookScope reports whether the path addresses CardDAV resources.
func (p davPath) isAddressBookScope() bool {
	switch p.Type {
	case davAddressBookHome, davAddressBook, davAddressObject:
		return true
	}
	return false
}

// ── URL construction ──────────────────────────────────────────────────────
//
// Hrefs are emitted as absolute paths rather than absolute URLs. A path is
// always correct regardless of how the reverse proxy presents the host, whereas
// an absolute URL built from a misconfigured base_url silently sends clients to
// the wrong origin.

func escapeSegment(s string) string { return url.PathEscape(s) }

func principalHref(user string) string {
	return "/dav/" + escapeSegment(user) + "/"
}

func calendarHomeHref(user string) string {
	return principalHref(user) + calendarsSegment + "/"
}

func addressBookHomeHref(user string) string {
	return principalHref(user) + contactsSegment + "/"
}

func calendarHref(user, collection string) string {
	return calendarHomeHref(user) + escapeSegment(collection) + "/"
}

func addressBookHref(user, collection string) string {
	return addressBookHomeHref(user) + escapeSegment(collection) + "/"
}

func calendarObjectHref(user, collection, resource string) string {
	return calendarHref(user, collection) + escapeSegment(resource)
}

func addressObjectHref(user, collection, resource string) string {
	return addressBookHref(user, collection) + escapeSegment(resource)
}
