package handler

// Test harness for the DAV handlers.
//
// Everything runs against an in-memory SQLite database and an Echo instance
// with the IMAP-backed auth middleware replaced by a stub, so the whole CalDAV
// and CardDAV surface can be exercised without a mail server, a network or any
// external client.

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-cubemail/internal/config"
	"go-cubemail/internal/database"
	"go-cubemail/internal/model"
	"go-cubemail/internal/activesync/state"
	"github.com/labstack/echo/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// davMethods mirrors the method list registered by the server package.
var davTestMethods = []string{
	http.MethodOptions, http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete,
	"PROPFIND", "PROPPATCH", "REPORT", "MKCALENDAR", "MKCOL",
}

// davTestUser identifies the principal used by the tests.
const davTestUser = "nilton@example.com"

// testEnv bundles everything a DAV test needs.
type testEnv struct {
	t    *testing.T
	db   *gorm.DB
	echo *echo.Echo
	h    *Handlers
	// userID is the model.User row the stub auth middleware authenticates as.
	userID uint
}

// newTestEnv builds an isolated environment with one user and a default calendar.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Identity{}, &model.Contact{}, &model.ContactGroup{},
		&model.Draft{}, &model.UserSettings{}, &model.Session{},
		&model.Calendar{}, &model.CalendarShare{}, &model.CalendarSubscription{},
		&model.Event{}, &model.EventAttendee{}, &model.PushSubscription{},
		&model.AddressBook{}, &model.DAVChange{},
		&state.EasDevice{}, &state.EasFolderState{}, &state.ImapFolderMapping{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if err := database.FinishDAVMigration(db); err != nil {
		t.Fatalf("dav backfill: %v", err)
	}

	user := model.User{ImapUser: davTestUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cfg := &config.Config{}
	cfg.Server.BaseURL = "http://localhost:8080"
	h := New(cfg, db)

	e := echo.New()
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		c.Response().WriteHeader(http.StatusInternalServerError)
	}

	// Stub of middleware.CalDAVAuth: the real one validates Basic credentials
	// against IMAP, which no unit test should require.
	stubAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("caldav_user_id", user.ID)
			c.Set("caldav_username", davTestUser)
			return next(c)
		}
	}

	group := e.Group("/dav", stubAuth)
	for _, m := range davTestMethods {
		group.Add(m, "", h.DAV.Handle)
		group.Add(m, "/", h.DAV.Handle)
		group.Add(m, "/*", h.DAV.Handle)
	}

	return &testEnv{t: t, db: db, echo: e, h: h, userID: user.ID}
}

// do issues a request against the test server and returns the recorder.
func (env *testEnv) do(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	env.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)
	return rec
}

// propfind issues a PROPFIND with the given depth and body.
func (env *testEnv) propfind(path, depth, body string) *httptest.ResponseRecorder {
	return env.do("PROPFIND", path, body, map[string]string{
		"Depth":        depth,
		"Content-Type": "application/xml",
	})
}

// report issues a REPORT with the given body.
func (env *testEnv) report(path, body string) *httptest.ResponseRecorder {
	return env.do("REPORT", path, body, map[string]string{
		"Depth":        "1",
		"Content-Type": "application/xml",
	})
}

// defaultCalendar returns the user's auto-provisioned calendar.
func (env *testEnv) defaultCalendar() *model.Calendar {
	env.t.Helper()
	cal, err := env.h.CalDAV.calRepo.EnsureDefault(env.userID)
	if err != nil {
		env.t.Fatalf("ensure default calendar: %v", err)
	}
	return cal
}

// defaultAddressBook returns the user's auto-provisioned address book.
func (env *testEnv) defaultAddressBook() *model.AddressBook {
	env.t.Helper()
	book, err := env.h.CardDAV.bookRepo.EnsureDefault(env.userID)
	if err != nil {
		env.t.Fatalf("ensure default address book: %v", err)
	}
	return book
}

// ── Multistatus parsing ───────────────────────────────────────────────────

// msResponse is one parsed DAV:response entry.
type msResponse struct {
	Href      string `xml:"href"`
	Status    string `xml:"status"`
	Propstats []struct {
		Status string `xml:"status"`
		Prop   struct {
			Inner string `xml:",innerxml"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

// multistatusDoc is a parsed 207 response body.
type multistatusDoc struct {
	XMLName   xml.Name     `xml:"multistatus"`
	Responses []msResponse `xml:"response"`
	SyncToken string       `xml:"sync-token"`
}

// parseMultistatus decodes a 207 body, failing the test if it is not valid XML.
func parseMultistatus(t *testing.T, body []byte) multistatusDoc {
	t.Helper()
	var doc multistatusDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("multistatus is not valid XML: %v\nbody: %s", err, body)
	}
	return doc
}

// find returns the response whose href ends with the given suffix.
func (d multistatusDoc) find(suffix string) (msResponse, bool) {
	for _, r := range d.Responses {
		if strings.HasSuffix(r.Href, suffix) {
			return r, true
		}
	}
	return msResponse{}, false
}

// okProps concatenates the property bodies reported with a 200 status.
func (r msResponse) okProps() string {
	var sb strings.Builder
	for _, ps := range r.Propstats {
		if strings.Contains(ps.Status, "200") {
			sb.WriteString(ps.Prop.Inner)
		}
	}
	return sb.String()
}

// notFoundProps concatenates the property bodies reported with a 404 status.
func (r msResponse) notFoundProps() string {
	var sb strings.Builder
	for _, ps := range r.Propstats {
		if strings.Contains(ps.Status, "404") {
			sb.WriteString(ps.Prop.Inner)
		}
	}
	return sb.String()
}

// isRemoved reports whether the entry is a sync-collection tombstone.
func (r msResponse) isRemoved() bool {
	return strings.Contains(r.Status, "404")
}

// assertStatus fails the test when the recorder holds an unexpected status.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, want, rec.Body.String())
	}
}

// assertContains fails the test when the haystack lacks the needle.
func assertContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s: expected to contain %q\ngot: %s", context, needle, haystack)
	}
}

// assertNotContains fails the test when the haystack holds the needle.
func assertNotContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("%s: expected NOT to contain %q\ngot: %s", context, needle, haystack)
	}
}

// bodyBytes returns the recorder body.
func bodyBytes(rec *httptest.ResponseRecorder) []byte { return bytes.TrimSpace(rec.Body.Bytes()) }
