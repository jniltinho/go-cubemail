package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"go-cubemail/internal/config"
	"go-cubemail/internal/handler"
	appMiddleware "go-cubemail/internal/server/middleware"
	"go-cubemail/internal/session"
	"gorm.io/gorm"
)

// AppVersion can be set via ldflags: -ldflags "-X go-cubemail/internal/server.AppVersion=1.2.3"
var AppVersion = "dev"

func Start(cfg *config.Config, db *gorm.DB, embeddedFiles embed.FS) error {
	session.InitDB(db)

	e := echo.New()

	// ── Global middlewares ────────────────────────────────────────────────────
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())
	e.Use(appMiddleware.SecurityHeaders())
	e.Use(appMiddleware.CSRF())
	e.Use(echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		LogMethod:        true,
		LogURI:           true,
		LogStatus:        true,
		LogLatency:       true,
		LogHost:          true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogUserAgent:     true,
		LogRemoteIP:      true,
		LogRequestID:     true,
		LogValuesFunc: func(c *echo.Context, v echoMiddleware.RequestLoggerValues) error {
			slog.Info("REQUEST",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency.Nanoseconds(),
				"host", v.Host,
				"bytes_in", v.ContentLength,
				"bytes_out", v.ResponseSize,
				"user_agent", v.UserAgent,
				"remote_ip", v.RemoteIP,
				"request_id", v.RequestID,
			)
			return nil
		},
	}))

	// ── Session cleanup goroutine ─────────────────────────────────────────────
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			session.Cleanup(30 * time.Minute)
		}
	}()

	// ── Embedded SPA filesystem (web/dist) ───────────────────────────────────
	distFS, err := fs.Sub(embeddedFiles, "web/dist")
	if err != nil {
		return fmt.Errorf("failed to open embedded dist: %w", err)
	}

	// ── API handlers ─────────────────────────────────────────────────────────
	h := handler.New(cfg)

	authRateLimit := appMiddleware.NewRateLimit(5, time.Minute)
	authMiddleware := appMiddleware.RequireAuth(cfg.Session.Name)

	// Public auth endpoints
	auth := e.Group("/api/auth")
	auth.POST("/login", h.Auth.DoLogin, authRateLimit)
	auth.POST("/logout", h.Auth.DoLogout)
	auth.GET("/me", h.Auth.Me, authMiddleware)

	// Protected mail endpoints
	api := e.Group("/api", authMiddleware)

	// Folders
	api.GET("/folders", h.Mailbox.FoldersJSON)
	api.POST("/folders", h.Mailbox.CreateSubfolder)
	api.POST("/folders/rename", h.Mailbox.RenameFolder)
	api.POST("/folders/delete", h.Mailbox.DeleteFolder)
	api.GET("/folders/:name/count", h.Mailbox.UnreadCountJSON)

	// Mail (mailbox messages)
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
	api.POST("/contacts", h.Contacts.Create)
	api.PUT("/contacts/:id", h.Contacts.Update)
	api.DELETE("/contacts/:id", h.Contacts.Delete)

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

	// ── Start server ─────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
		e.Logger.Info("Starting server with TLS/HTTPS")
		srv := &http.Server{
			Addr:              addr,
			Handler:           e,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		}
		return srv.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey)
	}
	return e.Start(addr)
}
