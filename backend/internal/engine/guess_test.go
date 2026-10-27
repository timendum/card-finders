package engine

import (
	"errors"
	"testing"
)

// playingState builds a PhasePlaying state with seat 0 current and LifePool set
// to lives. Tests place cards by id and value so hit/miss and ownership are
// deterministic rather than dependent on a random deal.
func playingState(lives int) *Engine {
	e := &Engine{}
	s := &GameState{
		Phase:       PhasePlaying,
		CurrentSeat: 0,
		LifePool:    lives,
		Cards:       make(map[CardID]*PlacedCard),
	}
	e.state = s
	return e
}

// place inserts a face-down card owned by owner into the engine state.
func place(e *Engine, id CardID, owner SeatID, c Card) {
	c.ID = id
	e.state.Cards[id] = &PlacedCard{Card: c, Owner: owner, Facing: FaceDown}
	e.state.HandOrder[owner] = append(e.state.HandOrder[owner], id)
}

// TestGuessHitRevealsBothNoLifeLoss: a matching guess flips both cards face-up,
// costs no life, and passes the turn clockwise.
func TestGuessHitRevealsBothNoLifeLoss(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	// Spare face-down card so not all cards are face-up after the hit; otherwise
	// the game would win and the turn would not pass.
	place(e, 3, 1, Card{Rank: 9, Kind: SpecialKindNone})

	res, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	src, tgt := e.state.Cards[1], e.state.Cards[2]
	if src.Facing != FaceUp || tgt.Facing != FaceUp {
		t.Errorf("facings = src:%d tgt:%d, want both FaceUp (Pair_Reveal)", src.Facing, tgt.Facing)
	}
	// Face-up cards show their value directly, so neither carries a token.
	if src.Token != nil || tgt.Token != nil {
		t.Errorf("tokens = src:%+v tgt:%+v, want both nil on reveal", src.Token, tgt.Token)
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged on a hit", e.state.LifePool)
	}
	if e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("CurrentSeat = %d, want %d (turn passed)", e.state.CurrentSeat, nextSeat(0))
	}
}

// TestGuessMissTokensTargetLeavesSource: a non-matching guess costs one life,
// tokens the target face-down, leaves the source untouched, and passes the turn.
func TestGuessMissTokensTargetLeavesSource(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})

	res, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if res != ResultNone {
		t.Errorf("result = %d, want ResultNone", res)
	}

	src, tgt := e.state.Cards[1], e.state.Cards[2]
	if e.state.LifePool != 3 {
		t.Errorf("LifePool = %d, want 3 (-1 on a miss)", e.state.LifePool)
	}
	if tgt.Facing != FaceDown || tgt.Token == nil || tgt.Token.Displayed != TokenKindNaturalIdentity {
		t.Errorf("target = facing:%d token:%+v, want face-down with natural-identity token", tgt.Facing, tgt.Token)
	}
	if src.Facing != FaceDown || src.Token != nil {
		t.Errorf("source = facing:%d token:%+v, want face-down and untokened", src.Facing, src.Token)
	}
	if e.state.CurrentSeat != nextSeat(0) {
		t.Errorf("CurrentSeat = %d, want %d (turn passed)", e.state.CurrentSeat, nextSeat(0))
	}
}

// TestGuessMissLifeClampedAtZero: a miss never drives Life_Pool below zero.
func TestGuessMissLifeClampedAtZero(t *testing.T) {
	e := playingState(0)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindNone})

	if _, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.LifePool != 0 {
		t.Errorf("LifePool = %d, want 0 clamped", e.state.LifePool)
	}
}

// TestGuessSilverFeatHit: a silver source against a silver target is a hit even
// when their ranks differ.
func TestGuessSilverFeatHit(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindSilver})
	place(e, 2, 1, Card{Rank: 9, Kind: SpecialKindSilver}) // rank differs; still a match

	if _, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[2].Facing != FaceUp {
		t.Errorf("silver feat hit did not flip both face-up")
	}
	if e.state.LifePool != 4 {
		t.Errorf("LifePool = %d, want 4 unchanged on silver hit", e.state.LifePool)
	}
}

// TestGuessSilverFeatMissSuitedTarget: a silver source against a suited target
// is a miss, even at the same rank.
func TestGuessSilverFeatMissSuitedTarget(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindSilver})
	place(e, 2, 1, Card{Rank: 5, Kind: SpecialKindNone}) // same rank, not silver -> miss

	if _, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2}); err != nil {
		t.Fatalf("Apply: unexpected error %v", err)
	}
	if e.state.LifePool != 3 {
		t.Errorf("LifePool = %d, want 3 (silver-vs-suited is a miss)", e.state.LifePool)
	}
	if e.state.Cards[2].Token == nil || e.state.Cards[2].Facing != FaceDown {
		t.Errorf("silver feat miss should token the target face-down")
	}
}

