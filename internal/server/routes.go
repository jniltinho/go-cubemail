package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/activesync"
	"go-cubemail/internal/config"
	"go-cubemail/internal/handler"
	appMiddleware "go-cubemail/internal/server/middleware"
	"gorm.io/gorm"
)

func registerAPIRoutes(g *echo.Group, h *handler.Handlers, authMiddleware, authRateLimit echo.MiddlewareFunc) {
	// Public: app version
	g.GET("/version", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"version": AppVersion})
	})

	// Public auth endpoints
	auth := g.Group("/auth")
	auth.POST("/login", h.Auth.DoLogin, authRateLimit)
	auth.POST("/logout", h.Auth.DoLogout)
	auth.GET("/me", h.Auth.Me, authMiddleware)
	auth.GET("/quota", h.Auth.Quota, authMiddleware)

	// Protected endpoints
	api := g.Group("", authMiddleware)

	// Folders
	api.GET("/folders", h.Mailbox.FoldersJSON)
	api.POST("/folders", h.Mailbox.CreateSubfolder)
	api.POST("/folders/rename", h.Mailbox.RenameFolder)
	api.POST("/folders/delete", h.Mailbox.DeleteFolder)
	api.GET("/folders/:name/count", h.Mailbox.UnreadCountJSON)

	// Mail
	api.GET("/mail/:mailbox", h.Mailbox.List)
	api.GET("/mail/:mailbox/:uid", h.Message.Read)
	api.GET("/mail/:mailbox/:uid/download", h.Message.Download)
	api.GET("/mail/:mailbox/:uid/raw", h.Message.Raw)
	api.POST("/mail/:mailbox/:uid/flag", h.Message.Flag)
	api.POST("/mail/:mailbox/:uid/move", h.Message.Move)
	api.DELETE("/mail/:mailbox/:uid", h.Message.Delete)
	api.DELETE("/mail/:mailbox", h.Message.EmptyTrash)
	api.GET("/mail/:mailbox/:uid/attachment/:part", h.Message.Attachment)

	// Compose
	api.POST("/compose/send", h.Compose.Send)
	api.POST("/compose/draft", h.Compose.SaveDraft)
	api.POST("/compose/upload", h.Compose.UploadAttachment)

	// Server-Sent Events (new mail push)
	api.GET("/events", h.SSE.Events)

	// Search
	api.GET("/search", h.Search.Results)

	// Settings
	api.GET("/settings", h.Settings.GetSettings)
	api.PUT("/settings", h.Settings.SaveSettings)

	// Identities
	api.GET("/identities", h.Settings.ListIdentities)
	api.POST("/identities", h.Settings.CreateIdentity)
	api.PUT("/identities/:id", h.Settings.UpdateIdentity)
	api.DELETE("/identities/:id", h.Settings.DeleteIdentity)
	api.POST("/identities/:id/default", h.Settings.SetDefaultIdentity)

	// Contacts
	api.GET("/contacts", h.Contacts.Index)
	api.GET("/contacts/export", h.Contacts.Export)
	api.POST("/contacts", h.Contacts.Create)
	api.POST("/contacts/import", h.Contacts.Import)
	api.PUT("/contacts/:id", h.Contacts.Update)
	api.DELETE("/contacts/:id", h.Contacts.Delete)

	// Calendar
	api.GET("/calendar", h.Calendar.List)
	api.POST("/calendar", h.Calendar.Create)
	api.PUT("/calendar/:id", h.Calendar.Update)
	api.DELETE("/calendar/:id", h.Calendar.Delete)
	api.POST("/calendar/activation", h.Calendar.SetActivation)
	api.GET("/calendar/:id/export", h.Calendar.Export)
	api.POST("/calendar/:id/import", h.Calendar.Import)

	// Calendar sharing
	api.GET("/calendar/:id/shares", h.CalendarShare.List)
	api.POST("/calendar/:id/shares", h.CalendarShare.Create)
	api.DELETE("/calendar/:id/shares/:shareId", h.CalendarShare.Delete)

	// Calendar subscriptions (remote .ics)
	api.GET("/calendar/subscriptions", h.CalendarSubscription.List)
	api.POST("/calendar/subscriptions", h.CalendarSubscription.Create)
	api.DELETE("/calendar/subscriptions/:id", h.CalendarSubscription.Delete)
	api.POST("/calendar/subscriptions/:id/refresh", h.CalendarSubscription.Refresh)

	// Events
	api.GET("/events", h.Event.List)
	api.GET("/events/:id", h.Event.Get)
	api.POST("/events", h.Event.Create)
	api.POST("/events/rsvp-from-mail", h.Event.RSVPFromMail)
	api.PUT("/events/:id", h.Event.Update)
	api.DELETE("/events/:id", h.Event.Delete)
	api.POST("/events/:id/move", h.Event.Move)
	api.POST("/events/:id/rsvp", h.Event.RSVP)
	api.GET("/events/freebusy", h.Event.FreeBusy)

}

