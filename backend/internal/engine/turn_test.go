package engine

import (
	"errors"
	"testing"
)

// advanceTurn skips an empty-handed seat and lands on the next clockwise seat
// that holds a face-down card.
func TestAdvanceTurnSkipsEmptyHandedSeat(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	// seat 1 gets no card: must be skipped.
	place(e, 2, 2, Card{Rank: 9, Kind: SpecialKindNone})

	if ok := e.advanceTurn(); !ok {
		t.Fatalf("advanceTurn returned false, want true (an eligible seat exists)")
	}
	if e.state.CurrentSeat != 2 {
		t.Errorf("CurrentSeat = %d, want 2 (seat 1 skipped)", e.state.CurrentSeat)
	}
}

// When the actor is the only seat with a face-down card, advancing wraps back
// to the actor.
func TestAdvanceTurnWrapsToSelfWhenOnlyActorHasCards(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})

	if ok := e.advanceTurn(); !ok {
		t.Fatalf("advanceTurn returned false, want true")
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 (wrapped back to the only eligible seat)", e.state.CurrentSeat)
	}
}

// When no seat has a face-down card, advanceTurn reports no eligible player and
// leaves CurrentSeat unchanged.
func TestAdvanceTurnNoEligiblePlayer(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	e.state.Cards[1].Facing = FaceUp

	if ok := e.advanceTurn(); ok {
		t.Errorf("advanceTurn returned true, want false (no eligible player)")
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 unchanged when no eligible seat", e.state.CurrentSeat)
	}
}

// A miss dropping Life_Pool to zero ends the game in a loss, and the turn does
// not advance once terminal.
func TestGuessMissToZeroLifeEndsInLoss(t *testing.T) {
	e := playingState(1)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // source
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone}) // target (miss)
	place(e, 3, 2, Card{Rank: 5, Kind: SpecialKindNone}) // spare, so not also a win

	res, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCards: []CardID{1}, TargetCards: []CardID{2}})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultLoss {
		t.Errorf("result = %d, want ResultLoss", res)
	}
	if e.state.Phase != PhaseEnded {
		t.Errorf("Phase = %d, want PhaseEnded", e.state.Phase)
	}
	if e.state.GameResult != ResultLoss {
		t.Errorf("GameResult = %d, want ResultLoss", e.state.GameResult)
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 (turn does not advance once ended)", e.state.CurrentSeat)
	}
}

// A hit flipping the last two face-down cards ends the game in a win.
func TestGuessHitRevealingLastCardsEndsInWin(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone}) // source
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone}) // target (match), last two cards

	res, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCards: []CardID{1}, TargetCards: []CardID{2}})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultWin {
		t.Errorf("result = %d, want ResultWin", res)
	}
	if e.state.Phase != PhaseEnded || e.state.GameResult != ResultWin {
		t.Errorf("phase/result = %d/%d, want Ended/ResultWin", e.state.Phase, e.state.GameResult)
	}
}

// A self-reveal flipping the last face-down cards ends the game in a win, so the
// terminal check runs for self-reveal too.
func TestSelfRevealLastCardsEndsInWin(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindSilver})
	place(e, 2, 0, Card{Rank: 9, Kind: SpecialKindSilver})

	res, err := e.Apply(0, selfReveal(SpecialKindSilver, 0))
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultWin {
		t.Errorf("result = %d, want ResultWin", res)
	}
	if e.state.Phase != PhaseEnded || e.state.GameResult != ResultWin {
		t.Errorf("phase/result = %d/%d, want Ended/ResultWin", e.state.Phase, e.state.GameResult)
	}
}

// Once the game is Ended, any action is rejected with ErrGameEnded, state is
// unchanged, and Apply returns the recorded GameResult.
func TestActionAfterEndedRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	e.state.Phase = PhaseEnded
	e.state.GameResult = ResultWin

	res, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCards: []CardID{1}, TargetCards: []CardID{2}})
	if !errors.Is(err, ErrGameEnded) {
		t.Errorf("err = %v, want ErrGameEnded", err)
	}
	if res != ResultWin {
		t.Errorf("result = %d, want recorded ResultWin", res)
	}
	if e.state.Cards[1].Facing != FaceDown || e.state.Cards[2].Facing != FaceDown {
		t.Errorf("cards mutated after game ended")
	}
}
