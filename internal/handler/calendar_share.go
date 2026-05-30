package handler

import (
	"net/http"
	"strconv"
	"strings"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalendarShareHandler manages sharing of calendars between users.
type CalendarShareHandler struct {
	cfg       *config.Config
	db        *gorm.DB
	calRepo   *repository.CalendarRepo
	shareRepo *repository.CalendarShareRepo
	userRepo  *userLookup
}

// userLookup resolves IMAP usernames to user IDs.
type userLookup struct{ db *gorm.DB }

func (u *userLookup) IDByEmail(email string) (uint, error) {
	var user model.User
	err := u.db.Where("imap_user = ?", email).First(&user).Error
	return user.ID, err
}

type shareRequest struct {
	Email    string `json:"email"`
	CanWrite bool   `json:"can_write"`
}

type shareResponse struct {
	ID           uint   `json:"id"`
	CalendarID   uint   `json:"calendar_id"`
	CalendarName string `json:"calendar_name"`
	SharedWith   string `json:"shared_with"`
	CanWrite     bool   `json:"can_write"`
}

// List handles GET /api/v1/calendar/:id/shares.
func (h *CalendarShareHandler) List(c *echo.Context) error {
	ownerID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	cal, err := h.calRepo.Get(ownerID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	shares, err := h.shareRepo.ListSharedBy(ownerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := []shareResponse{}
	for _, s := range shares {
		if s.CalendarID != cal.ID {
			continue
		}
		var grantee model.User
		h.db.First(&grantee, s.SharedWithID)
		out = append(out, shareResponse{
			ID: s.ID, CalendarID: s.CalendarID, CalendarName: cal.Name,
			SharedWith: grantee.ImapUser, CanWrite: s.CanWrite,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"shares": out})
}

// Create handles POST /api/v1/calendar/:id/shares.
func (h *CalendarShareHandler) Create(c *echo.Context) error {
	ownerID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if _, err := h.calRepo.Get(ownerID, uint(id)); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	var req shareRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}
	granteeID, err := h.userRepo.IDByEmail(email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user not found"})
	}
	if granteeID == ownerID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot share with yourself"})
	}
	share := model.CalendarShare{
		CalendarID:   uint(id),
		OwnerID:      ownerID,
		SharedWithID: granteeID,
		CanWrite:     req.CanWrite,
	}
	if err := h.shareRepo.Create(&share); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, shareResponse{
		ID: share.ID, CalendarID: share.CalendarID,
		SharedWith: email, CanWrite: share.CanWrite,
	})
}

// Delete handles DELETE /api/v1/calendar/:id/shares/:shareId.
func (h *CalendarShareHandler) Delete(c *echo.Context) error {
	ownerID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	shareID, err := strconv.ParseUint(c.Param("shareId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid shareId"})
	}
	if err := h.shareRepo.Delete(ownerID, uint(shareID)); err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
