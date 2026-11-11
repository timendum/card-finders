package engine

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"pgregory.net/rapid"
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

// remainingRanks returns the eligible special ranks not in used, ascending.
func remainingRanks(used []Rank) []Rank {
	taken := make(map[Rank]bool, len(used))
	for _, r := range used {
		taken[r] = true
	}
	var out []Rank
	for _, r := range eligibleSpecialRankList() {
		if !taken[r] {
			out = append(out, r)
		}
	}
	return out
}

// overwriteDecoys replaces e's decoy ranks directly with a random subset of the
// ranks not used by a real special of the same kind. It does not touch RNG or
// hands, so two engines with equal hands can differ only in their decoys.
func overwriteDecoys(t *rapid.T, e *Engine, label string) {
	p := &e.state.SpecialPool
	pick := func(used []Rank, name string) []Rank {
		rest := remainingRanks(used)
		if len(rest) == 0 {
			return nil
		}
		return rapid.SliceOfNDistinct(rapid.SampledFrom(rest), 0, len(rest),
			func(r Rank) Rank { return r }).Draw(t, label+name)
	}
	p.DecoySilvers = pick(cardRanks(p.Silvers), "Silvers")
	p.DecoyGolds = pick(cardRanks(p.Golds), "Golds")
}

func snapshotState(t *rapid.T, e *Engine) string {
	b, err := json.Marshal(e.state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	return string(b)
}

// allSelfRevealTargets lists every self-reveal value a client could submit.
func allSelfRevealTargets() []SelfRevealTarget {
	out := []SelfRevealTarget{{Kind: SpecialKindGold}, {Kind: SpecialKindSilver}}
	for r := Rank(2); r <= 13; r++ {
		out = append(out, SelfRevealTarget{Kind: SpecialKindNone, Rank: r})
	}
	return out
}

// Feature: card-finders, Property 7: self-reveal eligibility ignores decoys
// (two states with identical hands but different decoys accept and reject the
// same self-reveals; a rejection leaves state unchanged).
func TestPropSelfRevealIgnoresDecoys(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevel(t) // zero extras: same seed => identical hands
		seed := rapid.Int64().Draw(t, "seed")
		a := newDealtEngine(t, seed, lvl)
		b := newDealtEngine(t, seed, lvl)
		overwriteDecoys(t, a, "decoyA")
		overwriteDecoys(t, b, "decoyB")

		if !reflect.DeepEqual(a.state.HandOrder, b.state.HandOrder) {
			t.Fatalf("setup: hands differ")
		}

		// Each action either pairs two cards or costs a life. The slack covers
		// the four opening reveals, the second step of each seat's one safe
		// call, and each seat's one single-card gold self-reveal.
		maxSteps := len(a.state.Cards)/2 + lvl.StartLives + 12
		for step := 0; a.state.Phase != PhaseEnded; step++ {
			if step >= maxSteps {
				t.Fatalf("game not ended after %d steps", maxSteps)
			}
			probeSelfReveals(t, a, b)
			if a.state.Phase == PhaseEnded {
				break
			}
			seat, act := pickAction(t, a)
			ra, errA := a.Apply(seat, act)
			rb, errB := b.Apply(seat, act)
			if errA != errB {
				t.Fatalf("action %+v: error %v vs %v", act, errA, errB)
			}
			if errA != nil {
				t.Fatalf("legal action %+v rejected: %v", act, errA)
			}
			if ra != rb {
				t.Fatalf("action %+v: result %v vs %v", act, ra, rb)
			}
		}
	})
}

// probeSelfReveals submits every self-reveal value for the current seat (the
// one pickAction acts for) to both engines. Outcomes must match; rejections
// must leave state untouched. When an action is accepted (identically by both)
// the state has moved on, so probing stops for this round.
func probeSelfReveals(t *rapid.T, a, b *Engine) {
	seat := a.state.CurrentSeat
	// Rejections never mutate, so one snapshot serves every probe until an
	// accept, which ends the round.
	beforeA, beforeB := snapshotState(t, a), snapshotState(t, b)
	for _, tgt := range allSelfRevealTargets() {
		act := Action{Kind: ActionKindSelfReveal, SelfRevealValue: tgt}
		ra, errA := a.Apply(seat, act)
		rb, errB := b.Apply(seat, act)
		if errA != errB {
			t.Fatalf("seat %d self-reveal %+v: error %v vs %v", seat, tgt, errA, errB)
		}
		if errA != nil {
			if snapshotState(t, a) != beforeA || snapshotState(t, b) != beforeB {
				t.Fatalf("seat %d self-reveal %+v rejected (%v) but changed state", seat, tgt, errA)
			}
			continue
		}
		if ra != rb {
			t.Fatalf("seat %d self-reveal %+v: result %v vs %v", seat, tgt, ra, rb)
		}
		return
	}
}
