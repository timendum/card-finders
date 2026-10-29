package server

// REST handlers under /api

import (
	"cardfinders/internal/engine"
	"log/slog"
	"net/http"
)

type levelsResponse struct {
	Levels []engine.Level `json:"levels"`
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
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, levelsResponse{Levels: levels})
}

func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
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
}

// tokenTag abbreviates a session token so full tokens never reach the logs.
func tokenTag(token string) string {
	const n = 6
	if len(token) <= n {
		return token
	}
	return token[:n] + "…"
}
