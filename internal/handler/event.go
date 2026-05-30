package handler

import (
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

// EventHandler handles CRUD operations for calendar events.
type EventHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	eventRepo *repository.EventRepo
}

// attendeeRequest is the JSON shape for an event participant on create/update.
type attendeeRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	PartStat string `json:"partstat"`
	Role     string `json:"role"`
	RSVP     *bool  `json:"rsvp"`
}

// eventRequest is the JSON body accepted by event create/update endpoints.
type eventRequest struct {
	CalendarID     uint              `json:"calendar_id"`
	Summary        string            `json:"summary"`
	Description    string            `json:"description"`
	Location       string            `json:"location"`
	StartAt        string            `json:"start_at"`
	EndAt          string            `json:"end_at"`
	IsAllDay       bool              `json:"is_all_day"`
	IsTransparent  bool              `json:"is_transparent"`
	Status         string            `json:"status"`
	Priority       int               `json:"priority"`
	Classification string            `json:"classification"`
	Categories     string            `json:"categories"`
	OrganizerName  string            `json:"organizer_name"`
	OrganizerEmail string            `json:"organizer_email"`
	RRule          string            `json:"rrule"`
	Attendees      []attendeeRequest `json:"attendees"`
}

// attendeeResponse is the JSON shape for an event participant in API responses.
type attendeeResponse struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	PartStat string `json:"partstat"`
	Role     string `json:"role"`
	RSVP     bool   `json:"rsvp"`
}

// eventResponse is the JSON shape returned by all event endpoints.
type eventResponse struct {
	ID             uint               `json:"id"`
	CalendarID     uint               `json:"calendar_id"`
	UID            string             `json:"uid"`
	Summary        string             `json:"summary"`
	Description    string             `json:"description"`
	Location       string             `json:"location"`
	StartAt        string             `json:"start_at"`
	EndAt          string             `json:"end_at"`
	IsAllDay       bool               `json:"is_all_day"`
	IsTransparent  bool               `json:"is_transparent"`
	Status         string             `json:"status"`
	Priority       int                `json:"priority"`
	Classification string             `json:"classification"`
	Categories     string             `json:"categories"`
	OrganizerName  string             `json:"organizer_name"`
	OrganizerEmail string             `json:"organizer_email"`
	RRule          string             `json:"rrule"`
	IsRecurring    bool               `json:"is_recurring"`
	Color          string             `json:"color"`
	Attendees      []attendeeResponse `json:"attendees,omitempty"`
}

// eventListResponse wraps the list of events returned by GET /events.
type eventListResponse struct {
	Events []eventResponse `json:"events"`
}

// eventMoveRequest is the JSON body for POST /events/:id/move (drag-resize).
type eventMoveRequest struct {
	StartAt    string `json:"start_at"`
	EndAt      string `json:"end_at"`
	CalendarID *uint  `json:"calendar_id"`
}

// formatTime serializes a time.Time as UTC RFC3339 for JSON responses.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// parseTime parses an ISO 8601 timestamp from a query parameter or JSON field.
// Supported layouts: RFC3339 and 2006-01-02T15:04:05Z.
func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	return time.Parse("2006-01-02T15:04:05Z", value)
}

// toAttendeeResponses maps model attendees to API response objects.
func toAttendeeResponses(attendees []model.EventAttendee) []attendeeResponse {
	out := make([]attendeeResponse, len(attendees))
	for i, a := range attendees {
		out[i] = attendeeResponse{
			Name:     a.Name,
			Email:    a.Email,
			PartStat: a.PartStat,
			Role:     a.Role,
			RSVP:     a.RSVP,
		}
	}
	return out
}

