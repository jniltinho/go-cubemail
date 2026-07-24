// Package handler — CalDAV server (RFC 4791) for Apple Calendar, Thunderbird,
// Evolution, DAVx⁵ and anything else speaking the standard.
//
// Discovery chain used by clients:
//   1. GET|PROPFIND /.well-known/caldav             → 301 /dav/{user}/
//   2. PROPFIND /dav/{user}/                        → current-user-principal, calendar-home-set
//   3. PROPFIND /dav/{user}/calendars/   Depth:1    → calendar list (resourcetype, ctag, sync-token)
//   4. REPORT   sync-collection on each calendar    → delta since the client's token
//   5. GET|PUT|DELETE individual .ics resources     → conditional via ETag
//
// Resource identity is (calendar, resource name), never the iCalendar UID: a
// client is free to name a resource anything, and a recurring series with
// overrides is one resource holding several VEVENTs that share a UID.
//
// Auth: HTTP Basic validated against IMAP by the CalDAVAuth middleware, which
// sets "caldav_user_id" and "caldav_username" on the context.
package handler

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/config"
	"go-cubemail/internal/dav"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalDAVHandler serves the CalDAV portion of the /dav namespace.
type CalDAVHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	eventRepo *repository.EventRepo
	sync      *dav.Store
}

// davCapabilities is the DAV compliance header advertised to clients.
const davCapabilities = "1, 2, 3, access-control, calendar-access, addressbook, extended-mkcol"

// maxResourceSize bounds a single calendar or address object.
const maxResourceSize = 10 * 1024 * 1024

// ── Auth context helpers ──────────────────────────────────────────────────

