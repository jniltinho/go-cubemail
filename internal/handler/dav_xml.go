package handler

// WebDAV XML plumbing shared by the CalDAV and CardDAV handlers.
//
// Requests are decoded with encoding/xml rather than matched with substring
// checks: a client asking for three properties must get those three, and a
// property the server does not know must come back in a 404 propstat instead of
// being silently dropped. Responses are still assembled as strings, but every
// value goes through XML escaping and every element through the namespace
// prefix table below.

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// XML namespaces used across CalDAV and CardDAV.
const (
	nsDAV     = "DAV:"
	nsCalDAV  = "urn:ietf:params:xml:ns:caldav"
	nsCardDAV = "urn:ietf:params:xml:ns:carddav"
	nsCS      = "http://calendarserver.org/ns/"
	nsApple   = "http://apple.com/ns/ical/"
)

// nsPrefixes maps namespaces to the prefixes declared on the multistatus root.
var nsPrefixes = map[string]string{
	nsDAV:     "D",
	nsCalDAV:  "C",
	nsCardDAV: "CR",
	nsCS:      "CS",
	nsApple:   "ICAL",
}

const multistatusOpen = `<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" ` +
	`xmlns:CR="urn:ietf:params:xml:ns:carddav" xmlns:CS="http://calendarserver.org/ns/" ` +
	`xmlns:ICAL="http://apple.com/ns/ical/">`

// ── Request parsing ───────────────────────────────────────────────────────

// anyElement captures the name of an arbitrary child element.
type anyElement struct {
	XMLName xml.Name
}

// propElement is the DAV:prop container listing requested property names.
type propElement struct {
	Props []anyElement `xml:",any"`
}

// propfindRequest is a decoded PROPFIND body.
type propfindRequest struct {
	AllProp  bool
	PropName bool
	Props    []xml.Name
}

// wants reports whether the request asks for the given property.
func (r propfindRequest) wants(space, local string) bool {
	if r.AllProp {
		return true
	}
	return r.wantsExplicit(space, local)
}

// wantsExplicit reports whether the property was named outright.
//
// It is the right test for calendar-data and address-data: those carry the
// whole object body, and RFC 4918 allprop covers live properties, not payloads.
// Treating allprop as "send everything" would put every .ics and .vcf in the
// collection into a single PROPFIND response.
func (r propfindRequest) wantsExplicit(space, local string) bool {
	for _, p := range r.Props {
		if p.Space == space && p.Local == local {
			return true
		}
	}
	return false
}

