package handler

import (
	"io"
	"net/http"
	"net/url"
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

// CalendarSubscriptionHandler manages remote .ics calendar subscriptions.
type CalendarSubscriptionHandler struct {
	cfg     *config.Config
	db      *gorm.DB
	calRepo *repository.CalendarRepo
	subRepo *repository.CalendarSubscriptionRepo
	evtRepo *repository.EventRepo
}

type subscriptionRequest struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	RefreshMins int    `json:"refresh_mins"`
}

type subscriptionResponse struct {
	ID           uint    `json:"id"`
	CalendarID   uint    `json:"calendar_id"`
	URL          string  `json:"url"`
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	RefreshMins  int     `json:"refresh_mins"`
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
	LastError    string  `json:"last_error,omitempty"`
}

func toSubResponse(s model.CalendarSubscription) subscriptionResponse {
	r := subscriptionResponse{
		ID: s.ID, CalendarID: s.CalendarID, URL: s.URL,
		Name: s.Name, Color: s.Color, RefreshMins: s.RefreshMins,
		LastError: s.LastError,
	}
	if s.LastSyncedAt != nil {
		t := s.LastSyncedAt.UTC().Format(time.RFC3339)
		r.LastSyncedAt = &t
	}
	return r
}

// List handles GET /api/v1/calendar/subscriptions.
func (h *CalendarSubscriptionHandler) List(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	subs, err := h.subRepo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]subscriptionResponse, len(subs))
	for i, s := range subs {
		out[i] = toSubResponse(s)
	}
	return c.JSON(http.StatusOK, map[string]any{"subscriptions": out})
}

// Create handles POST /api/v1/calendar/subscriptions.
// Fetches the remote URL immediately to validate it, then stores the subscription.
func (h *CalendarSubscriptionHandler) Create(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req subscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid url"})
	}

	defaultCal, err := h.calRepo.EnsureDefault(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	color := req.Color
	if color == "" {
		color = "#999999"
	}
	refresh := req.RefreshMins
	if refresh <= 0 {
		refresh = 60
	}
	name := req.Name
	if name == "" {
		name = rawURL
	}

	sub := model.CalendarSubscription{
		UserID:      userID,
		CalendarID:  defaultCal.ID,
		URL:         rawURL,
		Name:        name,
		Color:       color,
		RefreshMins: refresh,
	}
	if err := h.subRepo.Create(&sub); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch immediately in background — don't block the response.
	go FetchAndImportSubscription(h.subRepo, h.evtRepo, sub)

	return c.JSON(http.StatusCreated, toSubResponse(sub))
}

// Delete handles DELETE /api/v1/calendar/subscriptions/:id.
func (h *CalendarSubscriptionHandler) Delete(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.subRepo.Delete(userID, uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Refresh handles POST /api/v1/calendar/subscriptions/:id/refresh.
// Triggers an immediate re-fetch of the subscription.
func (h *CalendarSubscriptionHandler) Refresh(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	subs, err := h.subRepo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	for _, sub := range subs {
		if sub.ID == uint(id) {
			go FetchAndImportSubscription(h.subRepo, h.evtRepo, sub)
			return c.JSON(http.StatusOK, map[string]string{"status": "refreshing"})
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
}

// FetchAndImportSubscription downloads a remote .ics URL and upserts its events.
// Exported so the background worker in server.go can call it directly.
func FetchAndImportSubscription(subRepo *repository.CalendarSubscriptionRepo, evtRepo *repository.EventRepo, sub model.CalendarSubscription) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sub.URL)
	if err != nil {
		_ = subRepo.UpdateSyncResult(sub.ID, err.Error())
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5 MB limit
	if err != nil {
		_ = subRepo.UpdateSyncResult(sub.ID, err.Error())
		return
	}
	events, err := calpkg.ParseICalImport(data)
	if err != nil {
		_ = subRepo.UpdateSyncResult(sub.ID, err.Error())
		return
	}
	for _, item := range events {
		existing, err := evtRepo.GetByUID(sub.UserID, item.UID)
		if err == nil {
			existing.Summary = item.Summary
			existing.Description = item.Description
			existing.Location = item.Location
			existing.StartAt = item.StartAt
			existing.EndAt = item.EndAt
			existing.IsAllDay = item.IsAllDay
			existing.Status = item.Status
			existing.RRule = item.RRule
			existing.Sequence++
			existing.ICalContent = calpkg.BuildICalContent(existing, nil)
			_ = evtRepo.Update(existing)
			continue
		}
		evt := model.Event{
			CalendarID:  sub.CalendarID,
			UserID:      sub.UserID,
			UID:         item.UID,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			StartAt:     item.StartAt,
			EndAt:       item.EndAt,
			IsAllDay:    item.IsAllDay,
			Status:      item.Status,
			RRule:       item.RRule,
		}
		if evt.Status == "" {
			evt.Status = "CONFIRMED"
		}
		evt.ICalContent = calpkg.BuildICalContent(&evt, nil)
		_ = evtRepo.Create(&evt)
	}
	_ = subRepo.UpdateSyncResult(sub.ID, "")
}
