package engine

import (
	"errors"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// safeCallState builds a PhasePlaying engine with seat 0 acting and all four
// seats holding an unused Safe_Call power.
func safeCallState(lives int) *Engine {
	e := playingState(lives)
	for i := range 4 {
		e.state.Players[i] = &Player{Seat: SeatID(i)}
	}
	return e
}

// One match: source and the matching target flip, the other is untouched, power
// spent, turn passes.
func TestSafeCallOneMatchHit(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // seat 0 source: suited 7
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 target A: match
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone}) // seat 1 target B: no match

	res, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	src, matched, other := e.state.Cards[1], e.state.Cards[2], e.state.Cards[3]
	if src.Facing != FaceUp || matched.Facing != FaceUp {
		t.Errorf("facings = src:%d matched:%d, want both FaceUp (Pair_Reveal)", src.Facing, matched.Facing)
	}
	if src.Token != nil || matched.Token != nil {
		t.Errorf("tokens = src:%+v matched:%+v, want both nil on reveal", src.Token, matched.Token)
	}
	if other.Facing != FaceDown || other.Token != nil {
		t.Errorf("non-matching target mutated: facing=%d token=%+v, want untouched", other.Facing, other.Token)
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged on a hit", e.state.LifePool)
	}
	if !e.state.Players[0].PowerUsed {
		t.Errorf("PowerUsed = false, want true after a safe call")
	}
	if e.state.PendingSafeChoice != nil {
		t.Errorf("PendingSafeChoice = %+v, want nil on a single-match hit", e.state.PendingSafeChoice)
	}
	if e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("CurrentSeat = %d, want %d (turn passed)", e.state.CurrentSeat, nextSeat(0))
	}
}

// Both match: power spent and a pending choice recorded, but no card flips and
// the turn does not pass until ActionKindSafeChooseReveal.
func TestSafeCallBothMatchPending(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // seat 0 source: suited 7
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 target A: match
	place(e, 3, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 target B: match

	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}

	if !e.state.Players[0].PowerUsed {
		t.Errorf("PowerUsed = false, want true (power spent on every valid call)")
	}
	p := e.state.PendingSafeChoice
	if p == nil {
		t.Fatalf("PendingSafeChoice = nil, want a recorded choice")
	}
	if p.Caller != 0 || p.TargetPlayer != 1 || p.SourceCard != 1 {
		t.Errorf("pending = %+v, want caller 0 / targetPlayer 1 / source 1", p)
	}
	if !((p.Options[0] == 2 && p.Options[1] == 3) || (p.Options[0] == 3 && p.Options[1] == 2)) {
		t.Errorf("pending options = %d,%d, want the two matching targets 2 and 3", p.Options[0], p.Options[1])
	}
	for _, id := range []CardID{1, 2, 3} {
		if c := e.state.Cards[id]; c.Facing != FaceDown {
			t.Errorf("card %d flipped early: facing=%d, want FaceDown until the choice", id, c.Facing)
		}
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged on a hit", e.state.LifePool)
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 (turn NOT passed until the choice)", e.state.CurrentSeat)
	}
}

// No match: life -1, each target tokened face-down, source untouched, power
// spent, turn passes.
func TestSafeCallNoMatchMiss(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})  // seat 0 source: suited 7
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})  // seat 1 target A: no match
	place(e, 3, 1, Card{Rank: 11, Kind: SpecialKindNone}) // seat 1 target B: no match

	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}

	if e.state.LifePool != 3 {
		t.Errorf("LifePool = %d, want 3 (-1 on a miss)", e.state.LifePool)
	}
	// The token and turn pass wait for the target's choice (see
	// TestSafeCallDoubleMissTargetChoosesToken).
	for _, id := range []CardID{1, 2, 3} {
		if c := e.state.Cards[id]; c.Facing != FaceDown || c.Token != nil {
			t.Errorf("card %d = facing:%d token:%+v, want untouched until the choice", id, c.Facing, c.Token)
		}
	}
	if !e.state.Players[0].PowerUsed {
		t.Errorf("PowerUsed = false, want true after a safe call")
	}
	if e.state.PendingSafeChoice == nil || !e.state.PendingSafeChoice.Miss {
		t.Errorf("PendingSafeChoice = %+v, want a Miss choice", e.state.PendingSafeChoice)
	}
}

