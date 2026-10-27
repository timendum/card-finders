// Package server is the HTTP/WebSocket transport layer. The static file server
// is a convenience, not a dependency of the API: an operator may serve the
// frontend elsewhere and point it at /api and /ws.
package server

import (
	"cardfinders/internal/engine"
	"net/http"
	"sync"
)

type Config struct {
	// StaticDir holds the built frontend, served from disk at request time;
	// empty leaves it unmounted.
	StaticDir string
}

type Server struct {
	mu     sync.Mutex
	engine *engine.Engine
	cfg    Config
	hub    *hub
}

func New(eng *engine.Engine, cfg Config) *Server {
	return &Server{engine: eng, cfg: cfg, hub: newHub()}
}

func (s *Server) Handler() *http.ServeMux {
	mux := http.NewServeMux()

	// Each handler enforces its method, so a wrong method yields 405 not a miss.
	mux.HandleFunc("/api/levels", s.handleLevels)
	mux.HandleFunc("/api/game", s.handleCreateGame)
	mux.HandleFunc("/api/game/join", s.handleJoin)

	mux.HandleFunc("/ws", s.serveWS)

	if s.cfg.StaticDir != "" {
		// Catch-all; the more specific /api and /ws patterns above win.
		mux.Handle("/", http.FileServer(http.Dir(s.cfg.StaticDir)))
	}

	return mux
}