// toEventResponse maps a model.Event (with preloaded Calendar and Attendees)
// to the API response shape, including calendar color and recurrence flag.
func toEventResponse(event model.Event) eventResponse {
	color := event.Calendar.Color
	if color == "" {
		color = "#3788d8"
	}
	status := event.Status
	if status == "" {
		status = "CONFIRMED"
	}
	return eventResponse{
		ID:             event.ID,
		CalendarID:     event.CalendarID,
		UID:            event.UID,
		Summary:        event.Summary,
		Description:    event.Description,
		Location:       event.Location,
		StartAt:        formatTime(event.StartAt),
		EndAt:          formatTime(event.EndAt),
		IsAllDay:       event.IsAllDay,
		IsTransparent:  event.IsTransparent,
		Status:         status,
		Priority:       event.Priority,
		Classification: event.Classification,
		Categories:     event.Categories,
		OrganizerName:  event.OrganizerName,
		OrganizerEmail: event.OrganizerEmail,
		RRule:          event.RRule,
		IsRecurring:    event.RRule != "",
		Color:          color,
		Attendees:      toAttendeeResponses(event.Attendees),
	}
}

// toAttendeeModels converts JSON attendee requests to model rows,
// skipping entries with an empty email address.
func toAttendeeModels(reqs []attendeeRequest) []model.EventAttendee {
	out := make([]model.EventAttendee, 0, len(reqs))
	for _, req := range reqs {
		email := strings.TrimSpace(req.Email)
		if email == "" {
			continue
		}
		a := model.EventAttendee{
			Name:     strings.TrimSpace(req.Name),
			Email:    email,
			PartStat: req.PartStat,
			Role:     req.Role,
			RSVP:     true,
		}
		if req.RSVP != nil {
			a.RSVP = *req.RSVP
		}
		out = append(out, a)
	}
	return out
}

// applyEventRequest copies validated fields from eventRequest onto a model.Event.
// Returns echo.HTTPError with 400 status when required fields are missing or invalid.
func applyEventRequest(event *model.Event, req eventRequest) error {
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "summary is required")
	}
	startAt, err := parseTime(req.StartAt)
	if err != nil || startAt.IsZero() {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start_at")
	}
	endAt, err := parseTime(req.EndAt)
	if err != nil || endAt.IsZero() {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end_at")
	}
	if !endAt.After(startAt) && !req.IsAllDay {
		return echo.NewHTTPError(http.StatusBadRequest, "end_at must be after start_at")
	}

	event.Summary = summary
	event.Description = req.Description
	event.Location = req.Location
	event.StartAt = startAt
	event.EndAt = endAt
	event.IsAllDay = req.IsAllDay
	event.IsTransparent = req.IsTransparent
	if req.Status != "" {
		event.Status = req.Status
	} else if event.Status == "" {
		event.Status = "CONFIRMED"
	}
	event.Priority = req.Priority
	if req.Classification != "" {
		event.Classification = req.Classification
	} else if event.Classification == "" {
		event.Classification = "PUBLIC"
	}
	event.Categories = req.Categories
	event.OrganizerName = req.OrganizerName
	event.OrganizerEmail = req.OrganizerEmail
	event.RRule = req.RRule
	event.Attendees = toAttendeeModels(req.Attendees)
	return nil
}

// parseCalendarIDs parses a comma-separated list of calendar IDs from a query string.
// Returns nil when raw is empty (no filter).
func parseCalendarIDs(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uint(id))
	}
	return ids, nil
}

// FreeBusy handles GET /api/v1/events/freebusy?start=&end=.
// Returns busy periods for the authenticated user within the requested range.
// Used by the event editor to show availability.
func (h *EventHandler) FreeBusy(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	start, err := parseTime(c.QueryParam("start"))
	if err != nil || start.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start"})
	}
	end, err := parseTime(c.QueryParam("end"))
	if err != nil || end.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end"})
	}

	events, err := h.eventRepo.ListByRange(userID, start, end, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	type busySlot struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Title string `json:"title,omitempty"`
	}
	var slots []busySlot
	for _, ev := range events {
		if ev.IsTransparent {
			continue
		}
		slots = append(slots, busySlot{
			Start: formatTime(ev.StartAt),
			End:   formatTime(ev.EndAt),
			Title: ev.Summary,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"user":  getSessionUsername(c),
		"start": formatTime(start),
		"end":   formatTime(end),
		"busy":  slots,
	})
}

