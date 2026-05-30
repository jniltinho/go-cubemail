// Package handler — CalDAV server (RFC 4791) for Apple Calendar, Thunderbird, Evolution.
//
// Discovery chain used by clients:
//   1. GET /.well-known/caldav                       → 301 /dav/{user}/
//   2. PROPFIND /dav/{user}/                         → current-user-principal, calendar-home-set
//   3. PROPFIND /dav/{user}/calendars/   Depth:1     → calendar list (resourcetype, displayname, ctag)
//   4. PROPFIND /dav/{user}/calendars/{c}/ Depth:1   → event list (getetag, getcontenttype)
//   5. REPORT   /dav/{user}/calendars/{c}/           → calendar-query / calendar-multiget
//   6. GET|PUT|DELETE individual .ics resources
//
// Auth: HTTP Basic Auth validated against IMAP.  The CalDAVAuth middleware
// creates/resolves the model.User record and sets "caldav_user_id" on the context.
package handler

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalDAVHandler handles all CalDAV HTTP methods.
type CalDAVHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	eventRepo *repository.EventRepo
}

// ── Auth context helper ───────────────────────────────────────────────────

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

// ── Namespace constants ───────────────────────────────────────────────────

const (
	nsDAV    = "DAV:"
	nsCalDAV = "urn:ietf:params:xml:ns:caldav"
	nsCS     = "http://calendarserver.org/ns/"
)

// ── OPTIONS ───────────────────────────────────────────────────────────────

// Options handles OPTIONS /* — returns DAV capability headers.
func (h *CalDAVHandler) Options(c *echo.Context) error {
	c.Response().Header().Set("DAV", "1, 2, calendar-access")
	c.Response().Header().Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, REPORT, MKCALENDAR")
	c.Response().Header().Set("Content-Length", "0")
	return c.NoContent(http.StatusOK)
}

// ── WellKnown ─────────────────────────────────────────────────────────────

// WellKnown handles GET|PROPFIND /.well-known/caldav → redirect to principal URL.
func (h *CalDAVHandler) WellKnown(c *echo.Context) error {
	user := caldavUsername(c)
	if user == "" {
		return c.NoContent(http.StatusUnauthorized)
	}
	target := fmt.Sprintf("%s/dav/%s/", strings.TrimRight(h.cfg.Server.BaseURL, "/"), user)
	c.Response().Header().Set("Location", target)
	return c.NoContent(http.StatusMovedPermanently)
}

// ── PROPFIND dispatcher ───────────────────────────────────────────────────

// PropFind dispatches PROPFIND by URL path depth.
func (h *CalDAVHandler) PropFind(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	username := caldavUsername(c)

	path := c.Request().URL.Path
	depth := c.Request().Header.Get("Depth")
	if depth == "" {
		depth = "1"
	}

	// Determine which resource the PROPFIND targets.
	calSlug := c.Param("cal")
	uidParam := strings.TrimSuffix(c.Param("uid"), ".ics")

	switch {
	case uidParam != "":
		return h.propfindEvent(c, userID, username, calSlug, uidParam)
	case calSlug != "":
		return h.propfindCalendar(c, userID, username, calSlug, depth)
	case strings.Contains(path, "/calendars"):
		return h.propfindCalendarHome(c, userID, username, depth)
	default:
		return h.propfindPrincipal(c, userID, username)
	}
}

// ── PROPFIND /dav/{user}/ — principal ────────────────────────────────────

func (h *CalDAVHandler) propfindPrincipal(c *echo.Context, userID uint, username string) error {
	base := davBase(h.cfg)
	principalURL := fmt.Sprintf("%s/dav/%s/", base, username)
	homeURL := fmt.Sprintf("%s/dav/%s/calendars/", base, username)

	resp := multistatus([]propResponse{
		{
			Href: principalURL,
			Props: []propstat{
				{
					Status: "HTTP/1.1 200 OK",
					Props: []xml.TokenReader{
						xmlProp("D:resourcetype", `<D:collection/><D:principal/>`),
						xmlProp("D:displayname", username),
						xmlProp("D:principal-URL", `<D:href>`+principalURL+`</D:href>`),
						xmlProp("D:current-user-principal", `<D:href>`+principalURL+`</D:href>`),
						xmlProp("C:calendar-home-set", `<D:href>`+homeURL+`</D:href>`),
					},
				},
			},
		},
	})
	return writePropfind(c, resp)
}

