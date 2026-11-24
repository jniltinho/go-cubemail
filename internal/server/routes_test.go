package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
	"go-cubemail/internal/handler"
	appMiddleware "go-cubemail/internal/server/middleware"
)

// The theme list is public: the real route table, with the real RequireAuth,
// must answer it without a session cookie while still guarding the settings.
func TestThemesRouteIsPublic(t *testing.T) {
	cfg := &config.Config{}
	cfg.Session.Name = "gorc_session"
	cfg.UI.Theme = "larry"

	e := echo.New()
	passthrough := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	// No request below reaches the database, so the handlers get none.
	registerAPIRoutes(e.Group("/api/v1"), handler.New(cfg, nil), appMiddleware.RequireAuth(cfg.Session.Name), passthrough)

	for path, want := range map[string]int{
		"/api/v1/themes":   http.StatusOK,
		"/api/v1/settings": http.StatusUnauthorized,
	} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s without a session = %d, want %d\nbody: %s", path, rec.Code, want, rec.Body.String())
		}
	}
}
