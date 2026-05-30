# Calendar Backend — Go Function Reference

> **Project:** go-cubemail-vue  
> **Packages:** `internal/model`, `internal/repository`, `internal/calendar`, `internal/handler`  
> **Related:** [Calendar cURL Testing Guide](CALENDAR_CURL_TESTING.md)

This document lists every type and function in the calendar backend with a short description. Full godoc comments live in the source files (`go doc` / IDE hover).

---

## Table of Contents

1. [Models (`internal/model`)](#1-models-internalmodel)
2. [Repositories (`internal/repository`)](#2-repositories-internalrepository)
3. [iCalendar helpers (`internal/calendar`)](#3-icalendar-helpers-internalcalendar)
4. [HTTP handlers (`internal/handler`)](#4-http-handlers-internalhandler)
5. [Shared helpers (`internal/handler/user.go`)](#5-shared-helpers-internalhandlerusergo)
6. [View godoc locally](#6-view-godoc-locally)

---

## 1. Models (`internal/model`)

### `calendar.go`

| Name | Kind | Description |
|------|------|-------------|
| `Calendar` | struct | User-owned calendar folder. Fields: ID, UserID, Name, Color, IsDefault, IsActive, IncludeInFreeBusy, SortOrder, timestamps. |

### `event.go`

| Name | Kind | Description |
|------|------|-------------|
| `Event` | struct | VEVENT record with denormalized fields + `ICalContent` blob. Relations: `Attendees`, `Calendar`. |

### `event_attendee.go`

| Name | Kind | Description |
|------|------|-------------|
| `EventAttendee` | struct | Event participant with email, PartStat (RSVP state), Role, RSVP flag. |

---

## 2. Repositories (`internal/repository`)

### `calendar.go` — `CalendarRepo`

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewCalendarRepo` | `(db *gorm.DB) *CalendarRepo` | Constructs a repo backed by the given DB connection. |
| `List` | `(userID uint) ([]model.Calendar, error)` | Returns all calendars for the user, ordered by sort_order then name. |
| `Get` | `(userID, id uint) (*model.Calendar, error)` | Fetches one calendar scoped to userID; returns `ErrRecordNotFound` when missing. |
| `Create` | `(cal *model.Calendar) error` | Inserts a new calendar row. |
| `Update` | `(cal *model.Calendar) error` | Saves all fields on an existing calendar (full update). |
| `Delete` | `(userID, id uint) error` | Transaction: deletes all events in the calendar, then the calendar row. |
| `EnsureDefault` | `(userID uint) (*model.Calendar, error)` | Returns the default calendar, creating "Personal" (#3788d8) if none exists. |
| `SetActive` | `(userID uint, ids []uint, active bool) error` | Bulk-updates `is_active` for the given calendar IDs. |
| `ListActive` | `(userID uint) ([]model.Calendar, error)` | Returns only calendars where `is_active = true`. |

### `event.go` — `EventRepo`

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewEventRepo` | `(db *gorm.DB) *EventRepo` | Constructs a repo backed by the given DB connection. |
| `ListByRange` | `(userID uint, start, end time.Time, calendarIDs []uint) ([]model.Event, error)` | Returns events overlapping `[start, end)`; optional calendar ID filter; preloads Attendees + Calendar. |
| `ListByCalendar` | `(userID, calendarID uint) ([]model.Event, error)` | Returns all events in one calendar (used for ICS export). |
| `Get` | `(userID, id uint) (*model.Event, error)` | Fetches one event with attendees and calendar color. |
| `GetByUID` | `(userID uint, uid string) (*model.Event, error)` | Looks up an event by iCalendar UID (used on ICS import upsert). |
| `Create` | `(event *model.Event) error` | Transaction: inserts event, then attendee rows with default PartStat/Role. |
| `Update` | `(event *model.Event) error` | Transaction: saves event, deletes old attendees, re-inserts new attendee list. |
| `Delete` | `(userID, id uint) error` | Deletes one event; returns `ErrRecordNotFound` when no row matched. |

---

## 3. iCalendar helpers (`internal/calendar`)

### Exported

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewUID` | `(domain string) string` | Generates `{16-byte-hex}@{domain}` UID; default domain `go-cubemail`. |
| `BuildICalContent` | `(event *model.Event, attendees []model.EventAttendee) string` | Builds a single-event VCALENDAR/VEVENT RFC 5545 document. |
| `BuildCalendarExport` | `(events []model.Event) string` | Merges multiple VEVENT blocks into one VCALENDAR for download. |
| `ParseICalImport` | `(data []byte) ([]ImportEvent, error)` | Parses ICS bytes; unfolds lines; extracts VEVENT fields and attendees. |

### Types

| Name | Description |
|------|-------------|
| `ImportEvent` | Intermediate struct holding parsed ICS fields before DB insert. |

### Unexported helpers

| Function | Description |
|----------|-------------|
| `parseICalTime` | Parses DTSTART/DTEND; detects VALUE=DATE (all-day). |
| `extractParam` | Reads ICS property parameters (CN, PARTSTAT, ROLE, RSVP). |
| `foldLine` | Escapes `\n` in SUMMARY/DESCRIPTION for ICS output. |

### Tests (`ical_test.go`)

| Function | Description |
|----------|-------------|
| `TestParseICalImport` | Verifies UID, SUMMARY, and DTSTART parsing from sample ICS. |
| `TestBuildICalContent` | Asserts VEVENT block and SUMMARY in generated output. |
| `TestNewUID` | Asserts UID suffix matches the supplied domain. |

---

## 4. HTTP handlers (`internal/handler`)

### `calendar.go` — `CalendarHandler`

| Field | Type | Description |
|-------|------|-------------|
| `cfg` | `*config.Config` | Application configuration. |
| `db` | `*gorm.DB` | Database handle for user lookup. |
| `calRepo` | `*repository.CalendarRepo` | Calendar persistence. |
| `eventRepo` | `*repository.EventRepo` | Event persistence (export/import). |

| Method | Route | HTTP | Description |
|--------|-------|------|-------------|
| `List` | `/api/v1/calendar` | GET | Lists calendars; auto-creates default. |
| `Create` | `/api/v1/calendar` | POST | Creates calendar; requires `name`. |
| `Update` | `/api/v1/calendar/:id` | PUT | Partial update of name, color, flags. |
| `Delete` | `/api/v1/calendar/:id` | DELETE | Deletes calendar + events; blocks default. |
| `SetActivation` | `/api/v1/calendar/activation` | POST | Sets `is_active` on multiple IDs. |
| `Export` | `/api/v1/calendar/:id/export` | GET | Downloads `.ics` file. |
| `Import` | `/api/v1/calendar/:id/import` | POST | Multipart ICS upload; upserts by UID. |

| Helper | Description |
|--------|-------------|
| `toCalendarResponse` | Maps `model.Calendar` → JSON; applies default color `#3788d8`. |

### `event.go` — `EventHandler`

| Method | Route | HTTP | Description |
|--------|-------|------|-------------|
| `List` | `/api/v1/events?start=&end=` | GET | Range query; optional `calendar_ids`. |
| `Get` | `/api/v1/events/:id` | GET | Single event with attendees. |
| `Create` | `/api/v1/events` | POST | Creates event; default calendar if ID omitted. |
| `Update` | `/api/v1/events/:id` | PUT | Full replace; increments SEQUENCE. |
| `Delete` | `/api/v1/events/:id` | DELETE | Removes event and attendees (cascade). |
| `Move` | `/api/v1/events/:id/move` | POST | Drag-resize: new start/end and optional calendar. |

| Helper | Description |
|--------|-------------|
| `formatTime` | UTC RFC3339 for JSON. |
| `parseTime` | Parses RFC3339 or `2006-01-02T15:04:05Z`. |
| `toAttendeeResponses` | Model → JSON attendees. |
| `toEventResponse` | Model → JSON event (color from calendar). |
| `toAttendeeModels` | JSON → model attendees; skips empty email. |
| `applyEventRequest` | Validates and applies create/update body. |
| `parseCalendarIDs` | Parses comma-separated `calendar_ids` query param. |

---

## 5. Shared helpers (`internal/handler/user.go`)

| Function | Signature | Description |
|----------|-----------|-------------|
| `getUserID` | `(c *echo.Context, db *gorm.DB) (uint, error)` | Maps IMAP session username → `model.User.ID` (FirstOrCreate). |
| `getSessionUsername` | `(c *echo.Context) string` | Returns the authenticated IMAP username from context. |

---

## 6. View godoc locally

From the project root:

```bash
# All calendar packages
go doc go-cubemail/internal/calendar
go doc go-cubemail/internal/repository.CalendarRepo
go doc go-cubemail/internal/handler.CalendarHandler

# Single function
go doc go-cubemail/internal/calendar.NewUID
go doc go-cubemail/internal/handler.EventHandler.Create

# Run unit tests with verbose output
go test -v ./internal/calendar/...
```

---

*Document version: 1.0 — Calendar backend Go function reference.*