// A miss never drives the Life_Pool below zero.
func TestSafeCallMissLifeClampedAtZero(t *testing.T) {
	e := safeCallState(0)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 11, Kind: SpecialKindNone})

	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.LifePool != 0 {
		t.Errorf("LifePool = %d, want 0 clamped", e.state.LifePool)
	}
}

// assertSafeUnchanged checks that a rejected safe call mutated nothing the
// handler touches.
func assertSafeUnchanged(t *testing.T, e *Engine, src, t1, t2 CardID, wantLife int) {
	t.Helper()
	for _, id := range []CardID{src, t1, t2} {
		if c := e.state.Cards[id]; c != nil && (c.Facing != FaceDown || c.Token != nil) {
			t.Errorf("card %d mutated by rejected safe call: facing=%d token=%+v", id, c.Facing, c.Token)
		}
	}
	if e.state.LifePool != wantLife {
		t.Errorf("LifePool = %d, want %d unchanged on rejection", e.state.LifePool, wantLife)
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 unchanged on rejection", e.state.CurrentSeat)
	}
	if e.state.PendingSafeChoice != nil {
		t.Errorf("PendingSafeChoice = %+v, want nil on rejection", e.state.PendingSafeChoice)
	}
	if e.state.Players[0].PowerUsed {
		t.Errorf("PowerUsed = true, want false unchanged on rejection")
	}
}

// An already-spent power is rejected with ErrSafeCallSpent.
func TestSafeCallSpentRejected(t *testing.T) {
	e := safeCallState(4)
	e.state.Players[0].PowerUsed = true
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrSafeCallSpent) {
		t.Errorf("err = %v, want ErrSafeCallSpent", err)
	}
	for _, id := range []CardID{1, 2, 3} {
		if c := e.state.Cards[id]; c.Facing != FaceDown || c.Token != nil {
			t.Errorf("card %d mutated by rejected safe call", id)
		}
	}
	if e.state.LifePool != 4 || e.state.CurrentSeat != 0 || e.state.PendingSafeChoice != nil {
		t.Errorf("state mutated by rejected safe call: life=%d seat=%d pending=%v", e.state.LifePool, e.state.CurrentSeat, e.state.PendingSafeChoice)
	}
}

// A gold source is rejected with ErrGoldSource.
func TestSafeCallGoldSourceRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindGold})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrGoldSource) {
		t.Errorf("err = %v, want ErrGoldSource", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// A source the seat does not own is rejected with ErrInvalidSource.
func TestSafeCallNonOwnedSourceRejected(t *testing.T) {
	e := safeCallState(4)
	// Seat 0 holds a face-down card so it is eligible, isolating source validation.
	place(e, 10, 0, Card{Rank: 3, Kind: SpecialKindNone})
	place(e, 1, 1, Card{Rank: 7, Kind: SpecialKindNone}) // owned by seat 1, not the actor
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrInvalidSource) {
		t.Errorf("err = %v, want ErrInvalidSource", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// Two identical targets are rejected with ErrSafeCallTargets.
func TestSafeCallNonDistinctTargetsRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 2}})
	if !errors.Is(err, ErrSafeCallTargets) {
		t.Errorf("err = %v, want ErrSafeCallTargets", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// Targets owned by different players are rejected with ErrSafeCallTargets.
func TestSafeCallSplitOwnerTargetsRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1
	place(e, 3, 2, Card{Rank: 9, Kind: SpecialKindNone}) // seat 2 (different owner)

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrSafeCallTargets) {
		t.Errorf("err = %v, want ErrSafeCallTargets", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// Targeting the caller's own cards is rejected with ErrSafeCallTargets.
func TestSafeCallOwnTargetsRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone}) // caller's own
	place(e, 3, 0, Card{Rank: 9, Kind: SpecialKindNone}) // caller's own

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrSafeCallTargets) {
		t.Errorf("err = %v, want ErrSafeCallTargets", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// A face-up target is rejected with ErrSafeCallTargets.
func TestSafeCallRevealedTargetRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})
	e.state.Cards[2].Facing = FaceUp

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrSafeCallTargets) {
		t.Errorf("err = %v, want ErrSafeCallTargets", err)
	}
	if e.state.LifePool != 4 || e.state.CurrentSeat != 0 || e.state.Players[0].PowerUsed {
		t.Errorf("state mutated by rejected safe call")
	}
}

