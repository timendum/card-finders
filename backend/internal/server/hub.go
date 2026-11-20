package server

import (
	"cardfinders/internal/engine"
	"sync"

	"github.com/coder/websocket"
)

// hub maps each session token to its one live WebSocket connection. Keyed by
// token, not seat, because a replay reshuffles seats. Guarded by its own mutex,
// independent of the engine mutex, since it holds only transport associations.
type hub struct {
	mu    sync.Mutex
	conns map[engine.SessionToken]*wsConn
}

func newHub() *hub {
	return &hub{conns: make(map[engine.SessionToken]*wsConn)}
}

// register binds c to its token if free, returning false when another
// connection already holds it (left untouched so the caller rejects the
// newcomer).
func (h *hub) register(c *wsConn) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, taken := h.conns[c.token]; taken {
		return false
	}
	h.conns[c.token] = c
	return true
}

// release unbinds c only if c still holds its token. The guard matters on
// reconnect races: an older connection's deferred release must not evict a
// newer one that already took the slot.
func (h *hub) release(c *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.token] == c {
		delete(h.conns, c.token)
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

// closeAll closes every registered connection with reason; each read loop then
// ends and releases its own entry.
func (h *hub) closeAll(reason string) {
	for _, c := range h.snapshot() {
		_ = c.conn.Close(websocket.StatusPolicyViolation, reason)
	}
}
