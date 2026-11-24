package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
	"go-cubemail/internal/theme"
)

// ThemesHandler serves the public theme registry.
type ThemesHandler struct {
	cfg *config.Config
}

// themesResponse is the JSON shape returned by GET /api/v1/themes.
type themesResponse struct {
	// Default is the server's default theme id (config key ui.theme).
	Default string        `json:"default"`
	Themes  []theme.Theme `json:"themes"`
}

// List godoc
// @Summary      List UI themes
// @Description  Returns every registered theme in display order and the server's default theme id. Public: the login screen needs it before a session exists.
// @Tags         themes
// @Produce      json
// @Success      200  {object}  themesResponse "Server default theme and the theme registry"
// @Router       /themes [get]
func (h *ThemesHandler) List(c *echo.Context) error {
	// config.Load already validates ui.theme; ByID keeps the answer a registered id for any other Config.
	def, _ := theme.ByID(h.cfg.UI.Theme)
	return c.JSON(http.StatusOK, themesResponse{Default: def.ID, Themes: theme.List()})
}
