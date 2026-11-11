package engine

import (
	"reflect"
	"sort"
	"testing"

	"pgregory.net/rapid"
)

// findViewCard returns the projected ViewCard with the given id, or fails.
func findViewCard(t *testing.T, v PlayerView, id CardID) ViewCard {
	t.Helper()
	for _, vc := range v.Cards {
		if vc.ID == id {
			return vc
		}
	}
	t.Fatalf("ViewCard %d not found in view", id)
	return ViewCard{}
}

func TestViewOwnFaceDownHasValue(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Suit: 2, Kind: SpecialKindNone}) // seat 0's own card

	vc := findViewCard(t, e.View(0), 1)
	if vc.Value == nil {
		t.Fatalf("own face-down card Value = nil, want populated")
	}
	if vc.Value.Rank != 7 || vc.Value.Suit != 2 || vc.Value.Kind != SpecialKindNone {
		t.Errorf("own card Value = %+v, want rank 7 suit 2 SpecialKindNone", *vc.Value)
	}
	if vc.Owner != 0 || vc.Facing != FaceDown {
		t.Errorf("owner/facing = %d/%d, want 0/FaceDown", vc.Owner, vc.Facing)
	}
}

func TestViewOtherFaceDownHidesValue(t *testing.T) {
	e := playingState(4)
	place(e, 2, 1, Card{Rank: 9, Suit: 3, Kind: SpecialKindNone}) // seat 1's card

	vc := findViewCard(t, e.View(0), 2) // viewer is seat 0
	if vc.Value != nil {
		t.Errorf("other player's face-down Value = %+v, want nil", *vc.Value)
	}
	if vc.Owner != 1 || vc.Facing != FaceDown {
		t.Errorf("owner/facing = %d/%d, want 1/FaceDown", vc.Owner, vc.Facing)
	}
}

func TestViewFaceUpVisibleToAll(t *testing.T) {
	e := playingState(4)
	place(e, 3, 1, Card{Rank: 11, Kind: SpecialKindNone}) // seat 1's card
	e.state.Cards[3].Facing = FaceUp

	vc := findViewCard(t, e.View(0), 3) // viewer is seat 0 (not the owner)
	if vc.Value == nil {
		t.Fatalf("face-up card Value = nil for non-owner, want populated")
	}
	if vc.Value.Rank != 11 {
		t.Errorf("face-up Value.Rank = %d, want 11", vc.Value.Rank)
	}
	if vc.Facing != FaceUp {
		t.Errorf("Facing = %d, want FaceUp", vc.Facing)
	}
}

func TestViewTokenedFaceDownNoLeak(t *testing.T) {
	e := playingState(4)
	place(e, 4, 1, Card{Rank: 5, Suit: 1, Kind: SpecialKindNone}) // seat 1's card
	e.state.Cards[4].Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}

	vc := findViewCard(t, e.View(0), 4) // viewer is seat 0
	if vc.Value != nil {
		t.Errorf("tokened face-down Value = %+v, want nil — no leak", *vc.Value)
	}
	if vc.Token == nil {
		t.Fatalf("Token = nil, want present for all viewers")
	}
	if vc.TokenDisplay == nil {
		t.Fatalf("TokenDisplay = nil, want resolved value when Token != nil")
	}
	if vc.TokenDisplay.Rank != 5 || vc.TokenDisplay.Kind != SpecialKindNone {
		t.Errorf("TokenDisplay = %+v, want natural identity rank 5", *vc.TokenDisplay)
	}
}

func TestViewIncludesFullSpecialPool(t *testing.T) {
	e := playingState(4)
	e.state.SpecialPool = SpecialPool{
		Silvers: []Card{{ID: 48, Rank: 4, Kind: SpecialKindSilver}},
		Golds:   []Card{{ID: 49, Rank: 6, Kind: SpecialKindGold}},
	}

	v := e.View(2)
	if len(v.SpecialPool.Silvers) != 1 || v.SpecialPool.Silvers[0].Rank != 4 {
		t.Errorf("SpecialPool.Silvers = %+v, want one silver rank 4", v.SpecialPool.Silvers)
	}
	if len(v.SpecialPool.Golds) != 1 || v.SpecialPool.Golds[0].Rank != 6 {
		t.Errorf("SpecialPool.Golds = %+v, want one gold rank 6", v.SpecialPool.Golds)
	}
}

func TestViewYourTurnOnlyForCurrentSeat(t *testing.T) {
	e := playingState(4) // CurrentSeat is 0, PhasePlaying
	if !e.View(0).YourTurn {
		t.Errorf("View(0).YourTurn = false, want true for the current seat")
	}
	if e.View(1).YourTurn {
		t.Errorf("View(1).YourTurn = true, want false for a non-current seat")
	}
}

func TestViewNoGameReturnsSeat(t *testing.T) {
	e := &Engine{}
	v := e.View(3)
	if v.Seat != 3 {
		t.Errorf("Seat = %d, want 3", v.Seat)
	}
	if len(v.Cards) != 0 {
		t.Errorf("Cards = %+v, want empty for no game", v.Cards)
	}
}

