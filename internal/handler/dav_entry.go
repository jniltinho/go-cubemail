package handler

// Single entry point for the /dav namespace.
//
// WebDAV uses methods the Echo router does not know (PROPFIND, REPORT, MKCOL,
// MKCALENDAR, PROPPATCH) and clients are inconsistent about trailing slashes.
// Rather than enumerating every method/path pair as a route, the whole subtree
// is routed here and dispatched from a single parsed path — which also gives
// one place to enforce that the user in the URL is the authenticated user.

import (
	"net/http"
	"strings"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/dav"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// DAVHandler routes every request under /dav to the CalDAV or CardDAV handler.
type DAVHandler struct {
	cfg      *config.Config
	calendar *CalDAVHandler
	contacts *CardDAVHandler
	calRepo  *repository.CalendarRepo
	bookRepo *repository.AddressBookRepo
}

// NewDAVHandler wires the DAV entry point to the shared repositories.
func NewDAVHandler(cfg *config.Config, db *gorm.DB, cal *CalDAVHandler, card *CardDAVHandler) *DAVHandler {
	return &DAVHandler{
		cfg:      cfg,
		calendar: cal,
		contacts: card,
		calRepo:  repository.NewCalendarRepo(db),
		bookRepo: repository.NewAddressBookRepo(db),
	}
}

// Handle dispatches one DAV request.
func (h *DAVHandler) Handle(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	username := caldavUsername(c)
	p := parseDAVPath(c.Request().URL.Path)
	if p.Type == davUnknown {
		return c.NoContent(http.StatusNotFound)
	}

	// The user segment must be the authenticated user. Answering 404 rather
	// than 403 keeps the existence of other users' collections invisible.
	if p.User != "" && !strings.EqualFold(p.User, username) {
		return c.NoContent(http.StatusNotFound)
	}

	method := strings.ToUpper(c.Request().Method)
	if method == http.MethodOptions {
		if p.isAddressBookScope() {
			return h.contacts.Options(c)
		}
		return h.calendar.Options(c)
	}

	switch p.Type {
	case davRoot, davPrincipal:
		return h.handlePrincipal(c, method, username)
	case davCalendarHome:
		return h.handleCalendarHome(c, method, userID, username)
	case davCalendar:
		return h.handleCalendar(c, method, userID, username, p)
	case davCalendarObject:
		return h.handleCalendarObject(c, method, userID, username, p)
	case davAddressBookHome:
		return h.handleAddressBookHome(c, method, userID, username)
	case davAddressBook:
		return h.handleAddressBook(c, method, userID, username, p)
	case davAddressObject:
		return h.handleAddressObject(c, method, userID, username, p)
	}
	return c.NoContent(http.StatusNotFound)
}

func (h *DAVHandler) handlePrincipal(c *echo.Context, method, username string) error {
	switch method {
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.calendar.propfindPrincipal(c, username, req)
	case http.MethodGet, http.MethodHead:
		c.Response().Header().Set("Location", principalHref(username))
		return c.NoContent(http.StatusMovedPermanently)
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleCalendarHome(c *echo.Context, method string, userID uint, username string) error {
	switch method {
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.calendar.propfindCalendarHome(c, userID, username, req, depthOf(c))
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleCalendar(c *echo.Context, method string, userID uint, username string, p davPath) error {
	// Creation methods act on a URI that does not exist yet.
	if method == "MKCALENDAR" || method == "MKCOL" {
		return h.calendar.mkCalendar(c, userID, p.Collection)
	}

	cal, err := h.calRepo.GetByURI(userID, p.Collection)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	switch method {
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.calendar.propfindCalendar(c, userID, username, cal, req, depthOf(c))
	case "REPORT":
		return h.calendar.report(c, userID, username, cal)
	case "PROPPATCH":
		return h.calendar.propPatchCalendar(c, username, cal)
	case http.MethodGet:
		return h.calendar.getCalendar(c, userID, cal)
	case http.MethodDelete:
		return h.calendar.deleteCalendar(c, userID, cal)
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleCalendarObject(c *echo.Context, method string, userID uint, username string, p davPath) error {
	cal, err := h.calRepo.GetByURI(userID, p.Collection)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	switch method {
	case http.MethodPut:
		return h.calendar.putCalendarObject(c, userID, cal, p.Resource)
	case http.MethodDelete:
		return h.calendar.deleteCalendarObject(c, userID, cal, p.Resource)
	}

	ev, err := h.calendar.eventRepo.GetByResourceURI(userID, cal.ID, p.Resource)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	switch method {
	case http.MethodGet:
		return h.calendar.getCalendarObject(c, ev, true)
	case http.MethodHead:
		return h.calendar.getCalendarObject(c, ev, false)
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.calendar.propfindCalendarObject(c, username, cal, ev, req)
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleAddressBookHome(c *echo.Context, method string, userID uint, username string) error {
	switch method {
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.contacts.propfindHome(c, userID, username, req, depthOf(c))
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleAddressBook(c *echo.Context, method string, userID uint, username string, p davPath) error {
	if method == "MKCOL" || method == "MKCALENDAR" {
		return h.contacts.mkAddressBook(c, userID, p.Collection)
	}

	book, err := h.bookRepo.GetByURI(userID, p.Collection)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	switch method {
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.contacts.propfindAddressBook(c, userID, username, book, req, depthOf(c))
	case "REPORT":
		return h.contacts.report(c, userID, username, book)
	case "PROPPATCH":
		return h.contacts.propPatchAddressBook(c, username, book)
	case http.MethodDelete:
		return h.contacts.deleteAddressBook(c, userID, book)
	}
	return methodNotAllowed(c)
}

func (h *DAVHandler) handleAddressObject(c *echo.Context, method string, userID uint, username string, p davPath) error {
	book, err := h.bookRepo.GetByURI(userID, p.Collection)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	switch method {
	case http.MethodPut:
		return h.contacts.putAddressObject(c, userID, book, p.Resource)
	case http.MethodDelete:
		return h.contacts.deleteAddressObject(c, userID, book, p.Resource)
	}

	ct, err := h.contacts.contactRepo.GetByResourceURI(userID, book.ID, p.Resource)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	switch method {
	case http.MethodGet:
		return h.contacts.getAddressObject(c, ct, true)
	case http.MethodHead:
		return h.contacts.getAddressObject(c, ct, false)
	case "PROPFIND":
		req, ok := propfindBody(c)
		if !ok {
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		return h.contacts.propfindAddressObject(c, username, book, ct, req)
	}
	return methodNotAllowed(c)
}

// propfindBody decodes the PROPFIND request document, reporting false when the
// client sent more than the budget. Parsing a truncated document would silently
// fall back to allprop and answer a question the client never asked.
func propfindBody(c *echo.Context) (propfindRequest, bool) {
	body, tooLarge := readBody(c, davRequestBodyLimit)
	if tooLarge {
		return propfindRequest{}, false
	}
	return parsePropfind(body), true
}

// methodNotAllowed answers with the allowed set so clients stop retrying.
func methodNotAllowed(c *echo.Context) error {
	c.Response().Header().Set("Allow",
		"OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, REPORT, MKCALENDAR, MKCOL")
	return c.NoContent(http.StatusMethodNotAllowed)
}

// CleanupChangelog prunes changelog rows older than the retention window.
// Collections whose entries are dropped record the highest discarded revision,
// so a client arriving with an older token is told to run a full sync instead
// of silently missing changes.
func CleanupChangelog(db *gorm.DB, retention time.Duration) error {
	return dav.NewStore(db).Cleanup(time.Now().Add(-retention))
}
