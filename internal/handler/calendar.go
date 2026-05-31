package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalendarHandler handles CRUD operations for user calendars and ICS import/export.
type CalendarHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	eventRepo *repository.EventRepo
}

// calendarRequest is the JSON body accepted by calendar create/update endpoints.
type calendarRequest struct {
	Name              string `json:"name"`
	Color             string `json:"color"`
	IncludeInFreeBusy *bool  `json:"include_in_free_busy"`
	SortOrder         *int   `json:"sort_order"`
}

// calendarResponse is the JSON shape returned by calendar endpoints.
type calendarResponse struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	Color             string `json:"color"`
	IsDefault         bool   `json:"is_default"`
	IsActive          bool   `json:"is_active"`
	IncludeInFreeBusy bool   `json:"include_in_free_busy"`
	SortOrder         int    `json:"sort_order"`
}

// calendarListResponse wraps the list of calendars returned by GET /calendar.
type calendarListResponse struct {
	Calendars []calendarResponse `json:"calendars"`
}

// calendarActivationRequest toggles is_active for multiple calendars at once.
type calendarActivationRequest struct {
	IDs    []uint `json:"ids"`
	Active bool   `json:"active"`
}

// toCalendarResponse maps a model.Calendar to the API response shape,
// applying the default color when none is stored.
func toCalendarResponse(cal model.Calendar) calendarResponse {
	color := cal.Color
	if color == "" {
		color = "#3788d8"
	}
	return calendarResponse{
		ID:                cal.ID,
		Name:              cal.Name,
		Color:             color,
		IsDefault:         cal.IsDefault,
		IsActive:          cal.IsActive,
		IncludeInFreeBusy: cal.IncludeInFreeBusy,
		SortOrder:         cal.SortOrder,
	}
}

// List godoc
// @Summary      List calendars
// @Description  Returns all calendars for the authenticated user. Creates a default "Personal" calendar on first access.
// @Tags         calendar
// @Produce      json
// @Success      200  {object}  calendarListResponse "calendars list"
// @Failure      500  {object}  map[string]string    "database error"
// @Security     CookieAuth
// @Router       /calendar [get]
func (h *CalendarHandler) List(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	if _, err := h.calRepo.EnsureDefault(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	calendars, err := h.calRepo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]calendarResponse, len(calendars))
	for i, cal := range calendars {
		out[i] = toCalendarResponse(cal)
	}
	return c.JSON(http.StatusOK, calendarListResponse{Calendars: out})
}

// Create godoc
// @Summary      Create calendar
// @Description  Creates a new calendar. Color defaults to #3788d8 when omitted.
// @Tags         calendar
// @Accept       json
// @Produce      json
// @Param        body  body      calendarRequest   true  "Calendar data"
// @Success      201  {object}  calendarResponse  "created calendar"
// @Failure      400  {object}  map[string]string "name required or invalid body"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /calendar [post]
func (h *CalendarHandler) Create(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req calendarRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#3788d8"
	}
	cal := model.Calendar{
		UserID:            userID,
		Name:              name,
		Color:             color,
		IsActive:          true,
		IncludeInFreeBusy: true,
	}
	if req.IncludeInFreeBusy != nil {
		cal.IncludeInFreeBusy = *req.IncludeInFreeBusy
	}
	if req.SortOrder != nil {
		cal.SortOrder = *req.SortOrder
	}
	if err := h.calRepo.Create(&cal); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, toCalendarResponse(cal))
}