// ── PROPFIND /dav/{user}/calendars/ — calendar home ──────────────────────

func (h *CalDAVHandler) propfindCalendarHome(c *echo.Context, userID uint, username, depth string) error {
	base := davBase(h.cfg)
	homeURL := fmt.Sprintf("%s/dav/%s/calendars/", base, username)

	if _, err := h.calRepo.EnsureDefault(userID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	responses := []propResponse{
		{
			Href: homeURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					xmlProp("D:resourcetype", `<D:collection/>`),
					xmlProp("D:displayname", "Calendars"),
				},
			}},
		},
	}

	if depth == "1" {
		cals, err := h.calRepo.List(userID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		for _, cal := range cals {
			calURL := fmt.Sprintf("%s/dav/%s/calendars/%s/", base, username, calDAVSlug(cal.Name))
			ctag := h.calCTag(userID, cal.ID)
			responses = append(responses, propResponse{
				Href: calURL,
				Props: []propstat{{
					Status: "HTTP/1.1 200 OK",
					Props: []xml.TokenReader{
						xmlProp("D:resourcetype", `<D:collection/><C:calendar/>`),
						xmlProp("D:displayname", xmlEsc(cal.Name)),
						xmlPropRaw("CS:getctag", ctag),
						xmlPropRaw("D:sync-token", ctag),
						xmlProp("C:supported-calendar-component-set",
							`<C:comp name="VEVENT"/>`),
						xmlPropRaw("ICAL:calendar-color", cal.Color),
					},
				}},
			})
		}
	}

	return writePropfind(c, multistatus(responses))
}

// ── PROPFIND /dav/{user}/calendars/{cal}/ — calendar collection ──────────

func (h *CalDAVHandler) propfindCalendar(c *echo.Context, userID uint, username, calSlug, depth string) error {
	base := davBase(h.cfg)
	cal, err := h.resolveCalendar(userID, calSlug)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	calURL := fmt.Sprintf("%s/dav/%s/calendars/%s/", base, username, calDAVSlug(cal.Name))
	ctag := h.calCTag(userID, cal.ID)

	responses := []propResponse{
		{
			Href: calURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					xmlProp("D:resourcetype", `<D:collection/><C:calendar/>`),
					xmlProp("D:displayname", xmlEsc(cal.Name)),
					xmlPropRaw("CS:getctag", ctag),
					xmlPropRaw("D:sync-token", ctag),
					xmlProp("C:supported-calendar-component-set", `<C:comp name="VEVENT"/>`),
					xmlPropRaw("ICAL:calendar-color", cal.Color),
				},
			}},
		},
	}

	if depth == "1" {
		events, err := h.eventRepo.ListByCalendar(userID, cal.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		for _, ev := range events {
			evURL := fmt.Sprintf("%s/dav/%s/calendars/%s/%s.ics",
				base, username, calDAVSlug(cal.Name), ev.UID)
			etag := eventETag(ev)
			responses = append(responses, propResponse{
				Href: evURL,
				Props: []propstat{{
					Status: "HTTP/1.1 200 OK",
					Props: []xml.TokenReader{
						xmlPropRaw("D:getetag", `"`+etag+`"`),
						xmlProp("D:getcontenttype", "text/calendar; charset=utf-8"),
						xmlProp("D:resourcetype", ""),
					},
				}},
			})
		}
	}

	return writePropfind(c, multistatus(responses))
}

// ── PROPFIND on individual event ──────────────────────────────────────────

func (h *CalDAVHandler) propfindEvent(c *echo.Context, userID uint, username, calSlug, uid string) error {
	base := davBase(h.cfg)
	cal, err := h.resolveCalendar(userID, calSlug)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	ev, err := h.eventRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	evURL := fmt.Sprintf("%s/dav/%s/calendars/%s/%s.ics",
		base, username, calDAVSlug(cal.Name), ev.UID)
	etag := eventETag(*ev)
	resp := multistatus([]propResponse{{
		Href: evURL,
		Props: []propstat{{
			Status: "HTTP/1.1 200 OK",
			Props: []xml.TokenReader{
				xmlPropRaw("D:getetag", `"`+etag+`"`),
				xmlProp("D:getcontenttype", "text/calendar; charset=utf-8"),
				xmlProp("D:resourcetype", ""),
			},
		}},
	}})
	return writePropfind(c, resp)
}