// A target player with fewer than two face-down cards is rejected with ErrSafeCallTargets.
func TestSafeCallTooFewFaceDownRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})
	e.state.Cards[3].Facing = FaceUp // seat 1 now has only one face-down card

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrSafeCallTargets) {
		t.Errorf("err = %v, want ErrSafeCallTargets", err)
	}
}

// A safe call outside PhasePlaying is rejected with ErrWrongPhase.
func TestSafeCallWrongPhaseRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})
	e.state.Phase = PhaseOpening

	_, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrWrongPhase) {
		t.Errorf("err = %v, want ErrWrongPhase", err)
	}
	assertSafeUnchanged(t, e, 1, 2, 3, 4)
}

// A safe call out of turn is rejected with ErrNotYourTurn.
func TestSafeCallOutOfTurnRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1's source
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 0, Card{Rank: 9, Kind: SpecialKindNone})

	_, err := e.Apply(1, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
	if e.state.Players[1].PowerUsed {
		t.Errorf("PowerUsed = true for seat 1, want false unchanged on rejection")
	}
}

// Two gold targets end the game as a loss; both flip, no life is lost.
func TestSafeCallBothGoldEndsGame(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindGold})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindGold})

	res, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if err != nil || res != ResultLoss || e.state.Phase != PhaseEnded {
		t.Fatalf("res=%d err=%v phase=%d, want loss/Ended", res, err, e.state.Phase)
	}
	if e.state.Cards[2].Facing != FaceUp || e.state.Cards[3].Facing != FaceUp {
		t.Errorf("gold targets must flip face-up")
	}
	if e.state.LifePool != 4 || e.state.Cards[1].Facing != FaceDown {
		t.Errorf("life=%d source=%d, want 4 / FaceDown", e.state.LifePool, e.state.Cards[1].Facing)
	}
}

// --- Safe_Call choose-reveal (double-hit second step) ---

// A valid choice completes the deferred Pair_Reveal (source + chosen target flip,
// no token), leaves the other option untouched, clears the pending choice, and
// passes the turn from the caller.
func TestSafeChooseRevealResolves(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // seat 0 source: suited 7
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 target A: match
	place(e, 3, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1 target B: match

	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}}); err != nil {
		t.Fatalf("safe call: unexpected error %v", err)
	}
	if e.state.PendingSafeChoice == nil {
		t.Fatalf("expected a pending choice after a both-match safe call")
	}

	res, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{2}})
	if err != nil {
		t.Fatalf("choose-reveal: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	src, chosen, other := e.state.Cards[1], e.state.Cards[2], e.state.Cards[3]
	if src.Facing != FaceUp || src.Token != nil {
		t.Errorf("source = facing:%d token:%+v, want FaceUp with no token (Pair_Reveal)", src.Facing, src.Token)
	}
	if chosen.Facing != FaceUp || chosen.Token != nil {
		t.Errorf("chosen = facing:%d token:%+v, want FaceUp with no token (Pair_Reveal)", chosen.Facing, chosen.Token)
	}
	if other.Facing != FaceDown || other.Token != nil {
		t.Errorf("unchosen = facing:%d token:%+v, want untouched", other.Facing, other.Token)
	}
	if e.state.PendingSafeChoice != nil {
		t.Errorf("PendingSafeChoice = %+v, want nil after the choice resolves", e.state.PendingSafeChoice)
	}
	if e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("CurrentSeat = %d, want %d (turn passes from the caller)", e.state.CurrentSeat, nextSeat(0))
	}
}

// pendingState builds a both-match safe call resolved to a pending choice: seat
// 0 called against seat 1's two matching cards (2 and 3).
func pendingState() *Engine {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 7, Kind: SpecialKindNone})
	e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	return e
}

// assertPendingUnchanged checks that a rejected choose-reveal mutated nothing:
// cards stay face-down and untokened, the pending choice remains, turn unchanged.
func assertPendingUnchanged(t *testing.T, e *Engine) {
	t.Helper()
	for _, id := range []CardID{1, 2, 3} {
		if c := e.state.Cards[id]; c.Facing != FaceDown || c.Token != nil {
			t.Errorf("card %d mutated by rejected choose-reveal: facing=%d token=%+v", id, c.Facing, c.Token)
		}
	}
	if e.state.PendingSafeChoice == nil {
		t.Errorf("PendingSafeChoice = nil, want it preserved on rejection")
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 unchanged on rejection", e.state.CurrentSeat)
	}
}

