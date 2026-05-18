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

	tmplRenderer, err := loadTemplates(embeddedFiles)
	if err != nil {
		return fmt.Errorf("falha ao carregar templates: %w", err)
	}
	e.Renderer = tmplRenderer

	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())
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

	// Housekeeping de sessões ociosas
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			session.Cleanup(30 * time.Minute)
		}
	}()

	// Arquivos estáticos
	publicFS, err := fs.Sub(embeddedFiles, "web/static")
	if err != nil {
		return fmt.Errorf("falha ao criar sub filesystem: %w", err)
	}
	staticHandler := http.FileServer(http.FS(publicFS))
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", staticHandler)))

	// Rotas públicas
	e.GET("/", func(c *echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/login")
	})

	h := handler.New(cfg)

	e.GET("/login", h.Auth.LoginPage)
	e.POST("/login", h.Auth.DoLogin)
	e.POST("/logout", h.Auth.DoLogout)

	authMiddleware := appMiddleware.RequireAuth(cfg.Session.Name)

	// Rotas protegidas
	mail := e.Group("/mail", authMiddleware)
	mail.GET("/:mailbox", h.Mailbox.List)
	mail.GET("/:mailbox/:uid", h.Message.Read)
	mail.GET("/:mailbox/:uid/download", h.Message.Download)
	mail.GET("/:mailbox/:uid/raw", h.Message.Raw)
	mail.POST("/:mailbox/:uid/flag", h.Message.Flag)
	mail.POST("/:mailbox/:uid/move", h.Message.Move)
	mail.DELETE("/:mailbox/:uid", h.Message.Delete)
	mail.DELETE("/:mailbox", h.Message.EmptyTrash)
	mail.GET("/:mailbox/:uid/attachment/:part", h.Message.Attachment)

	compose := e.Group("/compose", authMiddleware)
	compose.GET("", h.Compose.New)
	compose.GET("/reply/:mailbox/:uid", h.Compose.Reply)
	compose.GET("/forward/:mailbox/:uid", h.Compose.Forward)
	compose.POST("/send", h.Compose.Send)
	compose.POST("/draft", h.Compose.SaveDraft)
	compose.POST("/upload", h.Compose.UploadAttachment)
	compose.GET("/attachment/:id", h.Compose.ServeAttachment)

	contacts := e.Group("/contacts", authMiddleware)
	contacts.GET("", h.Contacts.Index)
	contacts.GET("/new", h.Contacts.New)
	contacts.POST("", h.Contacts.Create)
	contacts.GET("/:id/edit", h.Contacts.Edit)
	contacts.PUT("/:id", h.Contacts.Update)
	contacts.DELETE("/:id", h.Contacts.Delete)

	settings := e.Group("/settings", authMiddleware)
	settings.GET("", h.Settings.Index)
	settings.POST("", h.Settings.Save)

	e.GET("/search", h.Search.Results, authMiddleware)

	// API JSON
	api := e.Group("/api", authMiddleware)
	api.GET("/folders", h.Mailbox.FoldersJSON)
	api.POST("/folders", h.Mailbox.CreateSubfolder)
	api.POST("/folders/rename", h.Mailbox.RenameFolder)
	api.POST("/folders/delete", h.Mailbox.DeleteFolder)
	api.GET("/folders/:name/count", h.Mailbox.UnreadCountJSON)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
		e.Logger.Info("Iniciando servidor com suporte a TLS/HTTPS")
		server := &http.Server{
			Addr:    addr,
			Handler: e,
		}
		return server.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey)
	}
	return e.Start(addr)
}
