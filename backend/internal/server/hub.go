package server

import (
	"cardfinders/internal/engine"
	"sync"
)

// hub maps each occupied seat to its one live WebSocket connection. Guarded by
// its own mutex, independent of the engine mutex, since it holds only transport
// associations.
type hub struct {
	mu    sync.Mutex
	conns map[engine.SeatID]*wsConn
}

func newHub() *hub {
	return &hub{conns: make(map[engine.SeatID]*wsConn)}
}

// register binds c to its seat if free, returning false when another connection
// already holds the seat (left untouched so the caller rejects the newcomer).
func (h *hub) register(c *wsConn) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, taken := h.conns[c.seat]; taken {
		return false
	}
	h.conns[c.seat] = c
	return true
}

// release unbinds c's seat only if c still holds it. The guard matters on
// reconnect races: an older connection's deferred release must not evict a
// newer one that already took the seat.
func (h *hub) release(c *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.seat] == c {
		delete(h.conns, c.seat)
	}
}

// snapshot copies the registered connections so the caller can broadcast
// without holding the hub lock during I/O.
func (h *hub) snapshot() []*wsConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*wsConn, 0, len(h.conns))
	for _, c := range h.conns {
		out = append(out, c)
	}
	return out
}