// ── REPORT ────────────────────────────────────────────────────────────────

// Report dispatches CalDAV REPORT (calendar-query, calendar-multiget).
func (h *CalDAVHandler) Report(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	username := caldavUsername(c)
	calSlug := c.Param("cal")

	cal, err := h.resolveCalendar(userID, calSlug)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	body, _ := io.ReadAll(io.LimitReader(c.Request().Body, 512*1024))
	bodyStr := string(body)

	if strings.Contains(bodyStr, "calendar-multiget") {
		return h.reportMultiget(c, userID, username, cal, bodyStr)
	}
	return h.reportCalendarQuery(c, userID, username, cal, bodyStr)
}

// reportCalendarQuery handles REPORT calendar-query (date-range + component filter).
func (h *CalDAVHandler) reportCalendarQuery(c *echo.Context, userID uint, username string, cal *model.Calendar, body string) error {
	base := davBase(h.cfg)

	// Parse optional date range from VCALENDAR filter / time-range element.
	start, end := parseTimeRange(body)
	var events []model.Event
	var err error
	if !start.IsZero() && !end.IsZero() {
		events, err = h.eventRepo.ListByRange(userID, start, end, []uint{cal.ID})
	} else {
		events, err = h.eventRepo.ListByCalendar(userID, cal.ID)
	}
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	wantData := strings.Contains(body, "calendar-data")
	responses := buildEventResponses(base, username, cal, events, wantData)
	return writePropfind(c, multistatus(responses))
}

// reportMultiget handles REPORT calendar-multiget (fetch events by href list).
func (h *CalDAVHandler) reportMultiget(c *echo.Context, userID uint, username string, cal *model.Calendar, body string) error {
	base := davBase(h.cfg)
	hrefs := parseHrefs(body)

	var responses []propResponse
	for _, href := range hrefs {
		uid := hrefToUID(href)
		if uid == "" {
			continue
		}
		ev, err := h.eventRepo.GetByUID(userID, uid)
		if err != nil {
			responses = append(responses, propResponse{
				Href: href,
				Props: []propstat{{Status: "HTTP/1.1 404 Not Found", Props: nil}},
			})
			continue
		}
		ics := ev.ICalContent
		if ics == "" {
			ics = calpkg.BuildICalContent(ev, ev.Attendees)
		}
		evURL := fmt.Sprintf("%s/dav/%s/calendars/%s/%s.ics",
			base, username, calDAVSlug(cal.Name), ev.UID)
		etag := eventETag(*ev)
		responses = append(responses, propResponse{
			Href: evURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					xmlPropRaw("D:getetag", `"`+etag+`"`),
					xmlProp("D:getcontenttype", "text/calendar; charset=utf-8"),
					xmlPropRaw("C:calendar-data", xmlEsc(ics)),
				},
			}},
		})
	}
	return writePropfind(c, multistatus(responses))
}

// ── GET / PUT / DELETE event resources ───────────────────────────────────

// GetEvent handles GET /dav/{user}/calendars/{cal}/{uid}.ics.
func (h *CalDAVHandler) GetEvent(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".ics")
	ev, err := h.eventRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	ics := ev.ICalContent
	if ics == "" {
		ics = calpkg.BuildICalContent(ev, ev.Attendees)
	}
	etag := eventETag(*ev)
	c.Response().Header().Set("ETag", `"`+etag+`"`)
	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

