package handler

import (
	"net/http"
	"strconv"

	"go-cubemail/internal/config"
	"github.com/labstack/echo/v5"
)

type ContactsHandler struct {
	cfg *config.Config
}

func (h *ContactsHandler) Index(c *echo.Context) error {
	return c.Render(http.StatusOK, "contacts/index.html", nil)
}

func (h *ContactsHandler) New(c *echo.Context) error {
	return c.Render(http.StatusOK, "contacts/edit.html", map[string]interface{}{
		"Mode": "new",
	})
}

func (h *ContactsHandler) Create(c *echo.Context) error {
	return c.Redirect(http.StatusSeeOther, "/contacts")
}

func (h *ContactsHandler) Edit(c *echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	return c.Render(http.StatusOK, "contacts/edit.html", map[string]interface{}{
		"Mode": "edit",
		"ID":   id,
	})
}

func (h *ContactsHandler) Update(c *echo.Context) error {
	return c.Redirect(http.StatusSeeOther, "/contacts")
}

func (h *ContactsHandler) Delete(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
