package handler

import (
	"fmt"
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

// SSEHandler serves Server-Sent Events for real-time new-mail notifications.
type SSEHandler struct {
	cfg *config.Config
}

// sseEvent formats a single SSE message with an optional event type.
func sseEvent(eventType, data string) string {
	if eventType != "" {
		return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	}
	return fmt.Sprintf("data: %s\n\n", data)
}

// Events handles GET /api/v1/events.
// Opens a streaming SSE connection that polls the inbox every 30 s and sends
// a "new-mail" event whenever the unread count increases.
// The connection is held open until the client disconnects or the server shuts down.
func (h *SSEHandler) Events(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	flush := func() { _ = rc.Flush() }

	// Send an immediate heartbeat so the client knows the stream is alive.
	fmt.Fprint(w, sseEvent("heartbeat", `{"status":"connected"}`))
	flush()

	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		fmt.Fprint(w, sseEvent("error", `{"error":"auth"}`))
		flush()
		return nil
	}

	timeout := time.Duration(h.cfg.IMAP.TimeoutSec) * time.Second
	client, err := imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		timeout, s.Username, pass, h.cfg.Server.Debug)
	if err != nil {
		fmt.Fprint(w, sseEvent("error", `{"error":"imap"}`))
		flush()
		return nil
	}
	defer client.Close()

	lastCount, _ := client.UnreadCount("INBOX")
	ctx := c.Request().Context()
	ticker := time.NewTicker(30 * time.Second)
	heartbeat := time.NewTicker(55 * time.Second) // keep-alive before proxy timeout
	defer ticker.Stop()
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			fmt.Fprint(w, sseEvent("heartbeat", `{}`))
			flush()
		case <-ticker.C:
			cur, err := client.UnreadCount("INBOX")
			if err != nil {
				// IMAP connection lost — tell client to fall back to polling.
				fmt.Fprint(w, sseEvent("error", `{"error":"imap"}`))
				flush()
				return nil
			}
			if cur > lastCount {
				fmt.Fprint(w, sseEvent("new-mail", fmt.Sprintf(`{"unread":%d}`, cur)))
				flush()
			}
			lastCount = cur
		}
	}
}
