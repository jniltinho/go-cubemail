package handler

// Tests for the per-user theme setting (GET/PUT /api/v1/settings).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/model"
	"go-cubemail/internal/session"
)

// settingsTestUserHeader names the IMAP user the stub session authenticates as.
const settingsTestUserHeader = "X-Test-User"

// newSettingsEnv registers the settings routes behind a stub of
// middleware.RequireAuth that takes the IMAP username from a request header.
func newSettingsEnv(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t)
	stubAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("imap_session", &session.IMAPSession{Username: c.Request().Header.Get(settingsTestUserHeader)})
			return next(c)
		}
	}
	api := env.echo.Group("/api/v1", stubAuth)
	api.GET("/settings", env.h.Settings.GetSettings)
	api.PUT("/settings", env.h.Settings.SaveSettings)
	return env
}

func (env *testEnv) getSettings(user string) *httptest.ResponseRecorder {
	env.t.Helper()
	return env.do(http.MethodGet, "/api/v1/settings", "", map[string]string{settingsTestUserHeader: user})
}

func (env *testEnv) putSettings(user, body string) *httptest.ResponseRecorder {
	env.t.Helper()
	return env.do(http.MethodPut, "/api/v1/settings", body, map[string]string{
		settingsTestUserHeader: user,
		"Content-Type":         "application/json",
	})
}

// settingsBody decodes a settings response, failing on a non-200 status.
func settingsBody(t *testing.T, rec *httptest.ResponseRecorder) settingsResponse {
	t.Helper()
	assertStatus(t, rec, http.StatusOK)
	var out settingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode settings response: %v\nbody: %s", err, rec.Body.String())
	}
	return out
}

// storedTheme reads the theme column directly, bypassing the handler.
func (env *testEnv) storedTheme(userID uint) string {
	env.t.Helper()
	var s model.UserSettings
	if err := env.db.First(&s, "user_id = ?", userID).Error; err != nil {
		env.t.Fatalf("load settings row: %v", err)
	}
	return s.Theme
}

func (env *testEnv) settingsRowCount() int64 {
	env.t.Helper()
	var n int64
	if err := env.db.Model(&model.UserSettings{}).Count(&n).Error; err != nil {
		env.t.Fatalf("count settings rows: %v", err)
	}
	return n
}

func TestSettingsThemeDefaultsToEmpty(t *testing.T) {
	env := newSettingsEnv(t)
	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "" {
		t.Errorf("theme of a user who never chose one = %q, want empty", got)
	}
	assertContains(t, env.getSettings(davTestUser).Body.String(), `"theme":""`, "theme key is always present")
}

func TestSettingsSaveTheme(t *testing.T) {
	env := newSettingsEnv(t)

	if got := settingsBody(t, env.putSettings(davTestUser, `{"theme":"apple-mail"}`)).Theme; got != "apple-mail" {
		t.Errorf("PUT response theme = %q, want apple-mail", got)
	}
	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "apple-mail" {
		t.Errorf("GET theme after save = %q, want apple-mail", got)
	}
}

// Setting and then clearing goes through SettingsRepo.Save both times: the
// empty string has to survive the upsert (design D6).
func TestSettingsClearTheme(t *testing.T) {
	env := newSettingsEnv(t)

	settingsBody(t, env.putSettings(davTestUser, `{"theme":"graphite"}`))
	if got := env.storedTheme(env.userID); got != "graphite" {
		t.Fatalf("stored theme = %q, want graphite", got)
	}

	if got := settingsBody(t, env.putSettings(davTestUser, `{"theme":""}`)).Theme; got != "" {
		t.Errorf("PUT response theme after clearing = %q, want empty", got)
	}
	if got := env.storedTheme(env.userID); got != "" {
		t.Errorf("stored theme after clearing = %q, want empty", got)
	}
	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "" {
		t.Errorf("GET theme after clearing = %q, want empty", got)
	}
}

