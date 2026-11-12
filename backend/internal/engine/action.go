package engine

import (
	"context"
	"log/slog"
	"math"
)

// ActionKind is the discriminant of the Action tagged union; it selects which
// of Action's fields are meaningful.
type ActionKind uint8

const (
	ActionKindOpeningReveal ActionKind = iota
	ActionKindGuess
	ActionKindSafeCall
	ActionKindSafeChooseReveal
	ActionKindSelfReveal
	ActionKindJoker
)

// SelfRevealTarget names the value chosen to self-reveal: Kind SpecialKindNone
// uses Rank (suited rank); SpecialKindSilver / SpecialKindGold ignore Rank.
type SelfRevealTarget struct {
	Kind SpecialKind
	Rank Rank
}

// Action is the tagged union of every gameplay action. Kind selects the active
// action; each applyX documents and checks the fields it reads. The slices are
// read-only to the engine.
type Action struct {
	Kind        ActionKind
	SourceCards []CardID
	TargetCards []CardID
	TargetSeats []SeatID
	// SelfReveal and Census: the chosen value.
	SelfRevealValue SelfRevealTarget
	// Joker: the rank activated.
	Joker Rank
}

// lenRange is the accepted length interval of one Action slice.
type lenRange struct{ min, max int }

var anyLen = lenRange{0, math.MaxInt}

func exactly(n int) lenRange { return lenRange{n, n} }

func (r lenRange) holds(n int) bool { return n >= r.min && n <= r.max }

// checkLens rejects an action whose slices fall outside the given lengths.
func (a Action) checkLens(src, tgt, seats lenRange) error {
	if !src.holds(len(a.SourceCards)) || !tgt.holds(len(a.TargetCards)) ||
		!seats.holds(len(a.TargetSeats)) {
		return ErrInvalidInput
	}
	return nil
}

// Apply is the single entry point for all gameplay actions. It returns the
// terminal Result (ResultNone while the game continues) or a typed sentinel
// error. Per-kind handlers run their own phase/turn validation.
func (e *Engine) Apply(seat SeatID, action Action) (Result, error) {
	res, err := e.apply(seat, action)
	e.logAppliedState(seat, action, res, err)
	return res, err
}

// logAppliedState emits a debug record of the state after an action. It reads
// only public-safe summary fields (no hidden card values).
func (e *Engine) logAppliedState(seat SeatID, action Action, res Result, err error) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	attrs := []any{
		"seat", seat,
		"actionKind", action.Kind,
		"result", res,
		"err", err,
	}
	if s := e.state; s != nil {
		faceDown := 0
		for _, pc := range s.Cards {
			if pc.Facing == FaceDown {
				faceDown++
			}
		}
		attrs = append(attrs,
			"phase", s.Phase,
			"seq", s.Seq,
			"currentSeat", s.CurrentSeat,
			"lifePool", s.LifePool,
			"faceDown", faceDown,
			"openingDone", s.OpeningDone,
			"pendingSafeChoice", s.PendingSafeChoice != nil,
			"gameResult", s.GameResult,
		)
	}
	slog.Debug("action applied", attrs...)
}

func (e *Engine) apply(seat SeatID, action Action) (Result, error) {
	if e.state == nil {
		return ResultNone, ErrNoGame
	}
	if e.state.Phase == PhaseEnded {
		return e.state.GameResult, ErrGameEnded
	}

	switch action.Kind {
	case ActionKindOpeningReveal:
		return e.applyOpeningReveal(seat, action)
	case ActionKindGuess:
		return e.applyGuess(seat, action)
	case ActionKindSafeCall:
		return e.applySafeCall(seat, action)
	case ActionKindSafeChooseReveal:
		return e.applySafeChooseReveal(seat, action)
	case ActionKindSelfReveal:
		return e.applySelfReveal(seat, action)
	case ActionKindJoker:
		return e.applyJoker(seat, action)
	default:
		return ResultNone, ErrUnknownAction
	}
}