// While a double-hit choice is pending, every seat sees the same response
// player and nobody has the turn; otherwise the fields are unset.
func TestViewResponsePlayerProjection(t *testing.T) {
	e := pendingState()
	for seat := range SeatID(4) {
		v := e.View(seat)
		if !v.HasResponsePlayer || v.ResponsePlayer != 1 {
			t.Errorf("seat %d: ResponsePlayer = %d (has=%v), want 1 (true)", seat, v.ResponsePlayer, v.HasResponsePlayer)
		}
		if v.YourTurn {
			t.Errorf("seat %d: YourTurn = true during AwaitingResponse", seat)
		}
	}

	p := safeCallState(4)
	if v := p.View(0); v.HasResponsePlayer {
		t.Errorf("HasResponsePlayer = true in Playing phase")
	}
}

// Feature: frontend-rework, Property 5: LastAction never leaks a non-actor's hidden source.
func TestPropLastActionSourcePrivacy(t *testing.T) {
	sawHidden := false
	rapid.Check(t, func(t *rapid.T) {
		runRandomGame(t, gameObs{onState: func(t *rapid.T, e *Engine) {
			auth := e.state.LastAction
			for viewer := range SeatID(seatCount) {
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
		for viewer := range SeatID(seatCount) {
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
			for viewer := range SeatID(seatCount) {
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

// Feature: card-finders, Property 21: every view includes the full Special_Pool
// (real plus decoy entries, sorted per kind, SilverInPlay/GoldInPlay = real
// counts) with no owner and no real-versus-decoy marker, in every phase.
// **Validates: Requirements 2.10, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 7.1**
func TestPropPoolViewRevealsEntriesOnly(t *testing.T) {
	sawEnded, sawDecoys := false, false
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		if lvl.ExtraSilver+lvl.ExtraGold > 0 {
			sawDecoys = true
		}
		check := func(e *Engine) {
			p := e.state.SpecialPool
			var wantS, wantG []CardValue
			for _, c := range p.Silvers {
				wantS = append(wantS, CardValue{Rank: c.Rank, Kind: SpecialKindSilver})
			}
			for _, r := range p.DecoySilvers {
				wantS = append(wantS, CardValue{Rank: r, Kind: SpecialKindSilver})
			}
			for _, c := range p.Golds {
				wantG = append(wantG, CardValue{Rank: c.Rank, Kind: SpecialKindGold})
			}
			for _, r := range p.DecoyGolds {
				wantG = append(wantG, CardValue{Rank: r, Kind: SpecialKindGold})
			}
			byRank := func(s []CardValue) {
				sort.Slice(s, func(i, j int) bool { return s[i].Rank < s[j].Rank })
			}
			byRank(wantS)
			byRank(wantG)
			for seat := range SeatID(seatCount) {
				sp := e.View(seat).SpecialPool
				if len(sp.Silvers) != lvl.SilverCount+lvl.ExtraSilver ||
					len(sp.Golds) != lvl.GoldCount+lvl.ExtraGold {
					t.Fatalf("seat %d pool sizes %d/%d, want %d/%d", seat, len(sp.Silvers), len(sp.Golds),
						lvl.SilverCount+lvl.ExtraSilver, lvl.GoldCount+lvl.ExtraGold)
				}
				if !reflect.DeepEqual(sp.Silvers, wantS) && len(wantS) > 0 {
					t.Fatalf("seat %d silvers %v, want %v", seat, sp.Silvers, wantS)
				}
				if !reflect.DeepEqual(sp.Golds, wantG) && len(wantG) > 0 {
					t.Fatalf("seat %d golds %v, want %v", seat, sp.Golds, wantG)
				}
				for i := 1; i < len(sp.Silvers); i++ {
					if sp.Silvers[i-1].Rank > sp.Silvers[i].Rank {
						t.Fatalf("silvers not ascending: %v", sp.Silvers)
					}
				}
				for i := 1; i < len(sp.Golds); i++ {
					if sp.Golds[i-1].Rank > sp.Golds[i].Rank {
						t.Fatalf("golds not ascending: %v", sp.Golds)
					}
				}
				if sp.SilverInPlay != lvl.SilverCount || sp.GoldInPlay != lvl.GoldCount {
					t.Fatalf("seat %d in-play %d/%d, want %d/%d", seat,
						sp.SilverInPlay, sp.GoldInPlay, lvl.SilverCount, lvl.GoldCount)
				}
				// The projection type carries only CardValue entries and two
				// counts: no owner, CardID, or decoy marker field exists.
				if n := reflect.TypeOf(sp).NumField(); n != 4 {
					t.Fatalf("ViewSpecialPool has %d fields, want 4", n)
				}
			}
		}
		e := runRandomGameAt(t, lvl, gameObs{onState: func(t *rapid.T, e *Engine) { check(e) }})
		if e.state.Phase == PhaseEnded {
			sawEnded = true
		}
	})
	if !sawEnded {
		t.Fatal("no run reached Ended")
	}
	if !sawDecoys {
		t.Fatal("no run used extras")
	}
}
