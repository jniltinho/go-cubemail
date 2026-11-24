package handler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"go-cubemail/internal/theme"
)

// The route is registered with no auth middleware at all, as on the real server.
func TestThemesListWithoutSession(t *testing.T) {
	env := newTestEnv(t)
	env.h.Themes.cfg.UI.Theme = "outlook"
	env.echo.GET("/api/v1/themes", env.h.Themes.List)

	rec := env.do(http.MethodGet, "/api/v1/themes", "", nil)
	assertStatus(t, rec, http.StatusOK)

	// The registry is generated from the theme stylesheets, so the list is compared with it, not spelled out.
	var got struct {
		Default string        `json:"default"`
		Themes  []theme.Theme `json:"themes"`
	}
	if err := json.Unmarshal(bodyBytes(rec), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Default != "outlook" {
		t.Errorf("default = %q, want outlook", got.Default)
	}
	if !reflect.DeepEqual(got.Themes, theme.List()) {
		t.Errorf("themes = %+v\nwant     %+v", got.Themes, theme.List())
	}
}

// A Config that did not go through config.Load still yields a registered default.
func TestThemesListDefaultIsAlwaysRegistered(t *testing.T) {
	env := newTestEnv(t)
	env.h.Themes.cfg.UI.Theme = "nope"
	env.echo.GET("/api/v1/themes", env.h.Themes.List)

	rec := env.do(http.MethodGet, "/api/v1/themes", "", nil)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec.Body.String(), `"default":"larry"`, "fallback default")
}
