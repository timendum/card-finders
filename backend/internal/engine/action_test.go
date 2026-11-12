package engine

// Apply dispatch tests: no game, game ended, unknown action kind. Per-kind
// handler behavior lives in each handler's own test file.

import (
	"errors"
	"testing"

	"pgregory.net/rapid"
)

func TestApplyNoGame(t *testing.T) {
	e := newTestEngine()
	res, err := e.Apply(0, Action{Kind: ActionKindGuess})
	if !errors.Is(err, ErrNoGame) {
		t.Errorf("err = %v, want ErrNoGame", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}
	if e.state != nil {
		t.Error("expected no game state created by a rejected Apply")
	}
}

// TestApplyAfterEndedRejected checks that a rejected post-end action returns the
// recorded terminal result (not ResultNone) and mutates nothing.
func TestApplyAfterEndedRejected(t *testing.T) {
	e := newTestEngine()
	if _, err := e.CreateGame(1); err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	e.state.Phase = PhaseEnded
	e.state.GameResult = ResultLoss

	actions := []Action{ // one of each kind
		{Kind: ActionKindGuess},
		{Kind: ActionKindSafeCall},
		{Kind: ActionKindSelfReveal},
		{Kind: ActionKindOpeningReveal},
		{Kind: ActionKindSafeChooseReveal},
	}
	for _, a := range actions {
		res, err := e.Apply(0, a)
		if !errors.Is(err, ErrGameEnded) {
			t.Errorf("Apply(kind=%d) after ended: err = %v, want ErrGameEnded", a.Kind, err)
		}
		if res != ResultLoss {
			t.Errorf("Apply(kind=%d) after ended: result = %d, want ResultLoss unchanged", a.Kind, res)
		}
	}
	if e.state.Phase != PhaseEnded {
		t.Errorf("phase = %d, want PhaseEnded unchanged", e.state.Phase)
	}
	if e.state.GameResult != ResultLoss {
		t.Errorf("result = %d, want ResultLoss unchanged", e.state.GameResult)
	}
}

func TestApplyUnknownActionKind(t *testing.T) {
	e := newTestEngine()
	if _, err := e.CreateGame(1); err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	e.state.Phase = PhasePlaying

	// 200 is outside the ActionKind iota range.
	res, err := e.Apply(0, Action{Kind: ActionKind(200)})
	if !errors.Is(err, ErrUnknownAction) {
		t.Errorf("err = %v, want ErrUnknownAction", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}
	if e.state.Phase != PhasePlaying {
		t.Errorf("phase = %d, want PhasePlaying unchanged", e.state.Phase)
	}
}

// TestApplyRejectsWrongLengths checks that every kind rejects a wrong number of
// cards or seats with ErrInvalidInput before any other gate.
func TestApplyRejectsWrongLengths(t *testing.T) {
	e := newTestEngine()
	if _, err := e.CreateGame(1); err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	one := []CardID{1}
	two := []CardID{1, 2}
	three := []CardID{1, 2, 3}
	seats := []SeatID{1}
	cases := []struct {
		name string
		a    Action
	}{
		{"opening none", Action{Kind: ActionKindOpeningReveal}},
		{"opening two", Action{Kind: ActionKindOpeningReveal, TargetCards: two}},
		{"guess no source", Action{Kind: ActionKindGuess, TargetCards: one}},
		{"guess two targets", Action{Kind: ActionKindGuess, SourceCards: one, TargetCards: two}},
		{"safeCall one target", Action{Kind: ActionKindSafeCall, SourceCards: one, TargetCards: one}},
		{"safeCall three targets", Action{Kind: ActionKindSafeCall, SourceCards: one, TargetCards: three}},
		{"choose none", Action{Kind: ActionKindSafeChooseReveal}},
		{"choose two", Action{Kind: ActionKindSafeChooseReveal, TargetCards: two}},
		{"triple call two targets", Action{Kind: ActionKindJoker, Joker: JokerTripleCall, SourceCards: one, TargetCards: two}},
		{"reverse call one source", Action{Kind: ActionKindJoker, Joker: JokerReverseCall, SourceCards: one, TargetCards: one}},
		{"pass no seat", Action{Kind: ActionKindJoker, Joker: JokerPass}},
		{"pass two seats", Action{Kind: ActionKindJoker, Joker: JokerPass, TargetSeats: []SeatID{1, 2}}},
		{"second wind three seats", Action{Kind: ActionKindJoker, Joker: JokerSecondWind, TargetSeats: []SeatID{1, 2, 3}}},
		{"shield with seat", Action{Kind: ActionKindJoker, Joker: JokerShield, TargetSeats: seats}},
	}
	for _, c := range cases {
		if _, err := e.Apply(0, c.a); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", c.name, err)
		}
	}
}

// Feature: frontend-rework, Property 6: Seq is monotonic.
func TestPropSeqMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var lastSeq [seatCount]uint64
		var prev uint64
		runRandomGame(t, gameObs{
			onApply: func(t *rapid.T, e *Engine, seat SeatID, a Action, err error) {
				want := prev + 1
				if err != nil {
					want = prev
				}
				if e.state.Seq != want {
					t.Fatalf("Seq = %d after apply (err=%v), want %d", e.state.Seq, err, want)
				}
				prev = e.state.Seq
			},
			onState: func(t *rapid.T, e *Engine) {
				for v := range SeatID(seatCount) {
					la := e.View(v).LastAction
					if la == nil {
						continue
					}
					if la.Seq < lastSeq[v] {
						t.Fatalf("viewer %d Seq went backwards: %d -> %d", v, lastSeq[v], la.Seq)
					}
					if la.Seq != e.state.Seq {
						t.Fatalf("viewer %d LastAction.Seq %d != state Seq %d", v, la.Seq, e.state.Seq)
					}
					lastSeq[v] = la.Seq
				}
			},
		})
	})
}
