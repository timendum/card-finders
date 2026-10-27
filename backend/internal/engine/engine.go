// Package engine is the pure, deterministic Card Finders game engine.
package engine

import "math/rand"

// Engine is the game state machine. rng seeds pool formation, dealing, and
// first-player selection; state is nil until a game is created.
type Engine struct {
	levels []Level
	rng    *rand.Rand
	state  *GameState
}

func NewEngine(levels []Level, rng *rand.Rand) *Engine {
	return &Engine{
		levels: levels,
		rng:    rng,
	}
}

// Levels returns a defensive copy, so callers cannot mutate engine state.
func (e *Engine) Levels() []Level {
	out := make([]Level, len(e.levels))
	copy(out, e.levels)
	return out
}
