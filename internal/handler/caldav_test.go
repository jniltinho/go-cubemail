package handler

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"

	"go-cubemail/internal/dav"
	"go-cubemail/internal/model"
)

// sampleICS is a realistic calendar object: it carries a VTIMEZONE, a VALARM
// and an X-* extension, none of which the flat Event columns can represent.
// Any handler that rebuilds the blob from those columns fails these tests.
const sampleICS = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//Test//EN\r\n" +
	"BEGIN:VTIMEZONE\r\n" +
	"TZID:America/Sao_Paulo\r\n" +
	"BEGIN:STANDARD\r\n" +
	"DTSTART:19700101T000000\r\n" +
	"TZOFFSETFROM:-0300\r\n" +
	"TZOFFSETTO:-0300\r\n" +
	"TZNAME:-03\r\n" +
	"END:STANDARD\r\n" +
	"END:VTIMEZONE\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:event-001@test\r\n" +
	"SUMMARY:Planning meeting\r\n" +
	"DTSTART:20260710T130000Z\r\n" +
	"DTEND:20260710T140000Z\r\n" +
	"X-CUSTOM-FIELD:keep-me\r\n" +
	"BEGIN:VALARM\r\n" +
	"ACTION:DISPLAY\r\n" +
	"TRIGGER:-PT15M\r\n" +
	"DESCRIPTION:Reminder\r\n" +
	"END:VALARM\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// recurringICS is a weekly series, used to prove a calendar-query returns the
// master object once instead of one href per occurrence.
const recurringICS = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//Test//EN\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:weekly-001@test\r\n" +
	"SUMMARY:Weekly sync\r\n" +
	"DTSTART:20260706T120000Z\r\n" +
	"DTEND:20260706T130000Z\r\n" +
	"RRULE:FREQ=WEEKLY;COUNT=10\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func calendarPath(resource string) string {
	p := "/dav/" + davTestUser + "/calendars/default/"
	if resource != "" {
		p += resource
	}
	return p
}

func TestOptionsAdvertisesCalDAVCapabilities(t *testing.T) {
	env := newTestEnv(t)
	rec := env.do(http.MethodOptions, calendarPath(""), "", nil)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec.Header().Get("DAV"), "calendar-access", "DAV header")
	assertContains(t, rec.Header().Get("Allow"), "REPORT", "Allow header")
	assertContains(t, rec.Header().Get("Allow"), "MKCALENDAR", "Allow header")
}

func TestPropfindPrincipalAdvertisesHomeSets(t *testing.T) {
	env := newTestEnv(t)
	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CR="urn:ietf:params:xml:ns:carddav">
	  <D:prop>
	    <D:current-user-principal/>
	    <C:calendar-home-set/>
	    <CR:addressbook-home-set/>
	    <D:supported-report-set/>
	  </D:prop>
	</D:propfind>`

	rec := env.propfind("/dav/"+davTestUser+"/", "0", body)
	assertStatus(t, rec, http.StatusMultiStatus)

	doc := parseMultistatus(t, bodyBytes(rec))
	if len(doc.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(doc.Responses))
	}
	props := doc.Responses[0].okProps()
	assertContains(t, props, "/dav/"+davTestUser+"/calendars/", "calendar-home-set")
	assertContains(t, props, "/dav/"+davTestUser+"/contacts/", "addressbook-home-set")
	// Without sync-collection in supported-report-set, clients fall back to
	// listing the whole collection on every poll.
	assertContains(t, props, "sync-collection", "supported-report-set")
}

// A client asking for three properties must get exactly those three, with
// anything unknown reported as 404 rather than silently omitted.
func TestPropfindReturnsOnlyRequestedPropsAnd404ForUnknown(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:propfind xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/">
	  <D:prop>
	    <D:displayname/>
	    <CS:getctag/>
	    <D:this-property-does-not-exist/>
	  </D:prop>
	</D:propfind>`

	rec := env.propfind(calendarPath(""), "0", body)
	assertStatus(t, rec, http.StatusMultiStatus)

	doc := parseMultistatus(t, bodyBytes(rec))
	resp := doc.Responses[0]
	ok := resp.okProps()
	assertContains(t, ok, "displayname", "200 propstat")
	assertContains(t, ok, "getctag", "200 propstat")
	assertNotContains(t, ok, "this-property-does-not-exist", "200 propstat")
	assertContains(t, resp.notFoundProps(), "this-property-does-not-exist", "404 propstat")
	// Properties that were not asked for must not leak into the response.
	assertNotContains(t, ok, "calendar-color", "200 propstat")
}

