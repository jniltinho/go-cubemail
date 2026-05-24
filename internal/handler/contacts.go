package handler

import (
	"net/http"
	"strconv"
	"strings"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

type ContactsHandler struct {
	cfg  *config.Config
	repo *repository.ContactRepo
	db   *gorm.DB
}

type contactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`
}

type contactResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`
}

func (h *ContactsHandler) getUserID(c *echo.Context) (uint, error) {
	s := c.Get("imap_session").(*session.IMAPSession)
	var user model.User
	err := h.db.Where("imap_user = ?", s.Username).
		FirstOrCreate(&user, model.User{ImapUser: s.Username}).Error
	return user.ID, err
}

func toResponse(c model.Contact) contactResponse {
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	return contactResponse{
		ID:      c.ID,
		Name:    name,
		Email:   c.Email,
		Title:   c.Title,
		Company: c.Company,
		Phone:   c.Phone,
		Notes:   c.Notes,
	}
}

func splitName(full string) (first, last string) {
	full = strings.TrimSpace(full)
	if f, l, ok := strings.Cut(full, " "); ok {
		return f, strings.TrimSpace(l)
	}
	return full, ""
}

func (h *ContactsHandler) Index(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	contacts, err := h.repo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]contactResponse, len(contacts))
	for i, ct := range contacts {
		out[i] = toResponse(ct)
	}
	return c.JSON(http.StatusOK, out)
}

func (h *ContactsHandler) Create(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var req contactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if req.Name == "" || req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name and email are required"})
	}
	first, last := splitName(req.Name)
	ct := model.Contact{
		UserID:    userID,
		FirstName: first,
		LastName:  last,
		Email:     req.Email,
		Title:     req.Title,
		Company:   req.Company,
		Phone:     req.Phone,
		Notes:     req.Notes,
	}
	if err := h.repo.Create(&ct); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, toResponse(ct))
}

func (h *ContactsHandler) Update(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, _ := strconv.Atoi(c.Param("id"))
	ct, err := h.repo.Get(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	var req contactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	first, last := splitName(req.Name)
	ct.FirstName = first
	ct.LastName = last
	ct.Email = req.Email
	ct.Title = req.Title
	ct.Company = req.Company
	ct.Phone = req.Phone
	ct.Notes = req.Notes
	if err := h.repo.Update(ct); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toResponse(*ct))
}

func (h *ContactsHandler) Delete(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(userID, uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