// Update godoc
// @Summary      Update calendar
// @Description  Updates an existing calendar. Only provided fields are changed.
// @Tags         calendar
// @Accept       json
// @Produce      json
// @Param        id    path      int               true  "Calendar ID"
// @Param        body  body      calendarRequest   true  "Calendar data"
// @Success      200  {object}  calendarResponse  "updated calendar"
// @Failure      400  {object}  map[string]string "invalid id or body"
// @Failure      404  {object}  map[string]string "not found"
// @Security     CookieAuth
// @Router       /calendar/{id} [put]
func (h *CalendarHandler) Update(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	cal, err := h.calRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	var req calendarRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		cal.Name = name
	}
	if color := strings.TrimSpace(req.Color); color != "" {
		cal.Color = color
	}
	if req.IncludeInFreeBusy != nil {
		cal.IncludeInFreeBusy = *req.IncludeInFreeBusy
	}
	if req.SortOrder != nil {
		cal.SortOrder = *req.SortOrder
	}
	if err := h.calRepo.Update(cal); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toCalendarResponse(*cal))
}

// Delete godoc
// @Summary      Delete calendar
// @Description  Removes a calendar and all its events. The default calendar cannot be deleted.
// @Tags         calendar
// @Produce      json
// @Param        id  path  int  true  "Calendar ID"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid id or cannot delete default"
// @Failure      404  {object}  map[string]string "not found"
// @Security     CookieAuth
// @Router       /calendar/{id} [delete]
func (h *CalendarHandler) Delete(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	cal, err := h.calRepo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	if cal.IsDefault {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete default calendar"})
	}
	if err := h.calRepo.Delete(userID, uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// SetActivation godoc
// @Summary      Toggle calendar visibility
// @Description  Sets the is_active flag for multiple calendars at once. Hidden calendars are excluded from the event view.
// @Tags         calendar
// @Accept       json
// @Produce      json
// @Param        body  body      calendarActivationRequest  true  "IDs and active flag"
// @Success      200  {object}  map[string]string           "status ok"
// @Failure      400  {object}  map[string]string           "ids required or invalid body"
// @Security     CookieAuth
// @Router       /calendar/activation [post]
func (h *CalendarHandler) SetActivation(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req calendarActivationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ids are required"})
	}
	if err := h.calRepo.SetActive(userID, req.IDs, req.Active); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Export godoc
// @Summary      Export calendar as ICS
// @Description  Streams all events in the calendar as a downloadable calendar.ics file.
// @Tags         calendar
// @Produce      text/calendar
// @Param        id  path  int  true  "Calendar ID"
// @Success      200  {file}    binary            "ICS file download"
// @Failure      400  {object}  map[string]string "invalid id"
// @Failure      404  {object}  map[string]string "not found"
// @Security     CookieAuth
// @Router       /calendar/{id}/export [get]
func (h *CalendarHandler) Export(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if _, err := h.calRepo.Get(userID, uint(id)); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	events, err := h.eventRepo.ListByCalendar(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	ics := calpkg.BuildCalendarExport(events)
	filename := "calendar.ics"
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	_, err = c.Response().Write([]byte(ics))
	return err
}

// Import godoc
// @Summary      Import calendar from ICS
// @Description  Accepts a multipart ICS file upload and upserts events by UID. Returns imported/updated/skipped counts.
// @Tags         calendar
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      int   true  "Calendar ID"
// @Param        file  formData  file  true  "ICS file"
// @Success      200  {object}  map[string]any    "imported, updated, skipped counts"
// @Failure      400  {object}  map[string]string "no file, parse error, or invalid id"
// @Failure      404  {object}  map[string]string "calendar not found"
// @Security     CookieAuth
// @Router       /calendar/{id}/import [post]
func (h *CalendarHandler) Import(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if _, err := h.calRepo.Get(userID, uint(id)); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no file uploaded"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
	}

	parsed, err := calpkg.ParseICalImport(content)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	imported, updated, skipped := 0, 0, 0
	for _, item := range parsed {
		existing, err := h.eventRepo.GetByUID(userID, item.UID)
		if err == nil {
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
				skipped++
				continue
			}
			updated++
			continue
		}
		if err != gorm.ErrRecordNotFound {
			skipped++
			continue
		}

		event := model.Event{
			CalendarID:  uint(id),
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
			skipped++
			continue
		}
		imported++
	}

	return c.JSON(http.StatusOK, map[string]any{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"total":    len(parsed),
	})
}