// allprop covers live properties, not payloads. Returning calendar-data would
// put every .ics in the collection into one response.
func TestAllpropDoesNotInlineObjectBodies(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	env.do(http.MethodPut, calendarPath("big.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})

	doc := parseMultistatus(t, bodyBytes(env.propfind(calendarPath(""), "1", "")))
	resp, ok := doc.find("big.ics")
	if !ok {
		t.Fatal("object missing from the Depth: 1 listing")
	}
	props := resp.okProps()
	assertContains(t, props, "getetag", "allprop should still carry the ETag")
	assertNotContains(t, props, "BEGIN:VCALENDAR", "allprop must not inline calendar-data")
}

func TestHeadReturnsMetadataWithoutBody(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	env.do(http.MethodPut, calendarPath("h.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})

	rec := env.do(http.MethodHead, calendarPath("h.ics"), "", nil)
	assertStatus(t, rec, http.StatusOK)
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD must not return a body, got %d bytes", rec.Body.Len())
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("HEAD must still carry the ETag")
	}
}

func TestUnsupportedMethodReportsAllowHeader(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	rec := env.do(http.MethodPut, "/dav/"+davTestUser+"/calendars/", sampleICS, nil)
	assertStatus(t, rec, http.StatusMethodNotAllowed)
	assertContains(t, rec.Header().Get("Allow"), "PROPFIND", "Allow header")
}

func TestPropfindCalendarHomeHonoursDepth(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	home := "/dav/" + davTestUser + "/calendars/"

	deep := parseMultistatus(t, bodyBytes(env.propfind(home, "1", "")))
	if len(deep.Responses) != 2 {
		t.Fatalf("Depth: 1 should list the home plus 1 calendar, got %d responses", len(deep.Responses))
	}
	if _, ok := deep.find("/calendars/default/"); !ok {
		t.Fatalf("Depth: 1 response is missing the default calendar: %+v", deep.Responses)
	}

	shallow := parseMultistatus(t, bodyBytes(env.propfind(home, "0", "")))
	if len(shallow.Responses) != 1 {
		t.Fatalf("Depth: 0 should list only the home collection, got %d responses", len(shallow.Responses))
	}
}

// The blob a client PUTs must come back byte-for-byte. Losing VALARM,
// VTIMEZONE or X-* properties is the most common CalDAV interop failure.
func TestPutThenGetPreservesRawICalendar(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	put := env.do(http.MethodPut, calendarPath("meeting.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})
	assertStatus(t, put, http.StatusCreated)
	if put.Header().Get("ETag") == "" {
		t.Fatal("PUT must return an ETag so clients can do conditional updates")
	}

	get := env.do(http.MethodGet, calendarPath("meeting.ics"), "", nil)
	assertStatus(t, get, http.StatusOK)
	if got := get.Body.String(); got != sampleICS {
		t.Fatalf("stored calendar object was rewritten.\nwant:\n%s\ngot:\n%s", sampleICS, got)
	}
	if get.Header().Get("ETag") != put.Header().Get("ETag") {
		t.Fatalf("ETag changed between PUT (%s) and GET (%s)",
			put.Header().Get("ETag"), get.Header().Get("ETag"))
	}

	// The index columns must still be populated from the parsed blob.
	ev, err := env.h.CalDAV.eventRepo.GetByResourceURI(env.userID, env.defaultCalendar().ID, "meeting.ics")
	if err != nil {
		t.Fatalf("event not indexed: %v", err)
	}
	if ev.Summary != "Planning meeting" {
		t.Fatalf("summary index = %q, want %q", ev.Summary, "Planning meeting")
	}
	if ev.UID != "event-001@test" {
		t.Fatalf("uid index = %q", ev.UID)
	}
	// The resource name is what the client chose, not the UID.
	if ev.ResourceURI != "meeting.ics" {
		t.Fatalf("resource URI = %q, want meeting.ics", ev.ResourceURI)
	}
}

func TestOversizedCalendarObjectIsRejectedNotTruncated(t *testing.T) {
	env := newTestEnv(t)
	cal := env.defaultCalendar()

	huge := strings.Replace(sampleICS, "X-CUSTOM-FIELD:keep-me",
		"DESCRIPTION:"+strings.Repeat("x", maxResourceSize+1024), 1)

	rec := env.do(http.MethodPut, calendarPath("huge.ics"), huge,
		map[string]string{"Content-Type": "text/calendar"})
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertContains(t, rec.Body.String(), "max-resource-size", "precondition element")

	// A malformed error document is worse than none: clients log a parse error
	// and retry the same oversized PUT forever.
	var doc struct {
		XMLName xml.Name `xml:"DAV: error"`
		Inner   string   `xml:",innerxml"`
	}
	if err := xml.Unmarshal(bodyBytes(rec), &doc); err != nil {
		t.Fatalf("error body is not valid XML: %v\n%s", err, rec.Body.String())
	}

	if _, err := env.h.CalDAV.eventRepo.GetByResourceURI(env.userID, cal.ID, "huge.ics"); err == nil {
		t.Fatal("a rejected calendar object must not be stored")
	}
}

// An oversized XML document must be refused too: parsing a truncated PROPFIND
// silently falls back to allprop and answers a question nobody asked.
func TestOversizedPropfindBodyIsRejected(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	body := `<?xml version="1.0" encoding="utf-8"?><D:propfind xmlns:D="DAV:"><D:prop>` +
		strings.Repeat(`<D:displayname/>`, 8000) + `</D:prop></D:propfind>`

	assertStatus(t, env.propfind(calendarPath(""), "0", body), http.StatusRequestEntityTooLarge)
}

func TestPutConditionalRequests(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	// If-None-Match: * creates only when absent.
	created := env.do(http.MethodPut, calendarPath("a.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar", "If-None-Match": "*"})
	assertStatus(t, created, http.StatusCreated)
	etag := created.Header().Get("ETag")

	// A second create attempt must be refused, not silently overwrite.
	conflict := env.do(http.MethodPut, calendarPath("a.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar", "If-None-Match": "*"})
	assertStatus(t, conflict, http.StatusPreconditionFailed)

	// A stale If-Match is the two-clients-editing case: it must fail.
	stale := env.do(http.MethodPut, calendarPath("a.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar", "If-Match": `"stale-etag"`})
	assertStatus(t, stale, http.StatusPreconditionFailed)

	// The current ETag is accepted.
	updated := strings.Replace(sampleICS, "Planning meeting", "Planning meeting v2", 1)
	fresh := env.do(http.MethodPut, calendarPath("a.ics"), updated,
		map[string]string{"Content-Type": "text/calendar", "If-Match": etag})
	assertStatus(t, fresh, http.StatusNoContent)
	if fresh.Header().Get("ETag") == etag {
		t.Fatal("ETag must change when the resource content changes")
	}

	// DELETE honours If-Match too.
	badDelete := env.do(http.MethodDelete, calendarPath("a.ics"), "",
		map[string]string{"If-Match": etag}) // etag is now stale
	assertStatus(t, badDelete, http.StatusPreconditionFailed)

	goodDelete := env.do(http.MethodDelete, calendarPath("a.ics"), "",
		map[string]string{"If-Match": fresh.Header().Get("ETag")})
	assertStatus(t, goodDelete, http.StatusNoContent)
}

// The whole point of the changelog: a client must learn about deletions.
func TestSyncCollectionReportsCreatesUpdatesAndDeletions(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	const syncBody = `<?xml version="1.0" encoding="utf-8"?>
	<D:sync-collection xmlns:D="DAV:">
	  <D:sync-token>%s</D:sync-token>
	  <D:sync-level>1</D:sync-level>
	  <D:prop><D:getetag/></D:prop>
	</D:sync-collection>`

	env.do(http.MethodPut, calendarPath("one.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})
	env.do(http.MethodPut, calendarPath("two.ics"),
		strings.ReplaceAll(sampleICS, "event-001@test", "event-002@test"),
		map[string]string{"Content-Type": "text/calendar"})

	// Initial sync: empty token, everything is reported as present.
	first := env.report(calendarPath(""), strings.Replace(syncBody, "%s", "", 1))
	assertStatus(t, first, http.StatusMultiStatus)
	initial := parseMultistatus(t, bodyBytes(first))
	if len(initial.Responses) != 2 {
		t.Fatalf("initial sync should list 2 objects, got %d", len(initial.Responses))
	}
	if initial.SyncToken == "" {
		t.Fatal("initial sync must return a sync-token")
	}

	// Nothing changed since: the delta must be empty.
	quiet := parseMultistatus(t, bodyBytes(
		env.report(calendarPath(""), strings.Replace(syncBody, "%s", initial.SyncToken, 1))))
	if len(quiet.Responses) != 0 {
		t.Fatalf("delta with no changes should be empty, got %d responses", len(quiet.Responses))
	}

	// Delete one object and update the other.
	env.do(http.MethodDelete, calendarPath("one.ics"), "", nil)
	env.do(http.MethodPut, calendarPath("two.ics"),
		strings.ReplaceAll(strings.Replace(sampleICS, "Planning meeting", "Renamed", 1),
			"event-001@test", "event-002@test"),
		map[string]string{"Content-Type": "text/calendar"})

	delta := parseMultistatus(t, bodyBytes(
		env.report(calendarPath(""), strings.Replace(syncBody, "%s", quiet.SyncToken, 1))))
	if len(delta.Responses) != 2 {
		t.Fatalf("delta should report 1 deletion and 1 update, got %d", len(delta.Responses))
	}

	removed, ok := delta.find("one.ics")
	if !ok {
		t.Fatal("deleted resource is missing from the delta — clients would keep a ghost copy")
	}
	if !removed.isRemoved() {
		t.Fatalf("deleted resource should carry a 404 status, got %q", removed.Status)
	}

	changed, ok := delta.find("two.ics")
	if !ok {
		t.Fatal("updated resource missing from the delta")
	}
	assertContains(t, changed.okProps(), "getetag", "updated resource")
}

// A token the changelog can no longer serve must be rejected explicitly, so the
// client resyncs instead of silently missing changes.
func TestSyncCollectionRejectsPrunedToken(t *testing.T) {
	env := newTestEnv(t)
	cal := env.defaultCalendar()

	env.do(http.MethodPut, calendarPath("one.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})

	// Simulate a changelog whose early entries were pruned.
	if err := env.db.Model(&model.Calendar{}).Where("id = ?", cal.ID).
		Update("pruned_revision", 5).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Model(&model.Calendar{}).Where("id = ?", cal.ID).
		Update("sync_token", 10).Error; err != nil {
		t.Fatal(err)
	}

	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:sync-collection xmlns:D="DAV:">
	  <D:sync-token>` + dav.SyncToken(2) + `</D:sync-token>
	  <D:prop><D:getetag/></D:prop>
	</D:sync-collection>`

	rec := env.report(calendarPath(""), body)
	assertStatus(t, rec, http.StatusForbidden)
	assertContains(t, rec.Body.String(), "valid-sync-token", "error body")
}

func TestCalendarMultigetReturnsRequestedResources(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	env.do(http.MethodPut, calendarPath("m1.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})

	body := `<?xml version="1.0" encoding="utf-8"?>
	<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
	  <D:prop><D:getetag/><C:calendar-data/></D:prop>
	  <D:href>` + calendarPath("m1.ics") + `</D:href>
	  <D:href>` + calendarPath("missing.ics") + `</D:href>
	</C:calendar-multiget>`

	doc := parseMultistatus(t, bodyBytes(env.report(calendarPath(""), body)))
	if len(doc.Responses) != 2 {
		t.Fatalf("multiget should answer for every href, got %d", len(doc.Responses))
	}
	found, _ := doc.find("m1.ics")
	assertContains(t, found.okProps(), "X-CUSTOM-FIELD", "calendar-data")

	gone, ok := doc.find("missing.ics")
	if !ok || !gone.isRemoved() {
		t.Fatalf("unknown href should be reported as 404, got %+v", gone)
	}
}

// A recurring series is one resource. Expanding it server-side would repeat the
// same href once per occurrence and confuse every client.
func TestCalendarQueryReturnsRecurringMasterOnce(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	env.do(http.MethodPut, calendarPath("weekly.ics"), recurringICS,
		map[string]string{"Content-Type": "text/calendar"})

	body := `<?xml version="1.0" encoding="utf-8"?>
	<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
	  <D:prop><D:getetag/></D:prop>
	  <C:filter>
	    <C:comp-filter name="VCALENDAR">
	      <C:comp-filter name="VEVENT">
	        <C:time-range start="20260701T000000Z" end="20260930T000000Z"/>
	      </C:comp-filter>
	    </C:comp-filter>
	  </C:filter>
	</C:calendar-query>`

	doc := parseMultistatus(t, bodyBytes(env.report(calendarPath(""), body)))
	count := 0
	for _, r := range doc.Responses {
		if strings.HasSuffix(r.Href, "weekly.ics") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("recurring series returned %d times, want exactly 1", count)
	}
}

func TestCalendarQueryComponentFilterSeparatesTasksFromEvents(t *testing.T) {
	env := newTestEnv(t)
	cal := env.defaultCalendar()
	env.do(http.MethodPut, calendarPath("ev.ics"), sampleICS,
		map[string]string{"Content-Type": "text/calendar"})

	// A task lives in the same collection but must not match a VEVENT filter.
	task := model.Event{
		UserID: env.userID, CalendarID: cal.ID, UID: "task-1@test",
		Summary: "Write docs", IsTask: true, ResourceURI: "task.ics",
		ICalContent: "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:task-1@test\r\nEND:VTODO\r\nEND:VCALENDAR\r\n",
	}
	if err := env.h.CalDAV.eventRepo.Create(&task); err != nil {
		t.Fatal(err)
	}

	query := func(comp string) multistatusDoc {
		body := `<?xml version="1.0" encoding="utf-8"?>
		<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
		  <D:prop><D:getetag/></D:prop>
		  <C:filter><C:comp-filter name="VCALENDAR">
		    <C:comp-filter name="` + comp + `"/>
		  </C:comp-filter></C:filter>
		</C:calendar-query>`
		return parseMultistatus(t, bodyBytes(env.report(calendarPath(""), body)))
	}

	events := query("VEVENT")
	if _, ok := events.find("task.ics"); ok {
		t.Fatal("VEVENT filter returned a VTODO")
	}
	if _, ok := events.find("ev.ics"); !ok {
		t.Fatal("VEVENT filter dropped the event")
	}

	todos := query("VTODO")
	if _, ok := todos.find("ev.ics"); ok {
		t.Fatal("VTODO filter returned a VEVENT")
	}
	if _, ok := todos.find("task.ics"); !ok {
		t.Fatal("VTODO filter dropped the task")
	}
}

// Nothing may leak between users, and the response must not reveal that the
// other user's collection exists.
func TestAnotherUsersPathIsNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	for _, path := range []string{
		"/dav/someone-else@example.com/calendars/default/",
		"/dav/someone-else@example.com/contacts/default/",
		"/dav/someone-else@example.com/",
	} {
		rec := env.propfind(path, "1", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, rec.Code)
		}
	}
}

// Path traversal must not escape the collection.
func TestTraversalPathsAreRejected(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()

	rec := env.do(http.MethodGet, "/dav/"+davTestUser+"/calendars/default/%2e%2e%2fsecret.ics", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("traversal attempt status = %d, want 404", rec.Code)
	}
}

func TestMkcalendarCreatesCollectionAndProppatchRenamesIt(t *testing.T) {
	env := newTestEnv(t)
	env.defaultCalendar()
	path := "/dav/" + davTestUser + "/calendars/work/"

	body := `<?xml version="1.0" encoding="utf-8"?>
	<C:mkcalendar xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"
	              xmlns:ICAL="http://apple.com/ns/ical/">
	  <D:set><D:prop>
	    <D:displayname>Work</D:displayname>
	    <ICAL:calendar-color>#ff0000</ICAL:calendar-color>
	  </D:prop></D:set>
	</C:mkcalendar>`

	assertStatus(t, env.do("MKCALENDAR", path, body, map[string]string{"Content-Type": "application/xml"}),
		http.StatusCreated)

	cal, err := env.h.CalDAV.calRepo.GetByURI(env.userID, "work")
	if err != nil {
		t.Fatalf("calendar was not created: %v", err)
	}
	if cal.Name != "Work" || cal.Color != "#ff0000" {
		t.Fatalf("MKCALENDAR properties ignored: name=%q color=%q", cal.Name, cal.Color)
	}

	patch := `<?xml version="1.0" encoding="utf-8"?>
	<D:propertyupdate xmlns:D="DAV:">
	  <D:set><D:prop><D:displayname>Work Renamed</D:displayname></D:prop></D:set>
	</D:propertyupdate>`
	assertStatus(t, env.do("PROPPATCH", path, patch, map[string]string{"Content-Type": "application/xml"}),
		http.StatusMultiStatus)

	cal, _ = env.h.CalDAV.calRepo.GetByURI(env.userID, "work")
	if cal.Name != "Work Renamed" {
		t.Fatalf("PROPPATCH displayname = %q", cal.Name)
	}
	// Renaming must not move the collection: client sync state is keyed on the URI.
	if cal.URI != "work" {
		t.Fatalf("collection URI changed on rename: %q", cal.URI)
	}
}

// Events created through the web API must reach DAV clients, otherwise the two
// interfaces drift apart.
func TestRestCreatedEventAppearsInSyncCollection(t *testing.T) {
	env := newTestEnv(t)
	cal := env.defaultCalendar()

	ev := model.Event{
		UserID: env.userID, CalendarID: cal.ID,
		UID: "from-webui@test", Summary: "Created in the web UI",
	}
	if err := env.h.CalDAV.eventRepo.Create(&ev); err != nil {
		t.Fatal(err)
	}
	if ev.ResourceURI == "" || ev.ETag == "" {
		t.Fatalf("repository must assign a resource name and ETag, got %q / %q",
			ev.ResourceURI, ev.ETag)
	}

	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:sync-collection xmlns:D="DAV:">
	  <D:sync-token></D:sync-token>
	  <D:prop><D:getetag/></D:prop>
	</D:sync-collection>`
	doc := parseMultistatus(t, bodyBytes(env.report(calendarPath(""), body)))
	if _, ok := doc.find(ev.ResourceURI); !ok {
		t.Fatal("event created through the repository is invisible to CalDAV clients")
	}
}