func registerCalDAVRoutes(e *echo.Echo, cfg *config.Config, h *handler.Handlers, db *gorm.DB) {
	caldavAuth := appMiddleware.CalDAVAuth(cfg, db)

	// /.well-known/caldav → principal discovery redirect.
	e.GET("/.well-known/caldav", h.CalDAV.WellKnown, caldavAuth)
	e.Add("PROPFIND", "/.well-known/caldav", h.CalDAV.WellKnown, caldavAuth)

	dav := e.Group("/dav", caldavAuth)
	dav.OPTIONS("/*", h.CalDAV.Options)

	// Principal
	dav.Add("PROPFIND", "/", h.CalDAV.PropFind)
	dav.Add("PROPFIND", "/:user/", h.CalDAV.PropFind)

	// Calendar home + collections
	dav.Add("PROPFIND", "/:user/calendars/", h.CalDAV.PropFind)
	dav.Add("PROPFIND", "/:user/calendars/:cal/", h.CalDAV.PropFind)
	dav.Add("PROPFIND", "/:user/calendars/:cal/:uid", h.CalDAV.PropFind)

	// REPORT (calendar-query, calendar-multiget)
	dav.Add("REPORT", "/:user/calendars/:cal/", h.CalDAV.Report)

	// Event resources
	dav.GET("/:user/calendars/:cal/", h.CalDAV.GetCalendar)
	dav.GET("/:user/calendars/:cal/:uid", h.CalDAV.GetEvent)
	dav.PUT("/:user/calendars/:cal/:uid", h.CalDAV.PutEvent)
	dav.DELETE("/:user/calendars/:cal/:uid", h.CalDAV.DeleteEvent)
}

func registerRoutes(e *echo.Echo, cfg *config.Config, h *handler.Handlers, db *gorm.DB, distFS fs.FS) {
	authRateLimit := appMiddleware.NewRateLimit(5, time.Minute)
	authMiddleware := appMiddleware.RequireAuth(cfg.Session.Name)

	registerAPIRoutes(e.Group("/api/v1"), h, authMiddleware, authRateLimit)
	registerActiveSyncRoutes(e, cfg, db)
	registerCalDAVRoutes(e, cfg, h, db)

	// ── SPA + Static file handler ────────────────────────────────────────────
	// Serves Vite build assets with correct MIME types.
	// For known static files (by extension), reads from distFS and streams.
	// For all other paths, falls back to index.html (Vue Router history mode).
	e.GET("/*", func(c *echo.Context) error {
		urlPath := c.Request().URL.Path

		// Paths with a file extension are treated as static assets
		ext := strings.ToLower(filepath.Ext(urlPath))
		if ext != "" {
			// Strip leading /
			fsPath := strings.TrimPrefix(urlPath, "/")
			data, err := fs.ReadFile(distFS, fsPath)
			if err != nil {
				return echo.ErrNotFound
			}
			// Resolve MIME type, fallback to binary stream
			ct := mime.TypeByExtension(ext)
			if ct == "" {
				ct = "application/octet-stream"
			}
			// JavaScript modules must be application/javascript
			if ext == ".js" || ext == ".mjs" {
				ct = "application/javascript; charset=utf-8"
			}
			// Long cache for hashed assets, no-cache for index
			if strings.HasPrefix(urlPath, "/assets/") {
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			c.Response().Header().Set("Content-Type", ct)
			_, _ = c.Response().Write(data)
			return nil
		}

		// No extension → SPA route → serve index.html
		indexHTML, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			return echo.ErrNotFound
		}
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		_, _ = c.Response().Write(indexHTML)
		return nil
	})
}

// registerActiveSyncRoutes mounts EAS and Autodiscover HTTP endpoints when activesync.enabled is true.
func registerActiveSyncRoutes(e *echo.Echo, cfg *config.Config, db *gorm.DB) {
	if !cfg.ActiveSync.Enabled {
		return
	}
	easAuth := appMiddleware.EASAuth(cfg)
	easHandler := activesync.NewHandler(cfg, db)
	autoHandler := activesync.NewAutodiscoverHandler(cfg)

	eas := e.Group("/Microsoft-Server-ActiveSync", easAuth)
	eas.OPTIONS("", easHandler.Options)
	eas.POST("", easHandler.Handle)

	ad := e.Group("/autodiscover")
	ad.POST("/autodiscover.xml", autoHandler.MobileSync, easAuth)
	e.GET("/.well-known/autodiscover/autodiscover.xml", autoHandler.MobileSync, easAuth)
}
