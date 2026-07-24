package handler

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go-cubemail/internal/config"
	"go-cubemail/internal/contacts"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// ContactsHandler handles CRUD operations for user contacts and CSV/vCard import-export.
type ContactsHandler struct {
	cfg  *config.Config
	repo *repository.ContactRepo
	db   *gorm.DB
}

// contactRequest is the JSON body for create/update contact endpoints.
type contactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`
}

// contactResponse is the JSON shape returned by all contact endpoints.
type contactResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`
}

// getUserID resolves the database user ID from the IMAP session username,
// creating a User record on first login if one does not yet exist.
func (h *ContactsHandler) getUserID(c *echo.Context) (uint, error) {
	s := c.Get("imap_session").(*session.IMAPSession)
	var user model.User
	err := h.db.Where("imap_user = ?", s.Username).
		FirstOrCreate(&user, model.User{ImapUser: s.Username}).Error
	return user.ID, err
}

// toResponse converts a model.Contact to the API response shape.
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

// splitName splits a full name string on the first space.
// The entire string is returned as first if no space is found.
func splitName(full string) (first, last string) {
	full = strings.TrimSpace(full)
	if f, l, ok := strings.Cut(full, " "); ok {
		return f, strings.TrimSpace(l)
	}
	return full, ""
}

// Index godoc
// @Summary      List contacts
// @Description  Returns all contacts for the authenticated user, ordered by name.
// @Tags         contacts
// @Produce      json
// @Success      200  {array}   contactResponse   "list of contacts"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /contacts [get]
func (h *ContactsHandler) Index(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	list, err := h.repo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]contactResponse, len(list))
	for i, ct := range list {
		out[i] = toResponse(ct)
	}
	return c.JSON(http.StatusOK, out)
}

// Create godoc
// @Summary      Create contact
// @Description  Adds a new contact. Both name and email are required.
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        body  body      contactRequest    true  "Contact data"
// @Success      201  {object}  contactResponse   "created contact"
// @Failure      400  {object}  map[string]string "name and email required"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /contacts [post]
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

// Update godoc
// @Summary      Update contact
// @Description  Replaces all fields of an existing contact identified by :id.
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        id    path      int              true  "Contact ID"
// @Param        body  body      contactRequest   true  "Contact data"
// @Success      200  {object}  contactResponse  "updated contact"
// @Failure      400  {object}  map[string]string "invalid body"
// @Failure      404  {object}  map[string]string "not found"
// @Security     CookieAuth
// @Router       /contacts/{id} [put]
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

// Delete godoc
// @Summary      Delete contact
// @Description  Permanently removes a contact by ID.
// @Tags         contacts
// @Produce      json
// @Param        id  path  int  true  "Contact ID"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /contacts/{id} [delete]
func (h *ContactsHandler) Delete(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(userID, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Export godoc
// @Summary      Export contacts as CSV
// @Description  Streams all contacts as a downloadable contacts.csv file.
// @Tags         contacts
// @Produce      text/csv
// @Success      200  {file}    binary "CSV file download"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /contacts/export [get]
func (h *ContactsHandler) Export(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	list, err := h.repo.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"First Name", "Last Name", "Email", "Title", "Company", "Phone", "Notes"})
	for _, ct := range list {
		_ = w.Write([]string{ct.FirstName, ct.LastName, ct.Email, ct.Title, ct.Company, ct.Phone, ct.Notes})
	}
	w.Flush()

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"contacts.csv\"")
	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	_, err = c.Response().Write(buf.Bytes())
	return err
}

// Import godoc
// @Summary      Import contacts from CSV or VCF
// @Description  Parses an uploaded .csv or .vcf file and bulk-inserts contacts. Returns imported/total counts.
// @Tags         contacts
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "CSV or VCF file"
// @Success      200  {object}  map[string]any    "imported and total counts"
// @Failure      400  {object}  map[string]string "no file or parse error"
// @Failure      500  {object}  map[string]string "file read error"
// @Security     CookieAuth
// @Router       /contacts/import [post]
func (h *ContactsHandler) Import(c *echo.Context) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
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

	var parsed []model.Contact
	name := strings.ToLower(file.Filename)
	if strings.HasSuffix(name, ".vcf") || strings.HasSuffix(name, ".vcard") {
		parsed, err = parseVCard(content, userID)
	} else {
		parsed, err = parseCSV(content, userID)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	imported := 0
	for i := range parsed {
		if err := h.repo.Create(&parsed[i]); err == nil {
			imported++
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"imported": imported, "total": len(parsed)})
}

// parseCSV handles Gmail-style and Outlook-style CSV exports.
func parseCSV(content []byte, userID uint) ([]model.Contact, error) {
	r := csv.NewReader(bytes.NewReader(content))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	idx := make(map[string]int)
	for i, h := range records[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	col := func(row []string, keys ...string) string {
		for _, k := range keys {
			if i, ok := idx[k]; ok && i < len(row) {
				if v := strings.TrimSpace(row[i]); v != "" {
					return v
				}
			}
		}
		return ""
	}

	var list []model.Contact
	for _, row := range records[1:] {
		email := col(row, "email", "e-mail address", "email address", "e-mail 1 - value")
		if email == "" {
			continue
		}
		first := col(row, "first name", "given name", "firstname")
		last := col(row, "last name", "family name", "surname", "lastname")
		if first == "" && last == "" {
			first, last = splitName(col(row, "name", "full name"))
		}
		list = append(list, model.Contact{
			UserID:    userID,
			FirstName: first,
			LastName:  last,
			Email:     email,
			Title:     col(row, "title", "job title", "occupation"),
			Company:   col(row, "company", "organization", "org"),
			Phone:     col(row, "phone", "mobile", "tel", "telephone", "mobile phone", "home phone"),
			Notes:     col(row, "notes", "note", "description"),
		})
	}
	return list, nil
}

// parseVCard handles single and multi-contact .vcf files (vCard 2.1, 3.0, 4.0).
//
// Each card is kept as its own blob so an imported contact carries everything
// the source file held — addresses, photos, birthdays, extra numbers — even
// though only the indexed fields are shown in the UI. Parsing itself is
// delegated to internal/contacts, which is the same code path CardDAV PUTs use.
func parseVCard(content []byte, userID uint) ([]model.Contact, error) {
	var out []model.Contact
	for _, block := range splitVCards(string(content)) {
		parsed, err := contacts.Parse(block)
		if err != nil {
			continue
		}
		parsed.UserID = userID
		parsed.VCardContent = block
		out = append(out, *parsed)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no vCard entries found in file")
	}
	return out, nil
}

// splitVCards separates a multi-card .vcf file into individual documents,
// preserving each card's own line endings.
func splitVCards(content string) []string {
	var cards []string
	var current []string
	inCard := false
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case trimmed == "BEGIN:VCARD":
			inCard = true
			current = []string{"BEGIN:VCARD"}
		case trimmed == "END:VCARD":
			if inCard {
				current = append(current, "END:VCARD")
				cards = append(cards, strings.Join(current, "\r\n")+"\r\n")
			}
			inCard = false
			current = nil
		case inCard:
			current = append(current, strings.TrimRight(line, "\r"))
		}
	}
	return cards
}
