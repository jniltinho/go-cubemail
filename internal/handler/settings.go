package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"go-cubemail/internal/theme"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// SettingsHandler handles user preferences and sender identity management.
type SettingsHandler struct {
	cfg          *config.Config
	db           *gorm.DB
	settingsRepo *repository.SettingsRepo
	identityRepo *repository.IdentityRepo
}

// settingsRequest is the JSON body for PUT /api/v1/settings.
type settingsRequest struct {
	RowsPerPage  *int    `json:"rows_per_page"`
	Timezone     *string `json:"timezone"`
	ComposeHTML  *bool   `json:"compose_html"`
	DateFormat   *string `json:"date_format"`
	SignaturePos *string `json:"signature_pos"`
	OOFEnabled   *bool   `json:"oof_enabled"`
	OOFMessage   *string `json:"oof_message"`
	// Theme is a registered theme id, or "" to follow the server default.
	Theme *string `json:"theme"`
}

// settingsResponse is the JSON shape returned by GET /api/v1/settings.
type settingsResponse struct {
	RowsPerPage  int    `json:"rows_per_page"`
	Timezone     string `json:"timezone"`
	ComposeHTML  bool   `json:"compose_html"`
	DateFormat   string `json:"date_format"`
	SignaturePos string `json:"signature_pos"`
	OOFEnabled   bool   `json:"oof_enabled"`
	OOFMessage   string `json:"oof_message"`
	// Theme is a registered theme id, or "" when the user follows the server default.
	Theme string `json:"theme"`
}

func toSettingsResponse(s *model.UserSettings) settingsResponse {
	// A stored id that is no longer registered reads as "follow the server default".
	themeID := s.Theme
	if !theme.Valid(themeID) {
		themeID = ""
	}
	return settingsResponse{
		RowsPerPage:  s.RowsPerPage,
		Timezone:     s.Timezone,
		ComposeHTML:  s.ComposeHTML,
		DateFormat:   s.DateFormat,
		SignaturePos: s.SignaturePos,
		OOFEnabled:   s.OOFEnabled,
		OOFMessage:   s.OOFMessage,
		Theme:        themeID,
	}
}

// identityRequest is the JSON body for POST/PUT identity endpoints.
type identityRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	ReplyTo     string `json:"reply_to"`
	Signature   string `json:"signature"`
	IsDefault   bool   `json:"is_default"`
}

// identityResponse is the JSON shape for identity API responses.
type identityResponse struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	ReplyTo     string `json:"reply_to"`
	Signature   string `json:"signature"`
	IsDefault   bool   `json:"is_default"`
}

func toIdentityResponse(id model.Identity) identityResponse {
	return identityResponse{
		ID:          id.ID,
		DisplayName: id.DisplayName,
		Email:       id.Email,
		ReplyTo:     id.ReplyTo,
		Signature:   id.Signature,
		IsDefault:   id.IsDefault,
	}
}

// GetSettings godoc
// @Summary      Get user preferences
// @Description  Returns the logged-in user's complete UI settings and preferences (e.g. theme, timezone, signatures, out of office status).
// @Tags         settings
// @Produce      json
// @Success      200  {object}  settingsResponse "Success response containing settings details"
// @Failure      500  {object}  map[string]string "User lookup or database retrieval failure"
// @Security     CookieAuth
// @Router       /settings [get]
func (h *SettingsHandler) GetSettings(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	s, err := h.settingsRepo.Get(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toSettingsResponse(s))
}

// SaveSettings godoc
// @Summary      Save user preferences
// @Description  Updates the logged-in user's settings. Only provided fields are changed. `theme` must be a registered theme id (see GET /themes) or an empty string to follow the server default.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body      settingsRequest   true  "Settings fields to update"
// @Success      200  {object}  settingsResponse  "updated settings"
// @Failure      400  {object}  map[string]string "invalid body or unknown theme"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /settings [put]
func (h *SettingsHandler) SaveSettings(c *echo.Context) error {
	// Bind and validate before touching the database: loading the settings creates the row.
	var req settingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if req.Theme != nil && *req.Theme != "" && !theme.Valid(*req.Theme) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown theme %q", *req.Theme)})
	}
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	s, err := h.settingsRepo.Get(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if req.RowsPerPage != nil && *req.RowsPerPage > 0 {
		s.RowsPerPage = *req.RowsPerPage
	}
	if req.Timezone != nil {
		s.Timezone = strings.TrimSpace(*req.Timezone)
	}
	if req.ComposeHTML != nil {
		s.ComposeHTML = *req.ComposeHTML
	}
	if req.DateFormat != nil {
		s.DateFormat = strings.TrimSpace(*req.DateFormat)
	}
	if req.SignaturePos != nil {
		s.SignaturePos = strings.TrimSpace(*req.SignaturePos)
	}
	if req.OOFEnabled != nil {
		s.OOFEnabled = *req.OOFEnabled
	}
	if req.OOFMessage != nil {
		s.OOFMessage = *req.OOFMessage
	}
	if req.Theme != nil {
		s.Theme = *req.Theme
	}
	if err := h.settingsRepo.Save(s); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toSettingsResponse(s))
}

