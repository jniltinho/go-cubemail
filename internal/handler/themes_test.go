package handler

import (
	"net/http"
	"testing"
)

// The route is registered with no auth middleware at all, as on the real server.
func TestThemesListWithoutSession(t *testing.T) {
	env := newTestEnv(t)
	env.h.Themes.cfg.UI.Theme = "outlook"
	env.echo.GET("/api/v1/themes", env.h.Themes.List)

	rec := env.do(http.MethodGet, "/api/v1/themes", "", nil)
	assertStatus(t, rec, http.StatusOK)

	const want = `{"default":"outlook","themes":[` +
		`{"id":"larry","label":"Larry","scheme":"light"},` +
		`{"id":"outlook","label":"Outlook","scheme":"light"},` +
		`{"id":"outlook-dark","label":"Outlook (dark)","scheme":"dark"},` +
		`{"id":"apple-mail","label":"Apple Mail","scheme":"light"},` +
		`{"id":"apple-mail-dark","label":"Apple Mail (dark)","scheme":"dark"},` +
		`{"id":"breeze","label":"Breeze","scheme":"light"},` +
		`{"id":"breeze-dark","label":"Breeze (dark)","scheme":"dark"},` +
		`{"id":"graphite","label":"Graphite (dark)","scheme":"dark"},` +
		`{"id":"high-contrast","label":"High contrast (dark)","scheme":"dark"}]}`
	if got := string(bodyBytes(rec)); got != want {
		t.Errorf("body = %s\nwant   %s", got, want)
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