func TestSettingsRepoSaveSetsAndClearsTheme(t *testing.T) {
	env := newSettingsEnv(t)
	repo := env.h.Settings.settingsRepo

	s, err := repo.Get(env.userID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	for _, want := range []string{"breeze-dark", ""} {
		s.Theme = want
		if err := repo.Save(s); err != nil {
			t.Fatalf("save theme %q: %v", want, err)
		}
		if got := env.storedTheme(env.userID); got != want {
			t.Errorf("stored theme = %q, want %q", got, want)
		}
	}
}

func TestSettingsUnknownThemeChangesNothing(t *testing.T) {
	env := newSettingsEnv(t)
	settingsBody(t, env.putSettings(davTestUser, `{"theme":"outlook","timezone":"Europe/Lisbon"}`))

	rec := env.putSettings(davTestUser, `{"theme":"nope","timezone":"Asia/Tokyo"}`)
	assertStatus(t, rec, http.StatusBadRequest)
	assertContains(t, rec.Body.String(), "nope", "error names the rejected value")

	got := settingsBody(t, env.getSettings(davTestUser))
	if got.Theme != "outlook" || got.Timezone != "Europe/Lisbon" {
		t.Errorf("settings after a rejected request = theme %q, timezone %q; want outlook, Europe/Lisbon",
			got.Theme, got.Timezone)
	}
}

func TestSettingsUnknownThemeCreatesNoRow(t *testing.T) {
	env := newSettingsEnv(t)
	if n := env.settingsRowCount(); n != 0 {
		t.Fatalf("settings rows before the request = %d, want 0", n)
	}

	// Once for the seeded user, once for a user the database has never seen.
	for _, user := range []string{davTestUser, "newcomer@example.com"} {
		rec := env.putSettings(user, `{"theme":"nope"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertContains(t, rec.Body.String(), "nope", "error names the rejected value")
	}
	if n := env.settingsRowCount(); n != 0 {
		t.Errorf("settings rows after rejected requests = %d, want 0", n)
	}
}

func TestSettingsRequestWithoutThemeKeepsIt(t *testing.T) {
	env := newSettingsEnv(t)
	settingsBody(t, env.putSettings(davTestUser, `{"theme":"breeze"}`))

	got := settingsBody(t, env.putSettings(davTestUser, `{"timezone":"Europe/Lisbon"}`))
	if got.Theme != "breeze" || got.Timezone != "Europe/Lisbon" {
		t.Errorf("PUT response = theme %q, timezone %q; want breeze, Europe/Lisbon", got.Theme, got.Timezone)
	}
	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "breeze" {
		t.Errorf("GET theme = %q, want breeze", got)
	}
}

func TestSettingsThemeIsPerUser(t *testing.T) {
	env := newSettingsEnv(t)
	const other = "other@example.com"

	settingsBody(t, env.putSettings(davTestUser, `{"theme":"outlook-dark"}`))
	if got := settingsBody(t, env.getSettings(other)).Theme; got != "" {
		t.Errorf("other user's theme = %q, want empty", got)
	}

	settingsBody(t, env.putSettings(other, `{"theme":"high-contrast"}`))
	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "outlook-dark" {
		t.Errorf("first user's theme = %q, want outlook-dark", got)
	}
	if got := settingsBody(t, env.getSettings(other)).Theme; got != "high-contrast" {
		t.Errorf("other user's theme = %q, want high-contrast", got)
	}
}

func TestSettingsUnregisteredStoredThemeReadsEmpty(t *testing.T) {
	env := newSettingsEnv(t)
	settingsBody(t, env.getSettings(davTestUser)) // creates the row
	if err := env.db.Model(&model.UserSettings{}).Where("user_id = ?", env.userID).
		Update("theme", "retired-theme").Error; err != nil {
		t.Fatalf("store a retired theme id: %v", err)
	}

	if got := settingsBody(t, env.getSettings(davTestUser)).Theme; got != "" {
		t.Errorf("GET theme for a retired id = %q, want empty", got)
	}
	// The save response follows the same rule.
	if got := settingsBody(t, env.putSettings(davTestUser, `{"timezone":"UTC"}`)).Theme; got != "" {
		t.Errorf("PUT response theme for a retired id = %q, want empty", got)
	}
}