// A chosen card that is neither pending option is rejected with ErrInvalidChoice.
func TestSafeChooseRevealNonOptionRejected(t *testing.T) {
	e := pendingState()
	place(e, 9, 1, Card{Rank: 7, Kind: SpecialKindNone}) // another seat-1 card, not an option

	_, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{9}})
	if !errors.Is(err, ErrInvalidChoice) {
		t.Errorf("err = %v, want ErrInvalidChoice", err)
	}
	assertPendingUnchanged(t, e)
}

// A seat other than the targeted player is rejected with ErrNotYourTurn.
func TestSafeChooseRevealWrongSeatRejected(t *testing.T) {
	e := pendingState()

	_, err := e.Apply(0, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{2}})
	if !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
	assertPendingUnchanged(t, e)
}

// A choose-reveal outside AwaitingResponse is rejected with ErrWrongPhase.
func TestSafeChooseRevealNoPendingRejected(t *testing.T) {
	e := safeCallState(4)
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{2}})
	if !errors.Is(err, ErrWrongPhase) {
		t.Errorf("err = %v, want ErrWrongPhase", err)
	}
	if e.state.PendingSafeChoice != nil {
		t.Errorf("PendingSafeChoice = %+v, want nil", e.state.PendingSafeChoice)
	}
	if c := e.state.Cards[2]; c.Facing != FaceDown || c.Token != nil {
		t.Errorf("card 2 mutated by rejected choose-reveal: facing=%d token=%+v", c.Facing, c.Token)
	}
}

// One gold + one match: the match reveals as a pair, gold is untouched, no
// game over, no life lost.
func TestSafeCallGoldAndMatchReveals(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindGold})

	res, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if err != nil || res != ResultNone {
		t.Fatalf("res=%d err=%v, want ResultNone", res, err)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[2].Facing != FaceUp {
		t.Errorf("source and match must flip")
	}
	if g := e.state.Cards[3]; g.Facing != FaceDown || g.Token != nil {
		t.Errorf("gold mutated: %+v", g)
	}
	if e.state.LifePool != 4 || e.state.Phase == PhaseEnded {
		t.Errorf("life=%d phase=%d, want 4 / not ended", e.state.LifePool, e.state.Phase)
	}
}

// One gold + one wrong non-gold: lose a life, only the non-gold is tokened, no
// choice is asked and the turn passes.
func TestSafeCallGoldAndMissTokensNonGold(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindGold})
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	res, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if err != nil || res != ResultNone {
		t.Fatalf("res=%d err=%v, want ResultNone", res, err)
	}
	if e.state.Cards[2].Token != nil || e.state.Cards[3].Token == nil {
		t.Errorf("tokens gold=%v nonGold=%v, want only the non-gold tokened",
			e.state.Cards[2].Token, e.state.Cards[3].Token)
	}
	if e.state.LifePool != 3 || e.state.PendingSafeChoice != nil || e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("life=%d pending=%v seat=%d", e.state.LifePool, e.state.PendingSafeChoice, e.state.CurrentSeat)
	}
}

// Double non-gold miss: the life is lost at call time, the target picks the one
// card that gets a token, nothing flips, then the turn passes.
func TestSafeCallDoubleMissTargetChoosesToken(t *testing.T) {
	e := safeCallState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 11, Kind: SpecialKindNone})

	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}}); err != nil {
		t.Fatalf("safe call: %v", err)
	}
	p := e.state.PendingSafeChoice
	if p == nil || !p.Miss || e.state.Phase != PhaseAwaitingResponse {
		t.Fatalf("pending=%+v phase=%d, want a Miss choice awaiting response", p, e.state.Phase)
	}
	if e.state.LifePool != 3 || e.state.Cards[2].Token != nil || e.state.Cards[3].Token != nil {
		t.Errorf("life=%d, tokens must wait for the choice", e.state.LifePool)
	}
	if _, err := e.Apply(0, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{2}}); !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("caller choosing: err=%v, want ErrNotYourTurn", err)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{3}}); err != nil {
		t.Fatalf("choose: %v", err)
	}
	if e.state.Cards[3].Token == nil || e.state.Cards[2].Token != nil {
		t.Errorf("only the chosen card must be tokened")
	}
	if e.state.Cards[1].Facing != FaceDown || e.state.Cards[3].Facing != FaceDown {
		t.Errorf("nothing may flip on a double miss")
	}
	if e.state.PendingSafeChoice != nil || e.state.Phase != PhasePlaying || e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("pending=%v phase=%d seat=%d", e.state.PendingSafeChoice, e.state.Phase, e.state.CurrentSeat)
	}
}

