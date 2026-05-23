package poll

import "sync"

// Hub manages per-session SSE channels.
type Hub struct {
	mu       sync.RWMutex
	channels map[string]chan string
}

func newHub() *Hub { return &Hub{channels: make(map[string]chan string)} }

// Default is the process-wide hub instance.
var Default = newHub()

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

func (h *Hub) Unsubscribe(sessID string) {
	h.mu.Lock()
	if ch, ok := h.channels[sessID]; ok {
		close(ch)
		delete(h.channels, sessID)
	}
	h.mu.Unlock()
}

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