func caldavUserID(c *echo.Context) (uint, bool) {
	v := c.Get("caldav_user_id")
	if v == nil {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

func caldavUsername(c *echo.Context) string {
	v := c.Get("caldav_username")
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// depthOf returns the Depth header, defaulting to 0.
//
// RFC 4918 defaults Depth to infinity, but for the collections served here
// infinity and 1 are equivalent (there is no nesting), and defaulting to 0
// would break clients that omit the header when listing. Treat anything that
// is not "0" as "list the children".
func depthOf(c *echo.Context) string {
	d := strings.TrimSpace(c.Request().Header.Get("Depth"))
	if d == "0" {
		return "0"
	}
	return "1"
}

// ── OPTIONS / well-known ──────────────────────────────────────────────────

// Options advertises the DAV capabilities and allowed methods.
func (h *CalDAVHandler) Options(c *echo.Context) error {
	c.Response().Header().Set("DAV", davCapabilities)
	c.Response().Header().Set("Allow",
		"OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, REPORT, MKCALENDAR, MKCOL")
	c.Response().Header().Set("Content-Length", "0")
	return c.NoContent(http.StatusOK)
}

// WellKnown handles GET|PROPFIND /.well-known/caldav → redirect to the principal.
func (h *CalDAVHandler) WellKnown(c *echo.Context) error {
	user := caldavUsername(c)
	if user == "" {
		return c.NoContent(http.StatusUnauthorized)
	}
	c.Response().Header().Set("Location", principalHref(user))
	return c.NoContent(http.StatusMovedPermanently)
}

// ── PROPFIND ──────────────────────────────────────────────────────────────

// propfindPrincipal answers PROPFIND on /dav/ and /dav/{user}/.
func (h *CalDAVHandler) propfindPrincipal(c *echo.Context, username string, req propfindRequest) error {
	href := principalHref(username)
	bag := newPropBag()
	bag.setRaw(nsDAV, "resourcetype", "<D:collection/><D:principal/>")
	bag.setText(nsDAV, "displayname", username)
	bag.setRaw(nsDAV, "principal-URL", hrefElement(href))
	bag.setRaw(nsDAV, "current-user-principal", hrefElement(href))
	bag.setRaw(nsDAV, "principal-collection-set", hrefElement("/dav/"))
	bag.setRaw(nsCalDAV, "calendar-home-set", hrefElement(calendarHomeHref(username)))
	bag.setRaw(nsCardDAV, "addressbook-home-set", hrefElement(addressBookHomeHref(username)))
	bag.setRaw(nsCalDAV, "calendar-user-address-set", hrefElement("mailto:"+username))
	bag.setRaw(nsDAV, "supported-report-set", supportedReportSet(true, true))
	bag.setRaw(nsDAV, "current-user-privilege-set", privilegeSet())

	return writeMultistatus(c, davCapabilities, []responseOut{
		{Href: href, Propstats: bag.render(req)},
	}, "")
}

// propfindCalendarHome answers PROPFIND on the calendar-home-set.
func (h *CalDAVHandler) propfindCalendarHome(c *echo.Context, userID uint, username string, req propfindRequest, depth string) error {
	if _, err := h.calRepo.EnsureDefault(userID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := h.calRepo.BackfillURIs(userID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	home := newPropBag()
	home.setRaw(nsDAV, "resourcetype", "<D:collection/>")
	home.setText(nsDAV, "displayname", "Calendars")
	home.setRaw(nsDAV, "current-user-principal", hrefElement(principalHref(username)))
	home.setRaw(nsDAV, "owner", hrefElement(principalHref(username)))

	responses := []responseOut{{Href: calendarHomeHref(username), Propstats: home.render(req)}}

	if depth != "0" {
		cals, err := h.calRepo.List(userID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		for i := range cals {
			responses = append(responses, responseOut{
				Href:      calendarHref(username, cals[i].URI),
				Propstats: h.calendarPropBag(username, &cals[i]).render(req),
			})
		}
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// propfindCalendar answers PROPFIND on one calendar collection.
func (h *CalDAVHandler) propfindCalendar(c *echo.Context, userID uint, username string, cal *model.Calendar, req propfindRequest, depth string) error {
	responses := []responseOut{{
		Href:      calendarHref(username, cal.URI),
		Propstats: h.calendarPropBag(username, cal).render(req),
	}}

	if depth != "0" {
		events, err := h.eventRepo.ListByCalendar(userID, cal.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		wantData := req.wantsExplicit(nsCalDAV, "calendar-data")
		for i := range events {
			responses = append(responses, responseOut{
				Href:      calendarObjectHref(username, cal.URI, events[i].ResourceURI),
				Propstats: eventPropBag(&events[i], wantData).render(req),
			})
		}
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// propfindCalendarObject answers PROPFIND on a single .ics resource.
func (h *CalDAVHandler) propfindCalendarObject(c *echo.Context, username string, cal *model.Calendar, ev *model.Event, req propfindRequest) error {
	return writeMultistatus(c, davCapabilities, []responseOut{{
		Href:      calendarObjectHref(username, cal.URI, ev.ResourceURI),
		Propstats: eventPropBag(ev, req.wantsExplicit(nsCalDAV, "calendar-data")).render(req),
	}}, "")
}

// calendarPropBag collects the properties of a calendar collection.
func (h *CalDAVHandler) calendarPropBag(username string, cal *model.Calendar) *propBag {
	bag := newPropBag()
	bag.setRaw(nsDAV, "resourcetype", "<D:collection/><C:calendar/>")
	bag.setText(nsDAV, "displayname", cal.Name)
	bag.setText(nsCS, "getctag", dav.CTag(cal.SyncToken))
	bag.setText(nsDAV, "sync-token", dav.SyncToken(cal.SyncToken))
	bag.setRaw(nsCalDAV, "supported-calendar-component-set",
		`<C:comp name="VEVENT"/><C:comp name="VTODO"/>`)
	bag.setRaw(nsCalDAV, "supported-calendar-data",
		`<C:calendar-data content-type="text/calendar" version="2.0"/>`)
	bag.setText(nsCalDAV, "calendar-description", cal.Description)
	if cal.TimeZone != "" {
		bag.setText(nsCalDAV, "calendar-timezone", cal.TimeZone)
	}
	bag.setText(nsCalDAV, "max-resource-size", strconv.Itoa(maxResourceSize))
	bag.setText(nsApple, "calendar-color", cal.Color)
	bag.setText(nsApple, "calendar-order", strconv.Itoa(cal.SortOrder))
	bag.setRaw(nsDAV, "current-user-principal", hrefElement(principalHref(username)))
	bag.setRaw(nsDAV, "owner", hrefElement(principalHref(username)))
	bag.setRaw(nsDAV, "supported-report-set", supportedReportSet(true, false))
	bag.setRaw(nsDAV, "current-user-privilege-set", privilegeSet())
	return bag
}

// eventPropBag collects the properties of a calendar object resource.
func eventPropBag(ev *model.Event, withData bool) *propBag {
	ics := eventICS(ev)
	bag := newPropBag()
	bag.setText(nsDAV, "getetag", dav.Quote(eventETag(ev)))
	bag.setText(nsDAV, "getcontenttype", "text/calendar; charset=utf-8; component=vevent")
	bag.setText(nsDAV, "getcontentlength", strconv.Itoa(len(ics)))
	bag.setText(nsDAV, "getlastmodified", ev.UpdatedAt.UTC().Format(http.TimeFormat))
	bag.setRaw(nsDAV, "resourcetype", "")
	if withData {
		bag.setText(nsCalDAV, "calendar-data", ics)
	}
	return bag
}

// ── REPORT ────────────────────────────────────────────────────────────────

// Report dispatches a CalDAV REPORT on a calendar collection.
func (h *CalDAVHandler) report(c *echo.Context, userID uint, username string, cal *model.Calendar) error {
	body, tooLarge := readBody(c, davReportBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	name, err := reportName(body)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	switch {
	case name.Space == nsDAV && name.Local == "sync-collection":
		return h.reportSyncCollection(c, userID, username, cal, body)
	case name.Space == nsCalDAV && name.Local == "calendar-multiget":
		return h.reportMultiget(c, userID, username, cal, body)
	case name.Space == nsCalDAV && name.Local == "calendar-query":
		return h.reportCalendarQuery(c, userID, username, cal, body)
	default:
		return c.NoContent(http.StatusBadRequest)
	}
}

// reportSyncCollection answers the RFC 6578 delta report: what changed since
// the client's token, including tombstones for deleted resources.
func (h *CalDAVHandler) reportSyncCollection(c *echo.Context, userID uint, username string, cal *model.Calendar, body []byte) error {
	var req syncCollectionRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}

	since, ok := dav.ParseSyncToken(req.SyncToken)
	if !ok {
		return invalidSyncToken(c)
	}

	var responses []responseOut
	var newToken uint64

	if since == 0 {
		// Initial sync: enumerate the collection rather than replaying the
		// changelog, which may have been pruned or predate these rows.
		events, err := h.eventRepo.ListByCalendar(userID, cal.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		newToken, err = h.sync.CurrentRevision(model.CollectionCalendar, cal.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		if over, resp := limitExceeded(req.Limit, len(events)); over {
			return resp(c)
		}
		wantData := propReq.wantsExplicit(nsCalDAV, "calendar-data")
		for i := range events {
			responses = append(responses, responseOut{
				Href:      calendarObjectHref(username, cal.URI, events[i].ResourceURI),
				Propstats: eventPropBag(&events[i], wantData).render(propReq),
			})
		}
	} else {
		changes, current, err := h.sync.ChangesSince(model.CollectionCalendar, cal.ID, since)
		if errors.Is(err, dav.ErrInvalidSyncToken) {
			return invalidSyncToken(c)
		}
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		newToken = current
		if over, resp := limitExceeded(req.Limit, len(changes)); over {
			return resp(c)
		}
		wantData := propReq.wantsExplicit(nsCalDAV, "calendar-data")
		for _, ch := range changes {
			href := calendarObjectHref(username, cal.URI, ch.URI)
			if ch.Deleted {
				responses = append(responses, notFoundResponse(href))
				continue
			}
			ev, err := h.eventRepo.GetByResourceURI(userID, cal.ID, ch.URI)
			if err != nil {
				responses = append(responses, notFoundResponse(href))
				continue
			}
			responses = append(responses, responseOut{
				Href:      href,
				Propstats: eventPropBag(ev, wantData).render(propReq),
			})
		}
	}

	trailer := "<D:sync-token>" + escapeXML(dav.SyncToken(newToken)) + "</D:sync-token>"
	return writeMultistatus(c, davCapabilities, responses, trailer)
}

// reportMultiget returns the resources named by the client's href list.
func (h *CalDAVHandler) reportMultiget(c *echo.Context, userID uint, username string, cal *model.Calendar, body []byte) error {
	var req multigetRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}
	wantData := propReq.wantsExplicit(nsCalDAV, "calendar-data")

	responses := make([]responseOut, 0, len(req.Hrefs))
	for _, href := range req.Hrefs {
		uri := dav.ResourceURIFromHref(href)
		if uri == "" {
			responses = append(responses, notFoundResponse(href))
			continue
		}
		ev, err := h.eventRepo.GetByResourceURI(userID, cal.ID, uri)
		if err != nil {
			responses = append(responses, notFoundResponse(href))
			continue
		}
		responses = append(responses, responseOut{
			Href:      calendarObjectHref(username, cal.URI, ev.ResourceURI),
			Propstats: eventPropBag(ev, wantData).render(propReq),
		})
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// reportCalendarQuery answers a filtered listing of the collection.
func (h *CalDAVHandler) reportCalendarQuery(c *echo.Context, userID uint, username string, cal *model.Calendar, body []byte) error {
	var req calendarQueryRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}
	wantData := propReq.wantsExplicit(nsCalDAV, "calendar-data")

	comp, start, end := calendarFilter(req.Filter.Comp)

	var events []model.Event
	var err error
	if !start.IsZero() && !end.IsZero() {
		events, err = h.eventRepo.ListByCalendarRange(userID, cal.ID, start, end)
	} else {
		events, err = h.eventRepo.ListByCalendar(userID, cal.ID)
	}
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	responses := make([]responseOut, 0, len(events))
	for i := range events {
		if !componentMatches(comp, &events[i]) {
			continue
		}
		responses = append(responses, responseOut{
			Href:      calendarObjectHref(username, cal.URI, events[i].ResourceURI),
			Propstats: eventPropBag(&events[i], wantData).render(propReq),
		})
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// calendarFilter walks the comp-filter tree, returning the requested component
// name and the time-range bounds if any.
func calendarFilter(root compFilterElem) (comp string, start, end time.Time) {
	// The outer filter is VCALENDAR; the component of interest is one level in.
	for _, child := range root.Comps {
		comp = strings.ToUpper(child.Name)
		if child.TimeRange != nil {
			start = parseICalUTC(child.TimeRange.Start)
			end = parseICalUTC(child.TimeRange.End)
		}
		for _, grand := range child.Comps {
			if grand.TimeRange != nil {
				start = parseICalUTC(grand.TimeRange.Start)
				end = parseICalUTC(grand.TimeRange.End)
			}
		}
		break
	}
	if root.TimeRange != nil && start.IsZero() {
		start = parseICalUTC(root.TimeRange.Start)
		end = parseICalUTC(root.TimeRange.End)
	}
	return comp, start, end
}

// componentMatches reports whether an event satisfies a comp-filter name.
func componentMatches(comp string, ev *model.Event) bool {
	switch comp {
	case "", "VCALENDAR":
		return true
	case "VTODO":
		return ev.IsTask
	case "VEVENT":
		return !ev.IsTask
	default:
		return false
	}
}

// parseICalUTC parses the iCalendar timestamp forms used in time-range filters.
func parseICalUTC(v string) time.Time {
	for _, layout := range []string{"20060102T150405Z", "20060102T150405", "20060102"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// ── GET / PUT / DELETE on calendar objects ────────────────────────────────

// getCalendarObject writes an .ics resource.
func (h *CalDAVHandler) getCalendarObject(c *echo.Context, ev *model.Event, body bool) error {
	ics := eventICS(ev)
	c.Response().Header().Set("ETag", dav.Quote(eventETag(ev)))
	c.Response().Header().Set("Last-Modified", ev.UpdatedAt.UTC().Format(http.TimeFormat))
	if !body {
		c.Response().Header().Set("Content-Length", strconv.Itoa(len(ics)))
		return c.NoContent(http.StatusOK)
	}
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

// getCalendar exports the whole collection as a single .ics document.
func (h *CalDAVHandler) getCalendar(c *echo.Context, userID uint, cal *model.Calendar) error {
	events, err := h.eventRepo.ListByCalendar(userID, cal.ID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8",
		[]byte(calpkg.BuildCalendarExport(events)))
}

// putCalendarObject creates or replaces an .ics resource.
//
// The request body is stored verbatim: the parsed fields only feed the index
// columns. Re-serialising the blob would strip VALARM, VTIMEZONE, X-* and the
// RECURRENCE-ID overrides that share the master's UID inside one resource.
func (h *CalDAVHandler) putCalendarObject(c *echo.Context, userID uint, cal *model.Calendar, resource string) error {
	if ct := c.Request().Header.Get("Content-Type"); ct != "" &&
		!strings.HasPrefix(strings.ToLower(ct), "text/calendar") {
		return c.NoContent(http.StatusUnsupportedMediaType)
	}

	body, tooLarge := readBody(c, maxResourceSize)
	if tooLarge {
		return writeTooLarge(c, nsCalDAV)
	}
	if len(body) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	existing, err := h.eventRepo.GetByResourceURI(userID, cal.ID, resource)
	exists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return c.NoContent(http.StatusInternalServerError)
	}

	currentETag := ""
	if exists {
		currentETag = eventETag(existing)
	}
	if err := dav.CheckPreconditions(c.Request().Header, exists, currentETag); err != nil {
		return c.NoContent(http.StatusPreconditionFailed)
	}

	parsed, err := calpkg.ParseICalImport(body)
	if err != nil || len(parsed) == 0 {
		return writeDAVError(c, http.StatusForbidden,
			`<C:valid-calendar-data xmlns:C="urn:ietf:params:xml:ns:caldav"/>`)
	}
	item := parsed[0]

	if !exists {
		ev := model.Event{
			CalendarID:  cal.ID,
			UserID:      userID,
			UID:         item.UID,
			ResourceURI: resource,
			ICalContent: string(body),
			Attendees:   item.Attendees,
		}
		applyImportedFields(&ev, item)
		if err := h.eventRepo.Create(&ev); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		c.Response().Header().Set("ETag", dav.Quote(ev.ETag))
		return c.NoContent(http.StatusCreated)
	}

	// A client may not repoint an existing resource at a different UID.
	if item.UID != "" && existing.UID != "" && item.UID != existing.UID {
		return writeDAVError(c, http.StatusForbidden,
			`<C:no-uid-conflict xmlns:C="urn:ietf:params:xml:ns:caldav"/>`)
	}
	existing.ICalContent = string(body)
	existing.Attendees = item.Attendees
	existing.Sequence++
	applyImportedFields(existing, item)
	if err := h.eventRepo.Update(existing); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	c.Response().Header().Set("ETag", dav.Quote(existing.ETag))
	return c.NoContent(http.StatusNoContent)
}

// applyImportedFields copies the parsed iCalendar values onto the index columns.
func applyImportedFields(ev *model.Event, item calpkg.ImportEvent) {
	if item.UID != "" {
		ev.UID = item.UID
	}
	ev.Summary = item.Summary
	ev.Description = item.Description
	ev.Location = item.Location
	ev.StartAt = item.StartAt
	ev.EndAt = item.EndAt
	ev.IsAllDay = item.IsAllDay
	ev.RRule = item.RRule
	ev.Status = item.Status
	if ev.Status == "" {
		ev.Status = "CONFIRMED"
	}
}

// deleteCalendarObject removes an .ics resource, honouring If-Match.
func (h *CalDAVHandler) deleteCalendarObject(c *echo.Context, userID uint, cal *model.Calendar, resource string) error {
	ev, err := h.eventRepo.GetByResourceURI(userID, cal.ID, resource)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	if err := dav.CheckPreconditions(c.Request().Header, true, eventETag(ev)); err != nil {
		return c.NoContent(http.StatusPreconditionFailed)
	}
	if err := h.eventRepo.Delete(userID, ev.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Collection management ─────────────────────────────────────────────────

// mkCalendar handles MKCALENDAR (and extended MKCOL) for a new collection.
func (h *CalDAVHandler) mkCalendar(c *echo.Context, userID uint, uri string) error {
	if _, err := h.calRepo.GetByURI(userID, uri); err == nil {
		return c.NoContent(http.StatusMethodNotAllowed) // already exists
	}
	body, tooLarge := readBody(c, davRequestBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	patch := parseProppatch(body)

	cal := model.Calendar{
		UserID:    userID,
		URI:       uri,
		Name:      uri,
		Color:     "#3788d8",
		IsActive:  true,
		SyncToken: 1,
	}
	for _, p := range patch.Set {
		applyCalendarProperty(&cal, p)
	}
	if err := h.calRepo.Create(&cal); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusCreated)
}

// propPatchCalendar applies a PROPPATCH to a calendar collection.
func (h *CalDAVHandler) propPatchCalendar(c *echo.Context, username string, cal *model.Calendar) error {
	body, tooLarge := readBody(c, davRequestBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	patch := parseProppatch(body)

	var okProps, failProps []string
	for _, p := range patch.Set {
		if applyCalendarProperty(cal, p) {
			okProps = append(okProps, emptyElement(p.Name))
		} else {
			failProps = append(failProps, emptyElement(p.Name))
		}
	}
	for _, n := range patch.Remove {
		// Protected live properties cannot be removed.
		failProps = append(failProps, emptyElement(n))
	}
	if len(okProps) > 0 {
		if err := h.calRepo.Update(cal); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	var stats []propstatOut
	if len(okProps) > 0 {
		stats = append(stats, propstatOut{Status: statusOK, Body: strings.Join(okProps, "")})
	}
	if len(failProps) > 0 {
		stats = append(stats, propstatOut{Status: statusForbidden, Body: strings.Join(failProps, "")})
	}
	return writeMultistatus(c, davCapabilities, []responseOut{
		{Href: calendarHref(username, cal.URI), Propstats: stats},
	}, "")
}

// applyCalendarProperty maps a settable DAV property onto the calendar row.
func applyCalendarProperty(cal *model.Calendar, p proppatchProp) bool {
	switch {
	case p.Name.Space == nsDAV && p.Name.Local == "displayname":
		cal.Name = p.Value
	case p.Name.Space == nsApple && p.Name.Local == "calendar-color":
		cal.Color = normaliseColor(p.Value)
	case p.Name.Space == nsApple && p.Name.Local == "calendar-order":
		if n, err := strconv.Atoi(strings.TrimSpace(p.Value)); err == nil {
			cal.SortOrder = n
		}
	case p.Name.Space == nsCalDAV && p.Name.Local == "calendar-description":
		cal.Description = p.Value
	case p.Name.Space == nsCalDAV && p.Name.Local == "calendar-timezone":
		cal.TimeZone = p.Value
	default:
		return false
	}
	return true
}

// normaliseColor trims Apple's optional alpha suffix to a plain #RRGGBB value.
func normaliseColor(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 9 {
		v = v[:9]
	}
	return v
}

// deleteCalendar removes a whole calendar collection.
func (h *CalDAVHandler) deleteCalendar(c *echo.Context, userID uint, cal *model.Calendar) error {
	if cal.IsDefault {
		return c.NoContent(http.StatusForbidden)
	}
	if err := h.calRepo.Delete(userID, cal.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Shared helpers ────────────────────────────────────────────────────────

// eventICS returns the stored blob, falling back to a generated one for rows
// created before blobs were kept (or by the REST API).
func eventICS(ev *model.Event) string {
	if ev.ICalContent != "" {
		return ev.ICalContent
	}
	return calpkg.BuildICalContent(ev, ev.Attendees)
}

// eventETag returns the stored entity tag, deriving one on the fly when the row
// predates ETag persistence.
func eventETag(ev *model.Event) string {
	if ev.ETag != "" {
		return ev.ETag
	}
	return dav.ComputeETag([]byte(eventICS(ev)))
}

// namesOf extracts the property names listed in a DAV:prop element.
func namesOf(p propElement) []xml.Name {
	out := make([]xml.Name, 0, len(p.Props))
	for _, e := range p.Props {
		out = append(out, e.XMLName)
	}
	return out
}

// hrefElement wraps a path in a DAV:href element.
func hrefElement(href string) string {
	return "<D:href>" + escapeXML(href) + "</D:href>"
}

// supportedReportSet advertises the REPORTs a resource accepts. Clients use it
// to decide whether they can rely on sync-collection instead of full listings.
func supportedReportSet(calendar, principal bool) string {
	reports := []string{"<D:sync-collection/>", "<D:expand-property/>"}
	if calendar {
		reports = append(reports,
			`<C:calendar-query/>`,
			`<C:calendar-multiget/>`,
			`<C:free-busy-query/>`,
			`<CR:addressbook-query/>`,
			`<CR:addressbook-multiget/>`)
	}
	if principal {
		reports = append(reports, "<D:principal-property-search/>")
	}
	var sb strings.Builder
	for _, r := range reports {
		sb.WriteString("<D:supported-report><D:report>" + r + "</D:report></D:supported-report>")
	}
	return sb.String()
}

// privilegeSet reports the privileges the authenticated user holds. Access is
// all-or-nothing here: a user either owns the collection or cannot see it.
func privilegeSet() string {
	return "<D:privilege><D:all/></D:privilege>" +
		"<D:privilege><D:read/></D:privilege>" +
		"<D:privilege><D:write/></D:privilege>" +
		"<D:privilege><D:write-content/></D:privilege>" +
		"<D:privilege><D:write-properties/></D:privilege>"
}

// invalidSyncToken tells the client its token is too old to serve a delta from,
// which is the signal to discard local state and start a full sync.
func invalidSyncToken(c *echo.Context) error {
	return writeDAVError(c, http.StatusForbidden, "<D:valid-sync-token/>")
}

// limitExceeded implements the DAV:limit element of sync-collection: when more
// results match than the client allowed, the request fails rather than
// returning a truncated set the client would mistake for the whole delta.
func limitExceeded(limit *limitElem, count int) (bool, func(*echo.Context) error) {
	if limit == nil || limit.NResults <= 0 || count <= limit.NResults {
		return false, nil
	}
	return true, func(c *echo.Context) error {
		return writeDAVError(c, http.StatusInsufficientStorage,
			fmt.Sprintf("<D:number-of-matches-within-limits>%d</D:number-of-matches-within-limits>", count))
	}
}