// GetCalendar handles GET /dav/{user}/calendars/{cal}/ — full ICS export fallback.
func (h *CalDAVHandler) GetCalendar(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	cal, err := h.resolveCalendar(userID, c.Param("cal"))
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	events, err := h.eventRepo.ListByCalendar(userID, cal.ID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	ics := calpkg.BuildCalendarExport(events)
	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

// PutEvent handles PUT /dav/{user}/calendars/{cal}/{uid}.ics — create or update.
func (h *CalDAVHandler) PutEvent(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	cal, err := h.resolveCalendar(userID, c.Param("cal"))
	if err != nil {
		// fall back to default calendar
		cal, err = h.calRepo.EnsureDefault(userID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	body, _ := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	parsed, err := calpkg.ParseICalImport(body)
	if err != nil || len(parsed) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}
	item := parsed[0]

	existing, err := h.eventRepo.GetByUID(userID, item.UID)
	if err == gorm.ErrRecordNotFound {
		ev := model.Event{
			CalendarID: cal.ID, UserID: userID, UID: item.UID,
			Summary: item.Summary, Description: item.Description, Location: item.Location,
			StartAt: item.StartAt, EndAt: item.EndAt, IsAllDay: item.IsAllDay,
			Status: item.Status, RRule: item.RRule, Attendees: item.Attendees,
		}
		if ev.Status == "" {
			ev.Status = "CONFIRMED"
		}
		ev.ICalContent = calpkg.BuildICalContent(&ev, item.Attendees)
		if err := h.eventRepo.Create(&ev); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		etag := eventETag(ev)
		c.Response().Header().Set("ETag", `"`+etag+`"`)
		return c.NoContent(http.StatusCreated)
	}
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	existing.Summary = item.Summary
	existing.Description = item.Description
	existing.Location = item.Location
	existing.StartAt = item.StartAt
	existing.EndAt = item.EndAt
	existing.IsAllDay = item.IsAllDay
	existing.Status = item.Status
	existing.RRule = item.RRule
	existing.Attendees = item.Attendees
	existing.Sequence++
	existing.ICalContent = calpkg.BuildICalContent(existing, item.Attendees)
	if err := h.eventRepo.Update(existing); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	etag := eventETag(*existing)
	c.Response().Header().Set("ETag", `"`+etag+`"`)
	return c.NoContent(http.StatusNoContent)
}

// DeleteEvent handles DELETE /dav/{user}/calendars/{cal}/{uid}.ics.
func (h *CalDAVHandler) DeleteEvent(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".ics")
	ev, err := h.eventRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	if err := h.eventRepo.Delete(userID, ev.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Helpers ───────────────────────────────────────────────────────────────

func (h *CalDAVHandler) resolveCalendar(userID uint, slug string) (*model.Calendar, error) {
	cals, err := h.calRepo.List(userID)
	if err != nil {
		return nil, err
	}
	for _, cal := range cals {
		if calDAVSlug(cal.Name) == slug {
			return &cal, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// calCTag returns a change token for a calendar collection based on the latest event update.
func (h *CalDAVHandler) calCTag(userID, calendarID uint) string {
	events, err := h.eventRepo.ListByCalendar(userID, calendarID)
	if err != nil || len(events) == 0 {
		return "0"
	}
	var latest int64
	for _, ev := range events {
		if t := ev.UpdatedAt.Unix(); t > latest {
			latest = t
		}
	}
	return strconv.FormatInt(latest, 10)
}

// eventETag produces a stable ETag string for a single event.
func eventETag(ev model.Event) string {
	return fmt.Sprintf("%d-%d", ev.Sequence, ev.UpdatedAt.Unix())
}

func davBase(cfg *config.Config) string {
	return strings.TrimRight(cfg.Server.BaseURL, "/")
}

// calDAVSlug converts a calendar name to a URL-safe slug.
func calDAVSlug(name string) string {
	s := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "calendar"
	}
	return b.String()
}

// xmlEsc escapes XML special characters.
func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// buildEventResponses creates REPORT/PROPFIND response entries for a slice of events.
func buildEventResponses(base, username string, cal *model.Calendar, events []model.Event, wantData bool) []propResponse {
	out := make([]propResponse, 0, len(events))
	for _, ev := range events {
		evURL := fmt.Sprintf("%s/dav/%s/calendars/%s/%s.ics",
			base, username, calDAVSlug(cal.Name), ev.UID)
		etag := eventETag(ev)
		props := []xml.TokenReader{
			xmlPropRaw("D:getetag", `"`+etag+`"`),
			xmlProp("D:getcontenttype", "text/calendar; charset=utf-8"),
		}
		if wantData {
			ics := ev.ICalContent
			if ics == "" {
				ics = calpkg.BuildICalContent(&ev, ev.Attendees)
			}
			props = append(props, xmlPropRaw("C:calendar-data", xmlEsc(ics)))
		}
		out = append(out, propResponse{
			Href:  evURL,
			Props: []propstat{{Status: "HTTP/1.1 200 OK", Props: props}},
		})
	}
	return out
}

// parseTimeRange extracts start/end from a VCALENDAR time-range XML element.
func parseTimeRange(body string) (start, end time.Time) {
	// Look for time-range start="..." end="..."
	lower := strings.ToLower(body)
	idx := strings.Index(lower, "time-range")
	if idx < 0 {
		return
	}
	sub := body[idx : idx+min(200, len(body)-idx)]
	start = extractAttr(sub, "start")
	end = extractAttr(sub, "end")
	return
}

func extractAttr(s, attr string) time.Time {
	key := attr + `="`
	i := strings.Index(strings.ToLower(s), key)
	if i < 0 {
		return time.Time{}
	}
	s = s[i+len(key):]
	j := strings.IndexByte(s, '"')
	if j < 0 {
		return time.Time{}
	}
	val := s[:j]
	for _, layout := range []string{"20060102T150405Z", "20060102T150405", "20060102"} {
		if t, err := time.Parse(layout, val); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseHrefs extracts <D:href>...</D:href> values from a REPORT body.
func parseHrefs(body string) []string {
	var hrefs []string
	for {
		open := strings.Index(body, "<")
		if open < 0 {
			break
		}
		rest := body[open:]
		// match <D:href> or <href> (case-insensitive tag match)
		var tag, content string
		for _, t := range []string{"D:href", "href"} {
			o := "<" + t + ">"
			c := "</" + t + ">"
			if i := strings.Index(strings.ToLower(rest), strings.ToLower(o)); i >= 0 {
				inner := rest[i+len(o):]
				if j := strings.Index(strings.ToLower(inner), strings.ToLower(c)); j >= 0 {
					tag = t
					content = inner[:j]
					body = inner[j+len(c):]
					break
				}
			}
		}
		if tag == "" {
			break
		}
		if content != "" {
			hrefs = append(hrefs, strings.TrimSpace(content))
		}
	}
	return hrefs
}

// hrefToUID extracts the event UID from a CalDAV event href.
// e.g. "/dav/user/calendars/personal/my-uid.ics" → "my-uid"
func hrefToUID(href string) string {
	parts := strings.Split(strings.TrimSuffix(href, ".ics"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// ── XML response builders ─────────────────────────────────────────────────

type propResponse struct {
	Href  string
	Props []propstat
}

type propstat struct {
	Status string
	Props  []xml.TokenReader
}

// xmlProp creates an XML element with escaped text content.
func xmlProp(name, content string) xml.TokenReader { return rawXML("<" + name + ">" + content + "</" + name + ">") }

// xmlPropRaw creates an XML element with raw (pre-escaped) content.
func xmlPropRaw(name, content string) xml.TokenReader {
	return rawXML("<" + name + ">" + content + "</" + name + ">")
}

type rawXML string

func (r rawXML) Token() (xml.Token, error) { return xml.CharData(r), nil }

// multistatus serialises propResponse slice to a DAV:multistatus XML string.
func multistatus(responses []propResponse) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/" xmlns:ICAL="http://apple.com/ns/ical/">`)
	for _, r := range responses {
		sb.WriteString(`<D:response>`)
		sb.WriteString(`<D:href>` + xmlEsc(r.Href) + `</D:href>`)
		for _, ps := range r.Props {
			sb.WriteString(`<D:propstat><D:prop>`)
			for _, p := range ps.Props {
				if rp, ok := p.(rawXML); ok {
					sb.WriteString(string(rp))
				}
			}
			sb.WriteString(`</D:prop>`)
			sb.WriteString(`<D:status>` + ps.Status + `</D:status>`)
			sb.WriteString(`</D:propstat>`)
		}
		sb.WriteString(`</D:response>`)
	}
	sb.WriteString(`</D:multistatus>`)
	return sb.String()
}

// writePropfind writes a 207 Multi-Status XML response.
func writePropfind(c *echo.Context, body string) error {
	c.Response().Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.Response().Header().Set("DAV", "1, 2, calendar-access")
	return c.Blob(http.StatusMultiStatus, "application/xml; charset=utf-8", []byte(body))
}
