// Package poll implements the Server-Sent Events hub and background IMAP poller
// that notifies connected clients when new mail arrives in their INBOX.
package poll

import "sync"

// Hub manages per-session SSE channels.
// Each connected browser tab subscribes to a buffered channel; the poller sends
// event names to that channel when new mail is detected.
type Hub struct {
	mu       sync.RWMutex
	channels map[string]chan string
}

func newHub() *Hub { return &Hub{channels: make(map[string]chan string)} }

// Default is the process-wide hub instance.
var Default = newHub()

// Subscribe registers a session for SSE events and returns a buffered channel.
// If the session already has a channel, the old one is closed first.
func (h *Hub) Subscribe(sessID string) chan string {
	ch := make(chan string, 8)
	h.mu.Lock()
	if old, ok := h.channels[sessID]; ok {
		close(old)
	}
	h.channels[sessID] = ch
	h.mu.Unlock()
	return ch
}

// Unsubscribe closes the session's channel and removes it from the hub.
func (h *Hub) Unsubscribe(sessID string) {
	h.mu.Lock()
	if ch, ok := h.channels[sessID]; ok {
		close(ch)
		delete(h.channels, sessID)
	}
	h.mu.Unlock()
}

// Send delivers an event name to the session's channel non-blocking.
// The event is silently dropped if the channel buffer is full.
func (h *Hub) Send(sessID, event string) {
	h.mu.RLock()
	ch, ok := h.channels[sessID]
	h.mu.RUnlock()
	if ok {
		select {
		case ch <- event:
		default:
		}
	}
}