// List handles GET /api/v1/events?start=&end=&calendar_ids=.
// Returns events overlapping the requested UTC time range.
func (h *EventHandler) List(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	if _, err := h.calRepo.EnsureDefault(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	start, err := parseTime(c.QueryParam("start"))
	if err != nil || start.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start parameter"})
	}
	end, err := parseTime(c.QueryParam("end"))
	if err != nil || end.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end parameter"})
	}

	calendarIDs, err := parseCalendarIDs(c.QueryParam("calendar_ids"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid calendar_ids parameter"})
	}

	events, err := h.eventRepo.ListByRange(userID, start, end, calendarIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]eventResponse, len(events))
	for i, event := range events {
		out[i] = toEventResponse(event)
	}
	return c.JSON(http.StatusOK, eventListResponse{Events: out})
}

// Get handles GET /api/v1/events/:id.
func (h *EventHandler) Get(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	event, err := h.eventRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return c.JSON(http.StatusOK, toEventResponse(*event))
}

// Create handles POST /api/v1/events.
// Uses the default calendar when calendar_id is omitted or zero.
func (h *EventHandler) Create(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req eventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if req.CalendarID == 0 {
		defaultCal, err := h.calRepo.EnsureDefault(userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		req.CalendarID = defaultCal.ID
	}
	if _, err := h.calRepo.Get(userID, req.CalendarID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid calendar_id"})
	}

	event := model.Event{
		CalendarID: req.CalendarID,
		UserID:     userID,
		UID:        calpkg.NewUID(""),
	}
	if err := applyEventRequest(&event, req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return c.JSON(he.Code, map[string]string{"error": he.Message})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	event.ICalContent = calpkg.BuildICalContent(&event, event.Attendees)
	if err := h.eventRepo.Create(&event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	created, err := h.eventRepo.Get(userID, event.ID)
	if err != nil {
		return c.JSON(http.StatusCreated, toEventResponse(event))
	}
	return c.JSON(http.StatusCreated, toEventResponse(*created))
}

// Update handles PUT /api/v1/events/:id.
// Replaces all event fields and regenerates ICalContent with an incremented sequence.
func (h *EventHandler) Update(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	event, err := h.eventRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	var req eventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if req.CalendarID != 0 && req.CalendarID != event.CalendarID {
		if _, err := h.calRepo.Get(userID, req.CalendarID); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid calendar_id"})
		}
		event.CalendarID = req.CalendarID
	}
	if err := applyEventRequest(event, req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return c.JSON(he.Code, map[string]string{"error": he.Message})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	event.Sequence++
	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := h.eventRepo.Update(event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	updated, err := h.eventRepo.Get(userID, event.ID)
	if err != nil {
		return c.JSON(http.StatusOK, toEventResponse(*event))
	}
	return c.JSON(http.StatusOK, toEventResponse(*updated))
}

// Delete handles DELETE /api/v1/events/:id.
func (h *EventHandler) Delete(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.eventRepo.Delete(userID, uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// RSVPFromMail handles POST /api/v1/events/rsvp-from-mail.
// Accepts {uid, partstat} and updates the attendee status on the matching event,
// creating a placeholder event from the email invitation when none exists yet.
func (h *EventHandler) RSVPFromMail(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req struct {
		UID      string `json:"uid"`
		PartStat string `json:"partstat"`
		Summary  string `json:"summary"`
		StartAt  string `json:"start_at"`
		EndAt    string `json:"end_at"`
		Location string `json:"location"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	partStat := strings.ToUpper(strings.TrimSpace(req.PartStat))
	validStates := map[string]bool{"ACCEPTED": true, "DECLINED": true, "TENTATIVE": true, "NEEDS-ACTION": true}
	if !validStates[partStat] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid partstat"})
	}
	uid := strings.TrimSpace(req.UID)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "uid is required"})
	}

	userEmail := getSessionUsername(c)
	event, err := h.eventRepo.GetByUID(userID, uid)
	if err == gorm.ErrRecordNotFound {
		// Auto-create a stub event from the invitation so the user has it in their calendar.
		defaultCal, cerr := h.calRepo.EnsureDefault(userID)
		if cerr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": cerr.Error()})
		}
		startAt, _ := parseTime(req.StartAt)
		endAt, _ := parseTime(req.EndAt)
		if startAt.IsZero() {
			startAt = time.Now()
		}
		if endAt.IsZero() {
			endAt = startAt.Add(time.Hour)
		}
		summary := req.Summary
		if summary == "" {
			summary = "(Meeting invitation)"
		}
		event = &model.Event{
			CalendarID: defaultCal.ID,
			UserID:     userID,
			UID:        uid,
			Summary:    summary,
			Location:   req.Location,
			StartAt:    startAt,
			EndAt:      endAt,
			Status:     "CONFIRMED",
			Attendees: []model.EventAttendee{
				{Email: userEmail, PartStat: partStat, Role: "REQ-PARTICIPANT", RSVP: true},
			},
		}
		event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
		if cerr := h.eventRepo.Create(event); cerr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": cerr.Error()})
		}
		created, _ := h.eventRepo.Get(userID, event.ID)
		if created != nil {
			return c.JSON(http.StatusCreated, toEventResponse(*created))
		}
		return c.JSON(http.StatusCreated, toEventResponse(*event))
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Update or insert the attendee entry for this user.
	updated := false
	for i := range event.Attendees {
		if strings.EqualFold(event.Attendees[i].Email, userEmail) {
			event.Attendees[i].PartStat = partStat
			updated = true
			break
		}
	}
	if !updated {
		event.Attendees = append(event.Attendees, model.EventAttendee{
			EventID:  event.ID,
			Email:    userEmail,
			PartStat: partStat,
			Role:     "REQ-PARTICIPANT",
			RSVP:     true,
		})
	}
	event.Sequence++
	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := h.eventRepo.Update(event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	result, _ := h.eventRepo.Get(userID, event.ID)
	if result != nil {
		return c.JSON(http.StatusOK, toEventResponse(*result))
	}
	return c.JSON(http.StatusOK, toEventResponse(*event))
}

// RSVP handles POST /api/v1/events/:id/rsvp.
// Updates the authenticated user's attendee participation status (ACCEPTED, DECLINED, TENTATIVE).
func (h *EventHandler) RSVP(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req struct {
		PartStat string `json:"partstat"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	validStates := map[string]bool{"ACCEPTED": true, "DECLINED": true, "TENTATIVE": true, "NEEDS-ACTION": true}
	partStat := strings.ToUpper(strings.TrimSpace(req.PartStat))
	if !validStates[partStat] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "partstat must be ACCEPTED, DECLINED, TENTATIVE, or NEEDS-ACTION"})
	}
	event, err := h.eventRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	userEmail := getSessionUsername(c)
	updated := false
	for i := range event.Attendees {
		if userEmail != "" && event.Attendees[i].Email == userEmail {
			event.Attendees[i].PartStat = partStat
			updated = true
			break
		}
	}
	if !updated && len(event.Attendees) > 0 {
		event.Attendees[0].PartStat = partStat
	}
	event.Sequence++
	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := h.eventRepo.Update(event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	result, err := h.eventRepo.Get(userID, event.ID)
	if err != nil {
		return c.JSON(http.StatusOK, toEventResponse(*event))
	}
	return c.JSON(http.StatusOK, toEventResponse(*result))
}

// Move handles POST /api/v1/events/:id/move.
// Updates start_at, end_at, and optionally calendar_id (drag-resize / move).
func (h *EventHandler) Move(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	event, err := h.eventRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	var req eventMoveRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	startAt, err := parseTime(req.StartAt)
	if err != nil || startAt.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start_at"})
	}
	endAt, err := parseTime(req.EndAt)
	if err != nil || endAt.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end_at"})
	}
	if !endAt.After(startAt) && !event.IsAllDay {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "end_at must be after start_at"})
	}
	if req.CalendarID != nil {
		if _, err := h.calRepo.Get(userID, *req.CalendarID); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid calendar_id"})
		}
		event.CalendarID = *req.CalendarID
	}
	event.StartAt = startAt
	event.EndAt = endAt
	event.Sequence++
	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := h.eventRepo.Update(event); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	updated, err := h.eventRepo.Get(userID, event.ID)
	if err != nil {
		return c.JSON(http.StatusOK, toEventResponse(*event))
	}
	return c.JSON(http.StatusOK, toEventResponse(*updated))
}
