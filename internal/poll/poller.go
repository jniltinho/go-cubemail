package poll

import (
	"log/slog"
	"time"

	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
)

// Start launches the background goroutine that checks for new mail every 5 minutes.
func Start(secretKey string, tlsEnabled bool, timeout time.Duration, hub *Hub) {
	go func() {
		unseen := make(map[string]uint32) // sessID → last known unseen count
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			checkAll(secretKey, tlsEnabled, timeout, hub, unseen)
		}
	}()
}

func checkAll(secretKey string, tlsEnabled bool, timeout time.Duration, hub *Hub, state map[string]uint32) {
	sessions := session.All()
	for sessID, s := range sessions {
		pass, err := s.Password(secretKey)
		if err != nil {
			continue
		}
		conn, err := imap.Connect(s.IMAPHost, s.IMAPPort, tlsEnabled, timeout, s.Username, pass, false)
		if err != nil {
			continue
		}
		count, err := conn.MessageCount("INBOX")
		conn.Close()
		if err != nil {
			continue
		}
		prev, exists := state[sessID]
		if !exists {
			state[sessID] = count
			continue
		}
		if count > prev {
			slog.Info("New mail detected", "user", s.Username, "new_messages", count-prev)
			hub.Send(sessID, "new-mail")
		}
		state[sessID] = count
	}
}
