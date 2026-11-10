package engine

import (
	"errors"
	"testing"
)

// openingGame fills four seats and returns the engine in PhaseOpening with the
// seat chosen to open first.
func openingGame(t *testing.T) (*Engine, SeatID) {
	t.Helper()
	level := Level{Number: 1, Name: "NoSpecials", SilverCount: 0, GoldCount: 0, StartLives: 4}
	e := fillFourSeats(t, level)
	if e.state.Phase != PhaseOpening {
		t.Fatalf("setup: phase = %d, want PhaseOpening", e.state.Phase)
	}
	return e, e.state.CurrentSeat
}

func firstCardOf(t *testing.T, e *Engine, seat SeatID) CardID {
	t.Helper()
	ids := e.state.HandOrder[seat]
	if len(ids) == 0 {
		t.Fatalf("seat %d has no cards", seat)
	}
	return ids[0]
}

func TestOpeningRevealSuccess(t *testing.T) {
	e, first := openingGame(t)
	card := firstCardOf(t, e, first)

	res, err := e.Apply(first, Action{Kind: ActionKindOpeningReveal, OpeningCard: card})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	pc := e.state.Cards[card]
	if pc.Facing != FaceDown {
		t.Errorf("facing = %d, want FaceDown", pc.Facing)
	}
	if pc.Token == nil || pc.Token.Displayed != TokenKindNaturalIdentity {
		t.Errorf("token = %+v, want natural-identity token", pc.Token)
	}
	if !e.state.OpeningDone[first] {
		t.Errorf("OpeningDone[%d] = false, want true", first)
	}
	if e.state.CurrentSeat != nextSeat(first) {
		t.Errorf("CurrentSeat = %d, want %d (next clockwise)", e.state.CurrentSeat, nextSeat(first))
	}
	if e.state.Phase != PhaseOpening {
		t.Errorf("phase = %d, want PhaseOpening until all four reveal", e.state.Phase)
	}
}

func TestOpeningAllFourTransitionToPlaying(t *testing.T) {
	e, first := openingGame(t)

	seat := first
	for i := 0; i < 4; i++ {
		card := firstCardOf(t, e, seat)
		if _, err := e.Apply(seat, Action{Kind: ActionKindOpeningReveal, OpeningCard: card}); err != nil {
			t.Fatalf("Apply reveal #%d (seat %d): %v", i+1, seat, err)
		}
		seat = nextSeat(seat)
	}

	for s := 0; s < 4; s++ {
		if !e.state.OpeningDone[s] {
			t.Errorf("OpeningDone[%d] = false, want true after all reveals", s)
		}
	}
	if e.state.Phase != PhasePlaying {
		t.Errorf("phase = %d, want PhasePlaying after four reveals", e.state.Phase)
	}
	if e.state.CurrentSeat != first {
		t.Errorf("CurrentSeat = %d, want %d (back to opening first player)", e.state.CurrentSeat, first)
	}
}

func TestOpeningOutOfTurnRejected(t *testing.T) {
	e, first := openingGame(t)
	other := nextSeat(first) // not yet this seat's turn
	card := firstCardOf(t, e, other)

	res, err := e.Apply(other, Action{Kind: ActionKindOpeningReveal, OpeningCard: card})
	if !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	pc := e.state.Cards[card]
	if pc.Facing != FaceDown || pc.Token != nil {
		t.Errorf("card mutated by rejected reveal: facing=%d token=%+v, want unchanged", pc.Facing, pc.Token)
	}
	if e.state.OpeningDone[other] {
		t.Errorf("OpeningDone[%d] = true, want unchanged", other)
	}
	if e.state.CurrentSeat != first {
		t.Errorf("CurrentSeat = %d, want %d unchanged", e.state.CurrentSeat, first)
	}
}

func TestOpeningNonOwnedCardRejected(t *testing.T) {
	e, first := openingGame(t)
	other := nextSeat(first)
	foreign := firstCardOf(t, e, other) // owned by a different seat

	res, err := e.Apply(first, Action{Kind: ActionKindOpeningReveal, OpeningCard: foreign})
	if !errors.Is(err, ErrInvalidOpeningSelection) {
		t.Errorf("err = %v, want ErrInvalidOpeningSelection", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	pc := e.state.Cards[foreign]
	if pc.Facing != FaceDown || pc.Token != nil {
		t.Errorf("foreign card mutated: facing=%d token=%+v, want unchanged", pc.Facing, pc.Token)
	}
	if e.state.OpeningDone[first] {
		t.Errorf("OpeningDone[%d] = true, want unchanged", first)
	}
	if e.state.CurrentSeat != first {
		t.Errorf("CurrentSeat = %d, want %d unchanged", e.state.CurrentSeat, first)
	}
}

func TestOpeningAlreadyFaceUpRejected(t *testing.T) {
	e, first := openingGame(t)
	card := firstCardOf(t, e, first)

	// Force the card face-up on the opener's turn to hit the already-revealed
	// guard without playing four reveals to wrap back around.
	e.state.Cards[card].Facing = FaceUp

	res, err := e.Apply(first, Action{Kind: ActionKindOpeningReveal, OpeningCard: card})
	if !errors.Is(err, ErrInvalidOpeningSelection) {
		t.Errorf("err = %v, want ErrInvalidOpeningSelection", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}
	if e.state.OpeningDone[first] {
		t.Errorf("OpeningDone[%d] = true, want unchanged", first)
	}
	if e.state.CurrentSeat != first {
		t.Errorf("CurrentSeat = %d, want %d unchanged", e.state.CurrentSeat, first)
	}
}

func TestOpeningSpecialCardRejected(t *testing.T) {
	for _, kind := range []SpecialKind{SpecialKindSilver, SpecialKindGold} {
		e, first := openingGame(t)
		card := firstCardOf(t, e, first)
		// Turn the opener's card into a special without a full specials deal.
		e.state.Cards[card].Card.Kind = kind

		res, err := e.Apply(first, Action{Kind: ActionKindOpeningReveal, OpeningCard: card})
		if !errors.Is(err, ErrSpecialOpening) {
			t.Errorf("kind %d: err = %v, want ErrSpecialOpening", kind, err)
		}
		if res != ResultNone {
			t.Errorf("kind %d: result = %d, want ResultNone", kind, res)
		}
		if e.state.Cards[card].Token != nil {
			t.Errorf("kind %d: token placed on rejected special", kind)
		}
		if e.state.OpeningDone[first] || e.state.CurrentSeat != first {
			t.Errorf("kind %d: state changed after rejected reveal", kind)
		}
	}
}
