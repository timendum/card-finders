package engine

import (
	"errors"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// Feature: frontend-rework, Property 5: LastAction never leaks a non-actor's hidden source.
func TestPropLastActionSourcePrivacy(t *testing.T) {
	sawHidden := false
	rapid.Check(t, func(t *rapid.T) {
		runRandomGame(t, gameObs{onState: func(t *rapid.T, e *Engine) {
			auth := e.state.LastAction
			for viewer := SeatID(0); viewer < seatCount; viewer++ {
				v := e.View(viewer)
				if auth == nil {
					if v.LastAction != nil {
						t.Fatalf("viewer %d sees a LastAction before any move", viewer)
					}
					continue
				}
				got := v.LastAction
				if got == nil {
					t.Fatalf("viewer %d has no LastAction", viewer)
				}
				if auth.SourceCard == nil {
					if got.SourceCard != nil {
						t.Fatalf("viewer %d sees a source that was never recorded", viewer)
					}
					continue
				}
				faceUp := e.state.Cards[*auth.SourceCard].Facing == FaceUp
				switch {
				case viewer == auth.Actor || faceUp:
					if got.SourceCard == nil || *got.SourceCard != *auth.SourceCard {
						t.Fatalf("viewer %d (actor %d, faceUp %v) lost the source", viewer, auth.Actor, faceUp)
					}
				default:
					sawHidden = true
					if got.SourceCard != nil {
						t.Fatalf("viewer %d sees hidden source of actor %d (outcome %d)", viewer, auth.Actor, auth.Outcome)
					}
				}
			}
		}})
	})
	if !sawHidden {
		t.Fatal("no run exercised a hidden source viewed by a non-actor")
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
				for v := SeatID(0); v < seatCount; v++ {
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
	for seat := SeatID(0); seat < seatCount; seat++ {
		reject(seat, Action{Kind: ActionKindOpeningReveal, OpeningCard: own[0]}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindGuess, SourceCard: own[0], TargetCard: other[0]}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindSafeCall, SourceCard: own[0], TargetCard: other[0], TargetCard2: other[1]}, ErrWrongPhase)
		reject(seat, Action{Kind: ActionKindSelfReveal, SelfRevealValue: SelfRevealTarget{Kind: SpecialKindGold}}, ErrWrongPhase)
		if seat != p.TargetPlayer {
			reject(seat, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: p.Option1}, ErrNotYourTurn)
		}
	}
	reject(p.TargetPlayer, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: bad}, ErrInvalidChoice)
}

// Feature: frontend-rework, Property 4: Opaque card ids leak no value and preserve invariants.
func TestPropOpaqueCardIDs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevel(t)
		seed := rapid.Int64().Draw(t, "seed")
		e := newDealtEngine(t, seed, lvl)
		s := e.state

		// Unique ids: a bijection onto 0..N-1, every map key equal to its card id.
		n := baseDeckSize + lvl.SilverCount + lvl.GoldCount
		if len(s.Cards) != n {
			t.Fatalf("dealt %d cards, want %d", len(s.Cards), n)
		}
		for id, pc := range s.Cards {
			if pc.Card.ID != id || int(id) >= n {
				t.Fatalf("card key %d / id %d outside permutation 0..%d", id, pc.Card.ID, n-1)
			}
		}

		// Every pool entry id is the id of the dealt card with the same identity.
		check := func(c Card) {
			pc, ok := s.Cards[c.ID]
			if !ok || pc.Card != c {
				t.Fatalf("pool entry %+v does not match dealt card %+v", c, pc)
			}
		}
		for _, c := range s.SpecialPool.Silvers {
			check(c)
		}
		for _, c := range s.SpecialPool.Golds {
			check(c)
		}

		// Same seed, same level: identical id assignment, pool, and ownership.
		e2 := newDealtEngine(t, seed, lvl)
		if !reflect.DeepEqual(s.Cards, e2.state.Cards) ||
			!reflect.DeepEqual(s.SpecialPool, e2.state.SpecialPool) {
			t.Fatalf("two engines on seed %d assigned ids differently", seed)
		}

		// No viewer sees an opponent's face-down value.
		for viewer := SeatID(0); viewer < seatCount; viewer++ {
			for _, vc := range e.View(viewer).Cards {
				if vc.Owner != viewer && vc.Facing == FaceDown && vc.Value != nil {
					t.Fatalf("seat %d sees opponent card %d value", viewer, vc.ID)
				}
			}
		}
	})
}

// slotsByOwner groups a view's card ids by owner in Slot order, checking Slot
// is a dense 1..N permutation per owner.
func slotsByOwner(t *rapid.T, v PlayerView) [seatCount][]CardID {
	var out [seatCount][]CardID
	bySlot := [seatCount]map[int]CardID{}
	for i := range bySlot {
		bySlot[i] = map[int]CardID{}
	}
	for _, vc := range v.Cards {
		if _, dup := bySlot[vc.Owner][vc.Slot]; dup {
			t.Fatalf("owner %d has duplicate slot %d", vc.Owner, vc.Slot)
		}
		bySlot[vc.Owner][vc.Slot] = vc.ID
	}
	for o := range bySlot {
		for slot := 1; slot <= len(bySlot[o]); slot++ {
			id, ok := bySlot[o][slot]
			if !ok {
				t.Fatalf("owner %d slots are not exactly 1..%d", o, len(bySlot[o]))
			}
			out[o] = append(out[o], id)
		}
	}
	return out
}

// Feature: frontend-rework, Property 1: Slot is a value-sorted per-seat permutation.
func TestPropSlotValueSortedPermutation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := newDealtEngine(t, rapid.Int64().Draw(t, "seed"), drawLevel(t))
		s := e.state
		for viewer := SeatID(0); viewer < seatCount; viewer++ {
			ordered := slotsByOwner(t, e.View(viewer))
			for owner, ids := range ordered {
				if len(ids) != len(s.HandOrder[owner]) {
					t.Fatalf("owner %d: %d slots for %d cards", owner, len(ids), len(s.HandOrder[owner]))
				}
				pos := map[CardID]int{}
				for i, id := range s.HandOrder[owner] {
					pos[id] = i
				}
				for i := 1; i < len(ids); i++ {
					prev, cur := s.Cards[ids[i-1]].Card, s.Cards[ids[i]].Card
					switch {
					case prev.SortValue() > cur.SortValue():
						t.Fatalf("owner %d slot %d: sort value %v after %v", owner, i+1, cur.SortValue(), prev.SortValue())
					case prev.SortValue() == cur.SortValue() && pos[ids[i-1]] > pos[ids[i]]:
						t.Fatalf("owner %d slot %d: tie not stable over HandOrder", owner, i+1)
					}
				}
			}
		}
	})
}

// Feature: frontend-rework, Property 2: Slot is constant for the whole game.
func TestPropSlotConstantForWholeGame(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var baseline map[CardID]int
		runRandomGame(t, gameObs{onState: func(t *rapid.T, e *Engine) {
			for viewer := SeatID(0); viewer < seatCount; viewer++ {
				got := map[CardID]int{}
				for _, vc := range e.View(viewer).Cards {
					got[vc.ID] = vc.Slot
				}
				if baseline == nil {
					baseline = got
				}
				if !reflect.DeepEqual(baseline, got) {
					t.Fatalf("slots changed for viewer %d in phase %d", viewer, e.state.Phase)
				}
			}
		}})
	})
}
