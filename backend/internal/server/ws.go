package server

import (
	"cardfinders/internal/engine"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Pinging keeps idle sockets from being reaped by intermediaries; the interval
// must stay under common 30-60s idle windows.
const (
	pingInterval = 10 * time.Second
	pingTimeout  = 5 * time.Second
)

// wsConn is one live client connection bound to a seat. coder/websocket's Write
// is concurrency-safe, so broadcast needs no per-connection lock.
type wsConn struct {
	conn *websocket.Conn
	seat engine.SeatID
}

// serveWS handles GET /ws?token=<SessionToken>. Token errors reject with HTTP
// before the upgrade; post-upgrade errors close the socket with a reason.
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: errorDetail{
			Code:    "token_required",
			Message: "a session token is required to connect",
		}})
		return
	}

	s.mu.Lock()
	seat, err := s.engine.ResolveSeat(engine.SessionToken(token))
	s.mu.Unlock()
	if err != nil {
		slog.Info("ws connect rejected: token invalid", "token", tokenTag(token))
		writeJSON(w, http.StatusUnauthorized, errorBody{Error: errorDetail{
			Code:    "token_invalid",
			Message: "the session token is invalid",
		}})
		return
	}
	slog.Info("ws connect", "token", tokenTag(token), "seat", seat)

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Error("ws accept failed", "seat", seat, "token", tokenTag(token), "err", err)
		return
	}
	c := &wsConn{conn: conn, seat: seat}

	// A seat already held by a live connection is rejected; the existing
	// connection is left untouched.
	if !s.hub.register(c) {
		slog.Info("ws connect rejected: seat occupied", "seat", seat, "token", tokenTag(token))
		_ = conn.Close(websocket.StatusPolicyViolation, "seat_occupied")
		return
	}
	// release frees only the seat→conn mapping; game state survives for reconnect.
	defer func() {
		s.hub.release(c)
		slog.Info("ws disconnect", "seat", seat, "token", tokenTag(token))
	}()

	ctx := r.Context()
	if err := s.pushView(ctx, c); err != nil {
		slog.Error("ws initial push failed", "seat", seat, "token", tokenTag(token), "err", err)
		_ = conn.Close(websocket.StatusInternalError, "initial push failed")
		return
	}

	pingCtx, stopPing := context.WithCancel(ctx)
	defer stopPing()
	go s.pingLoop(pingCtx, c)

	s.readLoop(ctx, c)
}

// pingLoop must run concurrently with readLoop: Ping waits for the pong but
// relies on a concurrent Reader to read it.
func (s *Server) pingLoop(ctx context.Context, c *wsConn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, pingTimeout)
			err := c.conn.Ping(pctx)
			cancel()
			if err != nil {
				if ctx.Err() == nil {
					slog.Warn("ws ping failed", "seat", c.seat, "err", err)
					_ = c.conn.Close(websocket.StatusPolicyViolation, "ping timeout")
				}
				return
			}
		}
	}
}

// readLoop serves one connection until it closes or errors.
func (s *Server) readLoop(ctx context.Context, c *wsConn) {
	for {
		var in inbound
		if err := wsjson.Read(ctx, c.conn, &in); err != nil {
			slog.Info("ws read loop ended", "seat", c.seat, "err", err,
				"closeStatus", websocket.CloseStatus(err))
			return
		}
		slog.Debug("ws recv", "seat", c.seat, "type", in.Type, "msg", in)

		action, code, ok := decodeAction(in)
		if !ok {
			s.sendError(ctx, c, code, "the message could not be decoded")
			continue
		}

		s.mu.Lock()
		_, applyErr := s.engine.Apply(c.seat, action)
		s.mu.Unlock()

		if applyErr != nil {
			s.sendError(ctx, c, actionErrorCode(applyErr), applyErr.Error())
			continue
		}

		s.broadcast(ctx)
	}
}

// pushView sends the seat's PlayerView as a `state` frame, read under the lock.
func (s *Server) pushView(ctx context.Context, c *wsConn) error {
	s.mu.Lock()
	view := s.engine.View(c.seat)
	s.mu.Unlock()
	msg := outboundState{Type: "state", View: view}
	slog.Debug("ws send", "seat", c.seat, "type", msg.Type, "msg", msg)
	return wsjson.Write(ctx, c.conn, msg)
}

// broadcast pushes each connected seat its own per-player snapshot.
func (s *Server) broadcast(ctx context.Context) {
	for _, c := range s.hub.snapshot() {
		_ = s.pushView(ctx, c)
	}
}

// sendError sends an `error` frame; a write failure is left to the read loop.
func (s *Server) sendError(ctx context.Context, c *wsConn, code, message string) {
	slog.Debug("ws send", "seat", c.seat, "type", "error", "code", code, "message", message)
	_ = wsjson.Write(ctx, c.conn, outboundError{
		Type:    "error",
		Code:    code,
		Message: message,
	})
}

// wsActionErrorMap maps Apply sentinel errors to stable WS error codes so the
// client can react to a specific rejection (e.g. safe_call_spent).
var wsActionErrorMap = map[error]string{
	engine.ErrNoGame:                  "no_game",
	engine.ErrGameEnded:               "game_ended",
	engine.ErrUnknownAction:           "unknown_action",
	engine.ErrWrongPhase:              "wrong_phase",
	engine.ErrNotYourTurn:             "not_your_turn",
	engine.ErrSeatIneligible:          "seat_ineligible",
	engine.ErrInvalidOpeningSelection: "invalid_opening_selection",
	engine.ErrSpecialOpening:          "special_opening",
	engine.ErrInvalidSource:           "invalid_source",
	engine.ErrGoldSource:              "gold_source",
	engine.ErrInvalidTarget:           "invalid_target",
	engine.ErrTargetRevealed:          "target_revealed",
	engine.ErrTargetOwn:               "target_own",
	engine.ErrGoldNotTargetable:       "gold_not_targetable",
	engine.ErrSafeCallSpent:           "safe_call_spent",
	engine.ErrSafeCallTargets:         "safe_call_targets",
	engine.ErrNoPendingChoice:         "no_pending_choice",
	engine.ErrInvalidChoice:           "invalid_choice",
	engine.ErrSelfRevealIncomplete:    "self_reveal_incomplete",
	engine.ErrGoldRevealIneligible:    "gold_reveal_ineligible",
}

// actionErrorCode resolves an Apply error to its WS code, else "invalid_action".
func actionErrorCode(err error) string {
	for sentinel, code := range wsActionErrorMap {
		if errors.Is(err, sentinel) {
			return code
		}
	}
	return "invalid_action"
}
