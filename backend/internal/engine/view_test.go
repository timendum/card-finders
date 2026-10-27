package engine

import "testing"

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
