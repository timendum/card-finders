package server

// REST handlers under /api

import (
	"cardfinders/internal/engine"
	"context"
	"log/slog"
	"net/http"
	"time"
)

// broadcastTimeout bounds the post-join state push to connected seats.
const broadcastTimeout = 5 * time.Second

type levelsResponse struct {
	Levels []engine.Level `json:"levels"`
	// CurrentLevel is the running game's level number, 0 when no game exists.
	CurrentLevel int `json:"currentLevel"`
}

type createRequest struct {
	LevelNumber int `json:"levelNumber"`
}

type createResponse struct {
	JoinCode string `json:"joinCode"`
}

type joinRequest struct {
	JoinCode string `json:"joinCode"`
}

type joinResponse struct {
	SessionToken string `json:"sessionToken"`
	Seat         int    `json:"seat"`
}

func (s *Server) handleLevels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	s.mu.Lock()
	levels := s.engine.Levels()
	current := s.engine.CurrentLevel()
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, levelsResponse{Levels: levels, CurrentLevel: current})
}

// handleGame routes /api/game: POST creates (or replays), DELETE ends.
func (s *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateGame(w, r)
	case http.MethodDelete:
		s.handleEndGame(w)
	default:
		methodNotAllowed(w, http.MethodPost, http.MethodDelete)
	}
}

// handleEndGame ends the game. A Waiting or Ended game is dropped and every
// socket closed; a running game becomes a loss and the seats are notified.
func (s *Server) handleEndGame(w http.ResponseWriter) {
	s.mu.Lock()
	closed, err := s.engine.EndGame()
	s.mu.Unlock()

	if err != nil {
		writeError(w, restError{http.StatusNotFound, "no_game"}, err)
		return
	}
	slog.Info("game ended by admin", "closed", closed)
	w.WriteHeader(http.StatusNoContent)

	if closed {
		s.hub.closeAll("game_closed")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
	defer cancel()
	s.broadcast(ctx)
}

func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	s.mu.Lock()
	joinCode, err := s.engine.CreateGame(req.LevelNumber)
	s.mu.Unlock()

	if err != nil {
		writeError(w, classifyCreateError(err), err)
		return
	}
	slog.Info("game created", "level", req.LevelNumber, "joinCode", joinCode)
	writeJSON(w, http.StatusCreated, createResponse{JoinCode: joinCode})

	// A replay deals to seated players who are already connected.
	ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
	defer cancel()
	s.broadcast(ctx)
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req joinRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	s.mu.Lock()
	seat, token, err := s.engine.Join(req.JoinCode)
	s.mu.Unlock()

	if err != nil {
		writeError(w, classifyJoinError(err), err)
		return
	}
	slog.Info("player joined", "joinCode", req.JoinCode, "seat", seat,
		"token", tokenTag(string(token)))
	writeJSON(w, http.StatusOK, joinResponse{
		SessionToken: string(token),
		Seat:         int(seat),
	})

	// Push the new state to seats already connected (the joiner has no socket
	// yet). Detached from r.Context(), which is cancelled once we return.
	ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
	defer cancel()
	s.broadcast(ctx)
}

// tokenTag abbreviates a session token so full tokens never reach the logs.
func tokenTag(token string) string {
	const n = 6
	if len(token) <= n {
		return token
	}
	return token[:n] + "…"
}
