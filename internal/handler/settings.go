package handler

import (
	"net/http"

	"go-cubemail/internal/config"
	"github.com/labstack/echo/v5"
)

type SettingsHandler struct {
	cfg *config.Config
}

func (h *SettingsHandler) Index(c *echo.Context) error {
	return c.Render(http.StatusOK, "settings/index.html", nil)
}

func (h *SettingsHandler) Save(c *echo.Context) error {
	return c.Redirect(http.StatusSeeOther, "/settings")
}
