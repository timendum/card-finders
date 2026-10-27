package engine

import (
	"errors"
	"testing"
)

func selfReveal(kind SpecialKind, rank Rank) Action {
	return Action{Kind: ActionKindSelfReveal, SelfRevealValue: SelfRevealTarget{Kind: kind, Rank: rank}}
}

// assertSelfRevealUnchanged checks a rejected self-reveal left the two cards
// face-down and untokened, life unchanged, and the turn with seat 0.
func assertSelfRevealUnchanged(t *testing.T, e *Engine, a, b CardID, wantLife int) {
	t.Helper()
	for _, id := range []CardID{a, b} {
		if pc := e.state.Cards[id]; pc != nil && (pc.Facing != FaceDown || pc.Token != nil) {
			t.Errorf("card %d mutated by rejected self-reveal: facing=%d token=%+v", id, pc.Facing, pc.Token)
		}
	}
	if e.state.LifePool != wantLife {
		t.Errorf("LifePool = %d, want %d unchanged", e.state.LifePool, wantLife)
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 unchanged", e.state.CurrentSeat)
	}
}

// TestSelfRevealSuitedFlipsOwnValueNoLifeChange: a suited-rank self-reveal flips
// only the seat's own cards of that rank, costs no life, and passes the turn.
func TestSelfRevealSuitedFlipsOwnValueNoLifeChange(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // own 7s, revealed
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 0, Card{Rank: 9, Kind: SpecialKindNone}) // own non-7, untouched
	place(e, 4, 1, Card{Rank: 5, Kind: SpecialKindNone}) // keeps the game going

	res, err := e.Apply(0, selfReveal(SpecialKindNone, 7))
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[2].Facing != FaceUp {
		t.Errorf("own 7s not flipped face-up")
	}
	if e.state.Cards[3].Facing != FaceDown {
		t.Errorf("non-matching own card flipped, want untouched")
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged on self-reveal", e.state.LifePool)
	}
	if e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("CurrentSeat = %d, want %d (turn passed)", e.state.CurrentSeat, nextSeat(0))
	}
}

// TestSelfRevealSilverFlipsAllOwnSilver: a silver self-reveal flips every silver
// the seat owns regardless of rank, leaving a suited card of its own untouched.
func TestSelfRevealSilverFlipsAllOwnSilver(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 3, Kind: SpecialKindSilver}) // own silver, differing ranks
	place(e, 2, 0, Card{Rank: 8, Kind: SpecialKindSilver})
	place(e, 3, 0, Card{Rank: 8, Kind: SpecialKindNone}) // own suited, untouched
	place(e, 4, 1, Card{Rank: 5, Kind: SpecialKindNone}) // keeps the game going

	if _, err := e.Apply(0, selfReveal(SpecialKindSilver, 0)); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[2].Facing != FaceUp {
		t.Errorf("own silver not all flipped face-up")
	}
	if e.state.Cards[3].Facing != FaceDown {
		t.Errorf("suited card flipped by silver self-reveal, want untouched")
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged", e.state.LifePool)
	}
}

// TestSelfRevealGoldFlipsOwnGold: a gold self-reveal, legal only when the seat's
// remaining face-down cards are all gold, flips them face-up with no life change.
func TestSelfRevealGoldFlipsOwnGold(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 4, Kind: SpecialKindGold}) // seat 0 holds only gold
	place(e, 2, 0, Card{Rank: 10, Kind: SpecialKindGold})
	place(e, 3, 1, Card{Rank: 5, Kind: SpecialKindNone}) // keeps the game going

	if _, err := e.Apply(0, selfReveal(SpecialKindGold, 0)); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[2].Facing != FaceUp {
		t.Errorf("own gold not flipped face-up")
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged", e.state.LifePool)
	}
}

// TestSelfRevealWrongPhaseRejected: a self-reveal outside PhasePlaying is
// rejected with ErrWrongPhase.
func TestSelfRevealWrongPhaseRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone})
	e.state.Phase = PhaseOpening

	_, err := e.Apply(0, selfReveal(SpecialKindNone, 7))
	if !errors.Is(err, ErrWrongPhase) {
		t.Errorf("err = %v, want ErrWrongPhase", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 2, 4)
}

// TestSelfRevealOutOfTurnRejected: a self-reveal by a seat other than the
// current one is rejected with ErrNotYourTurn.
func TestSelfRevealOutOfTurnRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1's cards
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(1, selfReveal(SpecialKindNone, 7))
	if !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 2, 4)
}

// TestSelfRevealEmptyHandedActorRejectedAndAdvances: an empty-handed current
// seat cannot self-reveal; rejected with ErrSeatIneligible and the turn
// advances to the next eligible seat.
func TestSelfRevealEmptyHandedActorRejectedAndAdvances(t *testing.T) {
	e := playingState(4)
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone}) // seat 0 holds nothing; seat 1 does

	_, err := e.Apply(0, selfReveal(SpecialKindNone, 9))
	if !errors.Is(err, ErrSeatIneligible) {
		t.Errorf("err = %v, want ErrSeatIneligible", err)
	}
	if e.state.CurrentSeat != 1 {
		t.Errorf("CurrentSeat = %d, want 1 (advanced past ineligible actor)", e.state.CurrentSeat)
	}
}

// TestSelfRevealOtherHoldsValueRejected: a suited self-reveal is rejected with
// ErrSelfRevealIncomplete when another player still holds that value face-down.
func TestSelfRevealOtherHoldsValueRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // seat 0's 7
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 also holds a 7

	_, err := e.Apply(0, selfReveal(SpecialKindNone, 7))
	if !errors.Is(err, ErrSelfRevealIncomplete) {
		t.Errorf("err = %v, want ErrSelfRevealIncomplete", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 2, 4)
}

// TestSelfRevealSeatHoldsNoneRejected: a self-reveal for a value the seat does
// not hold is rejected with ErrSelfRevealIncomplete. Seat 0 holds a face-down
// card of another value, so this is the no-match branch, not the ineligible one.
func TestSelfRevealSeatHoldsNoneRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 9, Kind: SpecialKindNone}) // seat 0 holds a 9, asks for 7

	_, err := e.Apply(0, selfReveal(SpecialKindNone, 7))
	if !errors.Is(err, ErrSelfRevealIncomplete) {
		t.Errorf("err = %v, want ErrSelfRevealIncomplete", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 1, 4)
}

// TestSelfRevealGoldWithNonGoldHeldRejected: gold self-reveal is rejected with
// ErrGoldRevealIneligible while the seat still holds a non-gold face-down card.
func TestSelfRevealGoldWithNonGoldHeldRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 4, Kind: SpecialKindGold})
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone}) // non-gold still held

	_, err := e.Apply(0, selfReveal(SpecialKindGold, 0))
	if !errors.Is(err, ErrGoldRevealIneligible) {
		t.Errorf("err = %v, want ErrGoldRevealIneligible", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 2, 4)
}

// TestSelfRevealGoldWithNoGoldRejected: gold self-reveal is rejected with
// ErrGoldRevealIneligible when the seat holds no gold at all.
func TestSelfRevealGoldWithNoGoldRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // eligible actor, but no gold

	_, err := e.Apply(0, selfReveal(SpecialKindGold, 0))
	if !errors.Is(err, ErrGoldRevealIneligible) {
		t.Errorf("err = %v, want ErrGoldRevealIneligible", err)
	}
	assertSelfRevealUnchanged(t, e, 1, 1, 4)
}