// TestGuessSilverFeatSelfTargetRejected: a silver source cannot pair against its
// owner's own card; rejected with ErrTargetOwn, nothing mutated.
func TestGuessSilverFeatSelfTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindSilver})
	place(e, 2, 0, Card{Rank: 9, Kind: SpecialKindSilver}) // target owned by actor

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrTargetOwn) {
		t.Errorf("err = %v, want ErrTargetOwn (silver feat self target)", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// assertUnchanged checks a rejected guess mutated none of source, target, life
// pool, or current seat.
func assertUnchanged(t *testing.T, e *Engine, src, tgt CardID, wantLife int) {
	t.Helper()
	if s := e.state.Cards[src]; s != nil && (s.Facing != FaceDown || s.Token != nil) {
		t.Errorf("source mutated by rejected guess: facing=%d token=%+v", s.Facing, s.Token)
	}
	if g := e.state.Cards[tgt]; g != nil && (g.Facing != FaceDown || g.Token != nil) {
		t.Errorf("target mutated by rejected guess: facing=%d token=%+v", g.Facing, g.Token)
	}
	if e.state.LifePool != wantLife {
		t.Errorf("LifePool = %d, want %d unchanged on rejection", e.state.LifePool, wantLife)
	}
	if e.state.CurrentSeat != 0 {
		t.Errorf("CurrentSeat = %d, want 0 unchanged on rejection", e.state.CurrentSeat)
	}
}

// TestGuessGoldSourceRejected: a gold source is rejected with ErrGoldSource.
func TestGuessGoldSourceRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindGold})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrGoldSource) {
		t.Errorf("err = %v, want ErrGoldSource", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessNonOwnedSourceRejected: a source the seat does not own is rejected
// with ErrInvalidSource.
func TestGuessNonOwnedSourceRejected(t *testing.T) {
	e := playingState(4)
	// Seat 0 holds a face-down card so it is eligible to act, isolating source
	// validation from the empty-handed guard.
	place(e, 10, 0, Card{Rank: 3, Kind: SpecialKindNone})
	place(e, 1, 1, Card{Rank: 7, Kind: SpecialKindNone}) // owned by seat 1
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrInvalidSource) {
		t.Errorf("err = %v, want ErrInvalidSource", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessUnknownTargetRejected: an absent target is rejected with
// ErrInvalidTarget.
func TestGuessUnknownTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 99})
	if !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("err = %v, want ErrInvalidTarget", err)
	}
	assertUnchanged(t, e, 1, 99, 4)
}

// TestGuessRevealedTargetRejected: a face-up target is rejected with
// ErrTargetRevealed.
func TestGuessRevealedTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	e.state.Cards[2].Facing = FaceUp

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrTargetRevealed) {
		t.Errorf("err = %v, want ErrTargetRevealed", err)
	}
	if e.state.LifePool != 4 || e.state.CurrentSeat != 0 {
		t.Errorf("state mutated by rejected guess: life=%d seat=%d", e.state.LifePool, e.state.CurrentSeat)
	}
}

// TestGuessOwnTargetRejected: targeting the seat's own card is rejected with
// ErrTargetOwn.
func TestGuessOwnTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone}) // target owned by actor

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrTargetOwn) {
		t.Errorf("err = %v, want ErrTargetOwn", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessWrongPhaseRejected: a guess outside PhasePlaying is rejected with
// ErrWrongPhase.
func TestGuessWrongPhaseRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindNone})
	e.state.Phase = PhaseOpening

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrWrongPhase) {
		t.Errorf("err = %v, want ErrWrongPhase", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessOutOfTurnRejected: a guess by a seat other than the current one is
// rejected with ErrNotYourTurn.
func TestGuessOutOfTurnRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 1, Card{Rank: 7, Kind: SpecialKindNone}) // seat 1's source
	place(e, 2, 0, Card{Rank: 7, Kind: SpecialKindNone})

	_, err := e.Apply(1, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessGoldTargetRejected: a normal guess targeting another player's gold
// card is rejected with ErrGoldNotTargetable.
func TestGuessGoldTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 7, Kind: SpecialKindNone})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindGold})

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrGoldNotTargetable) {
		t.Errorf("err = %v, want ErrGoldNotTargetable", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}

// TestGuessSilverFeatGoldTargetRejected: the silver feat targeting another
// player's gold card is likewise rejected with ErrGoldNotTargetable.
func TestGuessSilverFeatGoldTargetRejected(t *testing.T) {
	e := playingState(4)
	place(e, 1, 0, Card{Rank: 5, Kind: SpecialKindSilver})
	place(e, 2, 1, Card{Rank: 7, Kind: SpecialKindGold})

	_, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 1, TargetCard: 2})
	if !errors.Is(err, ErrGoldNotTargetable) {
		t.Errorf("err = %v, want ErrGoldNotTargetable (silver feat)", err)
	}
	assertUnchanged(t, e, 1, 2, 4)
}
