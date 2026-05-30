package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalDAVHandler provides a minimal CalDAV (RFC 4791) interface for calendar clients
// such as Apple Calendar, Thunderbird, and Evolution.
//
// Endpoints exposed:
//   GET  /dav/{user}/calendars/                  → calendar list (PROPFIND-lite)
//   GET  /dav/{user}/calendars/{cal}/            → event list in .ics URLs
//   GET  /dav/{user}/calendars/{cal}/{uid}.ics   → single event ICS
//   PUT  /dav/{user}/calendars/{cal}/{uid}.ics   → create/update event
//   DELETE /dav/{user}/calendars/{cal}/{uid}.ics → delete event
//
// Full WebDAV PROPFIND/REPORT is not implemented; this is enough for export/import
// workflows and basic sync with tolerant clients.
type CalDAVHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	eventRepo *repository.EventRepo
}

// PropFind handles PROPFIND /dav/{user}/calendars/ — returns a minimal multi-status XML
// listing calendar collections for tolerant clients.
func (h *CalDAVHandler) PropFind(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}
	if _, err := h.calRepo.EnsureDefault(userID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	calendars, err := h.calRepo.List(userID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	username := getSessionUsername(c)

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">`)
	for _, cal := range calendars {
		href := fmt.Sprintf("%s/dav/%s/calendars/%s/", base, username, calDAVSlug(cal.Name))
		sb.WriteString(`<D:response>`)
		sb.WriteString(`<D:href>` + href + `</D:href>`)
		sb.WriteString(`<D:propstat><D:prop>`)
		sb.WriteString(`<D:resourcetype><D:collection/><C:calendar/></D:resourcetype>`)
		sb.WriteString(`<D:displayname>` + xmlEsc(cal.Name) + `</D:displayname>`)
		sb.WriteString(`<C:supported-calendar-component-set><C:comp name="VEVENT"/></C:supported-calendar-component-set>`)
		sb.WriteString(`</D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>`)
		sb.WriteString(`</D:response>`)
	}
	sb.WriteString(`</D:multistatus>`)

	c.Response().Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.Response().Header().Set("DAV", "1, 2, calendar-access")
	return c.Blob(http.StatusMultiStatus, "application/xml; charset=utf-8", []byte(sb.String()))
}

// GetCalendar handles GET /dav/{user}/calendars/{cal}/ — returns all events as ICS.
func (h *CalDAVHandler) GetCalendar(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
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

// GetEvent handles GET /dav/{user}/calendars/{cal}/{uid}.ics — returns a single event.
func (h *CalDAVHandler) GetEvent(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".ics")
	event, err := h.eventRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	ics := event.ICalContent
	if ics == "" {
		ics = calpkg.BuildICalContent(event, event.Attendees)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, event.Sequence))
	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

// PutEvent handles PUT /dav/{user}/calendars/{cal}/{uid}.ics — upserts an event from ICS.
func (h *CalDAVHandler) PutEvent(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}
	cal, err := h.resolveCalendar(userID, c.Param("cal"))
	if err != nil {
		// Create calendar on demand.
		newCal, cerr := h.calRepo.EnsureDefault(userID)
		if cerr != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		cal = newCal
	}

	body, _ := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	parsed, err := calpkg.ParseICalImport(body)
	if err != nil || len(parsed) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}
	item := parsed[0]

	existing, err := h.eventRepo.GetByUID(userID, item.UID)
	if err == gorm.ErrRecordNotFound {
		event := model.Event{
			CalendarID:  cal.ID,
			UserID:      userID,
			UID:         item.UID,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			StartAt:     item.StartAt,
			EndAt:       item.EndAt,
			IsAllDay:    item.IsAllDay,
			Status:      item.Status,
			RRule:       item.RRule,
			Attendees:   item.Attendees,
		}
		if event.Status == "" {
			event.Status = "CONFIRMED"
		}
		event.ICalContent = calpkg.BuildICalContent(&event, item.Attendees)
		if err := h.eventRepo.Create(&event); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		c.Response().Header().Set("ETag", `"0"`)
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
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, existing.Sequence))
	return c.NoContent(http.StatusNoContent)
}

// DeleteEvent handles DELETE /dav/{user}/calendars/{cal}/{uid}.ics.
func (h *CalDAVHandler) DeleteEvent(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".ics")
	event, err := h.eventRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	if err := h.eventRepo.Delete(userID, event.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// Options handles OPTIONS for CalDAV capability discovery.
func (h *CalDAVHandler) Options(c *echo.Context) error {
	c.Response().Header().Set("DAV", "1, 2, calendar-access")
	c.Response().Header().Set("Allow", "OPTIONS, GET, PUT, DELETE, PROPFIND, REPORT")
	return c.NoContent(http.StatusOK)
}

// resolveCalendar finds a calendar by slug (URL-friendly name) for the given user.
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

// calDAVSlug converts a calendar name to a URL-safe slug.
func calDAVSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
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

// xmlEsc escapes special XML characters in a string.
func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