// A double miss on the last life ends the game at once; no choice is asked.
func TestSafeCallDoubleMissLastLifeEnds(t *testing.T) {
	e := safeCallState(1)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})
	place(e, 3, 1, Card{Rank: 11, Kind: SpecialKindNone})

	res, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{1}, TargetCards: []CardID{2, 3}})
	if err != nil || res != ResultLoss || e.state.PendingSafeChoice != nil {
		t.Errorf("res=%d err=%v pending=%v, want loss and no pending choice", res, err, e.state.PendingSafeChoice)
	}
}

func allViews(e *Engine) [seatCount]PlayerView {
	var vs [seatCount]PlayerView
	for i := range vs {
		vs[i] = e.View(SeatID(i))
	}
	return vs
}

// Feature: frontend-rework, Property 7: AwaitingResponse gates all but the awaited response.
func TestPropAwaitingResponseGating(t *testing.T) {
	visits := 0
	rapid.Check(t, func(t *rapid.T) {
		runRandomGame(t, gameObs{onState: func(t *rapid.T, e *Engine) {
			s := e.state
			vs := allViews(e)
			if s.Phase != PhaseAwaitingResponse {
				for i, v := range vs {
					if v.HasResponsePlayer {
						t.Fatalf("seat %d HasResponsePlayer outside AwaitingResponse (phase %d)", i, s.Phase)
					}
				}
				return
			}
			visits++
			p := s.PendingSafeChoice
			if p == nil || s.CurrentSeat != p.Caller {
				t.Fatalf("AwaitingResponse needs a pending choice with CurrentSeat the caller")
			}
			for i, v := range vs {
				if v.YourTurn || !v.HasResponsePlayer || v.ResponsePlayer != p.TargetPlayer || v.CurrentSeat != p.Caller {
					t.Fatalf("seat %d view wrong in AwaitingResponse: %+v", i, v)
				}
			}
			probeGated(t, e, p)
		}})
	})
	if visits == 0 {
		t.Fatal("no run reached AwaitingResponse")
	}
}

// probeGated applies every non-awaited action from every seat and asserts each
// is rejected as a no-op, then checks the chooser's invalid choices too.
func probeGated(t *rapid.T, e *Engine, p *PendingSafeChoice) {
	own := hiddenNonGold(e, p.Caller)
	other := hiddenNonGold(e, p.TargetPlayer)
	bad := CardID(65535) // never a real id
	reject := func(seat SeatID, a Action, want error) {
		before := allViews(e)
		_, err := e.Apply(seat, a)
		if !errors.Is(err, want) {
			t.Fatalf("seat %d action %+v: err = %v, want %v", seat, a, err, want)
		}
		if !reflect.DeepEqual(before, allViews(e)) || e.state.Phase != PhaseAwaitingResponse {
			t.Fatalf("seat %d rejected action %+v changed state", seat, a)
		}
	}
	for seat := range SeatID(seatCount) {
		reject(seat, Action{Kind: ActionKindOpeningReveal, TargetCards: []CardID{own[0]}}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindGuess, SourceCards: []CardID{own[0]}, TargetCards: []CardID{other[0]}}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindSafeCall, SourceCards: []CardID{own[0]}, TargetCards: []CardID{other[0], other[1]}}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindSelfReveal, SelfRevealValue: SelfRevealTarget{Kind: SpecialKindGold}}, ErrWrongPhase)
		if seat != p.TargetPlayer {
			reject(seat, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{p.Options[0]}}, ErrNotYourTurn)
		}
	}
	reject(p.TargetPlayer, Action{Kind: ActionKindSafeChooseReveal, TargetCards: []CardID{bad}}, ErrInvalidChoice)
}