// parsePropfind decodes a PROPFIND body. An empty body means allprop, which is
// what several clients send on their first request.
func parsePropfind(body []byte) propfindRequest {
	if len(bytes.TrimSpace(body)) == 0 {
		return propfindRequest{AllProp: true}
	}
	var doc struct {
		XMLName  xml.Name
		AllProp  *struct{}    `xml:"DAV: allprop"`
		PropName *struct{}    `xml:"DAV: propname"`
		Prop     *propElement `xml:"DAV: prop"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return propfindRequest{AllProp: true}
	}
	req := propfindRequest{
		AllProp:  doc.AllProp != nil,
		PropName: doc.PropName != nil,
	}
	if doc.Prop != nil {
		for _, p := range doc.Prop.Props {
			req.Props = append(req.Props, p.XMLName)
		}
	}
	if !req.AllProp && !req.PropName && len(req.Props) == 0 {
		req.AllProp = true
	}
	return req
}

// reportName returns the qualified name of a REPORT body's root element, which
// is what selects the report type.
func reportName(body []byte) (xml.Name, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if err != nil {
			return xml.Name{}, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name, nil
		}
	}
}

// syncCollectionRequest is a decoded RFC 6578 sync-collection REPORT.
type syncCollectionRequest struct {
	SyncToken string      `xml:"DAV: sync-token"`
	SyncLevel string      `xml:"DAV: sync-level"`
	Limit     *limitElem  `xml:"DAV: limit"`
	Prop      propElement `xml:"DAV: prop"`
}

type limitElem struct {
	NResults int `xml:"DAV: nresults"`
}

// multigetRequest is a decoded calendar-multiget / addressbook-multiget REPORT.
type multigetRequest struct {
	Prop  propElement `xml:"DAV: prop"`
	Hrefs []string    `xml:"DAV: href"`
}

// timeRangeElem is the CalDAV time-range filter.
type timeRangeElem struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

// compFilterElem is one level of a CalDAV comp-filter tree.
type compFilterElem struct {
	Name      string           `xml:"name,attr"`
	Comps     []compFilterElem `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
	TimeRange *timeRangeElem   `xml:"urn:ietf:params:xml:ns:caldav time-range"`
}

// calendarQueryRequest is a decoded CalDAV calendar-query REPORT.
type calendarQueryRequest struct {
	Prop   propElement `xml:"DAV: prop"`
	Filter struct {
		Comp compFilterElem `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
	} `xml:"urn:ietf:params:xml:ns:caldav filter"`
}

// addressbookQueryRequest is a decoded CardDAV addressbook-query REPORT.
type addressbookQueryRequest struct {
	Prop   propElement `xml:"DAV: prop"`
	Filter struct {
		Test  string `xml:"test,attr"`
		Props []struct {
			Name      string `xml:"name,attr"`
			TextMatch string `xml:"urn:ietf:params:xml:ns:carddav text-match"`
		} `xml:"urn:ietf:params:xml:ns:carddav prop-filter"`
	} `xml:"urn:ietf:params:xml:ns:carddav filter"`
	Limit *struct {
		NResults int `xml:"urn:ietf:params:xml:ns:carddav nresults"`
	} `xml:"urn:ietf:params:xml:ns:carddav limit"`
}

// proppatchRequest is a decoded PROPPATCH body: the properties to set and to
// remove, in document order.
type proppatchRequest struct {
	Set    []proppatchProp
	Remove []xml.Name
}

type proppatchProp struct {
	Name  xml.Name
	Value string
}

// parseProppatch decodes a PROPPATCH body by streaming, because property values
// are arbitrary XML that no fixed struct can describe.
func parseProppatch(body []byte) proppatchRequest {
	var out proppatchRequest
	dec := xml.NewDecoder(bytes.NewReader(body))
	var mode string // "set" | "remove"
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Space == nsDAV && t.Name.Local == "set":
				mode, depth = "set", 0
			case t.Name.Space == nsDAV && t.Name.Local == "remove":
				mode, depth = "remove", 0
			case t.Name.Space == nsDAV && t.Name.Local == "prop":
				depth = 1
			case depth == 1 && mode != "":
				var value string
				if err := dec.DecodeElement(&value, &t); err != nil {
					value = ""
				}
				if mode == "set" {
					out.Set = append(out.Set, proppatchProp{Name: t.Name, Value: value})
				} else {
					out.Remove = append(out.Remove, t.Name)
				}
			}
		case xml.EndElement:
			if t.Name.Space == nsDAV && (t.Name.Local == "set" || t.Name.Local == "remove") {
				mode = ""
			}
			if t.Name.Space == nsDAV && t.Name.Local == "prop" {
				depth = 0
			}
		}
	}
}

// Request body budgets. Property and collection-creation documents are small;
// a REPORT may carry a long href list, so it gets more room.
const (
	davRequestBodyLimit = 64 * 1024
	davReportBodyLimit  = 1 << 20
)

// readBody reads a request body with an upper bound so a hostile client cannot
// exhaust memory.
//
// tooLarge reports that the client sent more than the limit. The caller must
// reject the request instead of acting on the truncated payload: silently
// storing a cut-off vCard or calendar object corrupts the resource, and the
// client has no way to find out — it gets a 201 and a valid-looking ETag.
func readBody(c *echo.Context, limit int64) (body []byte, tooLarge bool) {
	// Read one byte past the limit: that byte existing is what distinguishes a
	// payload that exactly fills the budget from one that overflows it.
	data, err := io.ReadAll(io.LimitReader(c.Request().Body, limit+1))
	if err != nil {
		return nil, false
	}
	if int64(len(data)) > limit {
		return nil, true
	}
	return data, false
}

// writeTooLarge rejects an oversized resource with the precondition element the
// DAV specs define for it (CALDAV/CARDDAV:max-resource-size), so a client can
// tell "too big" apart from a generic refusal.
func writeTooLarge(c *echo.Context, ns string) error {
	prefix := nsPrefixes[ns]
	return writeDAVError(c, http.StatusRequestEntityTooLarge,
		`<`+prefix+`:max-resource-size xmlns:`+prefix+`="`+ns+`"/>`)
}

// ── Response building ─────────────────────────────────────────────────────

// propBag accumulates the properties available on a resource, preserving
// insertion order so responses are deterministic.
type propBag struct {
	order []xml.Name
	vals  map[xml.Name]string
}

func newPropBag() *propBag {
	return &propBag{vals: make(map[xml.Name]string)}
}

// setRaw stores a property whose inner content is already valid XML.
func (b *propBag) setRaw(space, local, inner string) {
	name := xml.Name{Space: space, Local: local}
	if _, exists := b.vals[name]; !exists {
		b.order = append(b.order, name)
	}
	b.vals[name] = inner
}

// setText stores a property whose inner content is character data to escape.
func (b *propBag) setText(space, local, text string) {
	b.setRaw(space, local, escapeXML(text))
}

// render turns the bag into propstat groups honouring what the client asked
// for. Requested properties the resource does not have come back as 404, which
// is what RFC 4918 requires and what stricter clients check.
func (b *propBag) render(req propfindRequest) []propstatOut {
	if req.PropName {
		var names []string
		for _, n := range b.order {
			names = append(names, emptyElement(n))
		}
		if len(names) == 0 {
			return nil
		}
		return []propstatOut{{Status: statusOK, Body: strings.Join(names, "")}}
	}

	if req.AllProp {
		var found []string
		for _, n := range b.order {
			found = append(found, element(n, b.vals[n]))
		}
		if len(found) == 0 {
			return nil
		}
		return []propstatOut{{Status: statusOK, Body: strings.Join(found, "")}}
	}

	var found, missing []string
	for _, n := range req.Props {
		if inner, ok := b.vals[n]; ok {
			found = append(found, element(n, inner))
			continue
		}
		missing = append(missing, emptyElement(n))
	}
	var out []propstatOut
	if len(found) > 0 {
		out = append(out, propstatOut{Status: statusOK, Body: strings.Join(found, "")})
	}
	if len(missing) > 0 {
		out = append(out, propstatOut{Status: statusNotFound, Body: strings.Join(missing, "")})
	}
	return out
}

// HTTP status lines used inside multistatus bodies.
const (
	statusOK       = "HTTP/1.1 200 OK"
	statusNotFound = "HTTP/1.1 404 Not Found"
	statusForbidden = "HTTP/1.1 403 Forbidden"
)

type propstatOut struct {
	Status string
	Body   string // serialised property elements
}

// responseOut is one DAV:response entry.
type responseOut struct {
	Href string
	// Status, when set, replaces the propstat list with a bare DAV:status —
	// the correct shape for a resource reported as removed or not found.
	Status    string
	Propstats []propstatOut
}

// notFoundResponse builds the response entry for a resource that is gone.
func notFoundResponse(href string) responseOut {
	return responseOut{Href: href, Status: statusNotFound}
}

// writeMultistatus serialises the 207 Multi-Status response. The trailer is
// appended inside the multistatus element, which is where DAV:sync-token goes.
func writeMultistatus(c *echo.Context, davHeader string, responses []responseOut, trailer string) error {
	var sb strings.Builder
	sb.WriteString(xml.Header)
	sb.WriteString(multistatusOpen)
	for _, r := range responses {
		sb.WriteString("<D:response>")
		sb.WriteString("<D:href>" + escapeXML(r.Href) + "</D:href>")
		if r.Status != "" {
			sb.WriteString("<D:status>" + r.Status + "</D:status>")
		}
		for _, ps := range r.Propstats {
			sb.WriteString("<D:propstat><D:prop>")
			sb.WriteString(ps.Body)
			sb.WriteString("</D:prop><D:status>" + ps.Status + "</D:status></D:propstat>")
		}
		sb.WriteString("</D:response>")
	}
	sb.WriteString(trailer)
	sb.WriteString("</D:multistatus>")

	c.Response().Header().Set("DAV", davHeader)
	return c.Blob(http.StatusMultiStatus, "application/xml; charset=utf-8", []byte(sb.String()))
}

// writeDAVError writes a bare DAV error document, used for 403
// DAV:valid-sync-token and similar precondition failures.
func writeDAVError(c *echo.Context, status int, inner string) error {
	body := xml.Header + `<D:error xmlns:D="DAV:">` + inner + `</D:error>`
	return c.Blob(status, "application/xml; charset=utf-8", []byte(body))
}

// ── Element helpers ───────────────────────────────────────────────────────

// element serialises a property with its inner content, declaring an inline
// namespace when the property is not one of the well-known ones.
func element(n xml.Name, inner string) string {
	prefix, ok := nsPrefixes[n.Space]
	if !ok {
		return `<x:` + n.Local + ` xmlns:x="` + escapeXML(n.Space) + `">` + inner + `</x:` + n.Local + `>`
	}
	return "<" + prefix + ":" + n.Local + ">" + inner + "</" + prefix + ":" + n.Local + ">"
}

// emptyElement serialises a property name with no content.
func emptyElement(n xml.Name) string {
	prefix, ok := nsPrefixes[n.Space]
	if !ok {
		return `<x:` + n.Local + ` xmlns:x="` + escapeXML(n.Space) + `"/>`
	}
	return "<" + prefix + ":" + n.Local + "/>"
}

// escapeXML escapes character data for inclusion in an XML document.
func escapeXML(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
