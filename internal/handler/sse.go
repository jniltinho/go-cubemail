package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/repository"
	"go-cubemail/internal/session"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

const (
	// sseCheckInterval is how often the INBOX unread count is polled via IMAP.
	sseCheckInterval = 1 * time.Minute

	// sseHeartbeatInterval sends an SSE comment every 10 seconds to prevent
	// browsers and proxies from closing an idle connection before the 1-minute
	// check fires.
	sseHeartbeatInterval = 10 * time.Second
)

// SSEHandler serves Server-Sent Events for real-time new-mail notifications.
type SSEHandler struct {
	cfg      *config.Config
	db       *gorm.DB
	pushRepo *repository.PushSubscriptionRepo
}

// sseEvent formats a single SSE message with an optional event type.
func sseEvent(eventType, data string) string {
	if eventType != "" {
		return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	}
	return fmt.Sprintf("data: %s\n\n", data)
}

// Events handles GET /api/v1/events/stream.
//
// Opens a long-lived SSE stream and checks the INBOX unread count every 5 minutes.
// Sends a "new-mail" event when the count increases. If the IMAP connection is lost,
// the stream closes cleanly so the browser's EventSource can reconnect automatically.
func (h *SSEHandler) Events(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	userID, _ := getUserID(c, h.db)

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx response buffering
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	flush := func() { _ = rc.Flush() }

	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		slog.Error("SSE: failed to decrypt session password", "user", s.Username, "error", err)
		fmt.Fprint(w, sseEvent("error", `{"error":"auth"}`))
		flush()
		return nil
	}

	timeout := time.Duration(h.cfg.IMAP.TimeoutSec) * time.Second
	client, err := imap.Connect(
		s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		timeout, s.Username, pass, h.cfg.Server.Debug,
	)
	if err != nil {
		slog.Error("SSE: IMAP connect failed", "user", s.Username, "error", err)
		fmt.Fprint(w, sseEvent("error", `{"error":"imap"}`))
		flush()
		return nil
	}
	defer client.Close()

	lastCount, err := client.UnreadCount("INBOX")
	if err != nil {
		slog.Warn("SSE: could not read initial unread count, starting from 0", "user", s.Username, "error", err)
		lastCount = 0
	}
	slog.Info("SSE: stream opened", "user", s.Username, "unread", lastCount)

	// Immediate heartbeat confirms the stream is alive to the client.
	fmt.Fprint(w, sseEvent("heartbeat", `{"status":"connected"}`))
	flush()

	ctx := c.Request().Context()
	checkTicker := time.NewTicker(sseCheckInterval)
	heartbeatTicker := time.NewTicker(sseHeartbeatInterval)
	defer checkTicker.Stop()
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("SSE: client disconnected", "user", s.Username)
			return nil

		case <-heartbeatTicker.C:
			// SSE comment line — keeps the connection alive through proxies, no log spam.
			fmt.Fprint(w, ":\n\n")
			flush()

		case <-checkTicker.C:
			cur, err := client.UnreadCount("INBOX")
			if err != nil {
				// IMAP connection is gone. Close the stream so the browser's
				// EventSource reconnects cleanly (it will open a fresh IMAP connection).
				slog.Warn("SSE: IMAP check failed, closing stream", "user", s.Username, "error", err)
				fmt.Fprint(w, sseEvent("error", `{"error":"imap_lost"}`))
				flush()
				return nil
			}

			if cur > lastCount {
				slog.Info("SSE: new mail detected", "user", s.Username, "unread", cur)
				fmt.Fprint(w, sseEvent("new-mail", fmt.Sprintf(`{"unread":%d}`, cur)))
				flush()
				if h.pushRepo != nil {
					go SendNewMailNotification(h.cfg, h.pushRepo, userID, cur)
				}
			}
			lastCount = cur
		}
	}
}