// ListIdentities godoc
// @Summary      List sender identities
// @Description  Returns all From: identities for the user. Seeds a default identity from the IMAP username on first access.
// @Tags         settings
// @Produce      json
// @Success      200  {object}  map[string]any    "identities list"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /identities [get]
func (h *SettingsHandler) ListIdentities(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	ids, err := h.identityRepo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if len(ids) == 0 {
		// Seed a default identity from the IMAP username.
		username := getSessionUsername(c)
		seed := model.Identity{UserID: userID, Email: username, DisplayName: "", IsDefault: true}
		if err := h.identityRepo.Create(&seed); err == nil {
			ids = []model.Identity{seed}
		}
	}
	out := make([]identityResponse, len(ids))
	for i, id := range ids {
		out[i] = toIdentityResponse(id)
	}
	return c.JSON(http.StatusOK, map[string]any{"identities": out})
}

// CreateIdentity godoc
// @Summary      Create sender identity
// @Description  Adds a new From: identity (display name, email, reply-to, signature).
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body      identityRequest   true  "Identity data"
// @Success      201  {object}  identityResponse  "created identity"
// @Failure      400  {object}  map[string]string "email required or invalid body"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /identities [post]
func (h *SettingsHandler) CreateIdentity(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req identityRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}
	if req.IsDefault {
		h.db.Model(&model.Identity{}).Where("user_id = ?", userID).Update("is_default", false)
	}
	id := model.Identity{
		UserID:      userID,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Email:       email,
		ReplyTo:     strings.TrimSpace(req.ReplyTo),
		Signature:   req.Signature,
		IsDefault:   req.IsDefault,
	}
	if err := h.identityRepo.Create(&id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, toIdentityResponse(id))
}

// UpdateIdentity godoc
// @Summary      Update sender identity
// @Description  Replaces all fields of the specified sender identity.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        id    path      int               true  "Identity ID"
// @Param        body  body      identityRequest   true  "Identity data"
// @Success      200  {object}  identityResponse  "updated identity"
// @Failure      400  {object}  map[string]string "invalid id, body, or missing email"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /identities/{id} [put]
func (h *SettingsHandler) UpdateIdentity(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	idNum, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req identityRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}
	if req.IsDefault {
		h.db.Model(&model.Identity{}).Where("user_id = ?", userID).Update("is_default", false)
	}
	id := model.Identity{
		ID:          uint(idNum),
		UserID:      userID,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Email:       email,
		ReplyTo:     strings.TrimSpace(req.ReplyTo),
		Signature:   req.Signature,
		IsDefault:   req.IsDefault,
	}
	if err := h.identityRepo.Update(&id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toIdentityResponse(id))
}

// DeleteIdentity godoc
// @Summary      Delete sender identity
// @Description  Permanently removes the specified sender identity.
// @Tags         settings
// @Produce      json
// @Param        id  path  int  true  "Identity ID"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid id"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /identities/{id} [delete]
func (h *SettingsHandler) DeleteIdentity(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	idNum, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.identityRepo.Delete(userID, uint(idNum)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// SetDefaultIdentity godoc
// @Summary      Set default sender identity
// @Description  Marks the specified identity as the default From: address, clearing the flag on all others.
// @Tags         settings
// @Produce      json
// @Param        id  path  int  true  "Identity ID"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid id"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /identities/{id}/default [post]
func (h *SettingsHandler) SetDefaultIdentity(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	idNum, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	h.db.Model(&model.Identity{}).Where("user_id = ?", userID).Update("is_default", false)
	if err := h.db.Model(&model.Identity{}).
		Where("id = ? AND user_id = ?", idNum, userID).
		Update("is_default", true).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
