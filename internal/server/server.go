package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
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

	// ── API handlers & Routes ────────────────────────────────────────────────
	h := handler.New(cfg)
	registerRoutes(e, cfg, h, distFS)

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
