package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
	"go-cubemail/internal/handler"
	appMiddleware "go-cubemail/internal/server/middleware"
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

	// Search
	api.GET("/search", h.Search.Results)

	// Contacts
	api.GET("/contacts", h.Contacts.Index)
	api.GET("/contacts/export", h.Contacts.Export)
	api.POST("/contacts", h.Contacts.Create)
	api.POST("/contacts/import", h.Contacts.Import)
	api.PUT("/contacts/:id", h.Contacts.Update)
	api.DELETE("/contacts/:id", h.Contacts.Delete)

}

func registerRoutes(e *echo.Echo, cfg *config.Config, h *handler.Handlers, distFS fs.FS) {
	authRateLimit := appMiddleware.NewRateLimit(5, time.Minute)
	authMiddleware := appMiddleware.RequireAuth(cfg.Session.Name)

	registerAPIRoutes(e.Group("/api/v1"), h, authMiddleware, authRateLimit)
	// registerAPIRoutes(e.Group("/api/v2"), h, authMiddleware, authRateLimit)

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
