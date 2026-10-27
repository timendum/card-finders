package engine

// Unit tests for the fourth-join deal: conservation, at-most-one balance, and
// the undersized-deck abort.

import (
	"errors"
	"math/rand"
	"testing"
)

// fillFourSeats creates a game at level and joins four players, returning the
// engine for inspection.
func fillFourSeats(t *testing.T, level Level) *Engine {
	t.Helper()
	e := NewEngine([]Level{level}, rand.New(rand.NewSource(42)))
	code, err := e.CreateGame(level.Number)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, _, err := e.Join(code); err != nil {
			t.Fatalf("Join #%d: %v", i+1, err)
		}
	}
	return e
}

// TestDealConservationAndBalance checks conservation and balance across several
// levels after the fourth join.
func TestDealConservationAndBalance(t *testing.T) {
	levels := []Level{
		{Number: 1, Name: "NoSpecials", SilverCount: 0, GoldCount: 0, StartLives: 4},
		{Number: 2, Name: "Silvers", SilverCount: 4, GoldCount: 0, StartLives: 5},
		{Number: 3, Name: "Mixed", SilverCount: 2, GoldCount: 3, StartLives: 6},
		{Number: 4, Name: "MaxSpecials", SilverCount: 10, GoldCount: 11, StartLives: 9},
	}
	for _, level := range levels {
		t.Run(level.Name, func(t *testing.T) {
			e := fillFourSeats(t, level)

			if e.state.Phase != PhaseOpening {
				t.Errorf("phase = %d, want PhaseOpening after the deal", e.state.Phase)
			}

			want := baseDeckSize + level.SilverCount + level.GoldCount

			// Conservation: every dealt id appears once across Cards and HandOrder.
			if got := len(e.state.Cards); got != want {
				t.Errorf("Cards count = %d, want %d", got, want)
			}
			seen := map[CardID]bool{}
			total := 0
			for seat := 0; seat < 4; seat++ {
				for _, id := range e.state.HandOrder[seat] {
					if seen[id] {
						t.Errorf("card %d dealt more than once", id)
					}
					seen[id] = true
					total++
					pc, ok := e.state.Cards[id]
					if !ok {
						t.Errorf("HandOrder card %d missing from Cards", id)
						continue
					}
					if int(pc.Owner) != seat {
						t.Errorf("card %d owner = %d, want seat %d", id, pc.Owner, seat)
					}
					if pc.Facing != FaceDown {
						t.Errorf("card %d facing = %d, want FaceDown", id, pc.Facing)
					}
					if pc.Token != nil {
						t.Errorf("card %d has a token before opening", id)
					}
				}
			}
			if total != want {
				t.Errorf("dealt %d cards across hands, want %d", total, want)
			}

			// Balance: max and min hand sizes differ by at most one.
			minSize, maxSize := len(e.state.HandOrder[0]), len(e.state.HandOrder[0])
			for seat := 1; seat < 4; seat++ {
				n := len(e.state.HandOrder[seat])
				if n < minSize {
					minSize = n
				}
				if n > maxSize {
					maxSize = n
				}
			}
			if maxSize-minSize > 1 {
				t.Errorf("hand sizes differ by %d (max %d, min %d), want <= 1", maxSize-minSize, maxSize, minSize)
			}
		})
	}
}

// TestDealCombinedDeckSmallDecks exercises round-robin distribution at sizes
// that are not multiples of four.
func TestDealCombinedDeckSmallDecks(t *testing.T) {
	cases := []struct {
		size  int
		sizes [4]int // expected per-seat hand sizes
	}{
		{size: 4, sizes: [4]int{1, 1, 1, 1}},
		{size: 5, sizes: [4]int{2, 1, 1, 1}},
		{size: 6, sizes: [4]int{2, 2, 1, 1}},
		{size: 7, sizes: [4]int{2, 2, 2, 1}},
	}
	for _, tc := range cases {
		e := NewEngine(nil, rand.New(rand.NewSource(1)))
		e.state = &GameState{}

		deck := make([]Card, tc.size)
		for i := range deck {
			deck[i] = Card{ID: CardID(i), Rank: minRank, Kind: SpecialKindNone}
		}
		e.dealCombinedDeck(deck)

		got := 0
		for seat := 0; seat < 4; seat++ {
			if n := len(e.state.HandOrder[seat]); n != tc.sizes[seat] {
				t.Errorf("size %d: seat %d hand = %d, want %d", tc.size, seat, n, tc.sizes[seat])
			}
			got += len(e.state.HandOrder[seat])
		}
		if got != tc.size {
			t.Errorf("size %d: dealt %d cards, want %d", tc.size, got, tc.size)
		}
		if len(e.state.Cards) != tc.size {
			t.Errorf("size %d: Cards count = %d, want %d", tc.size, len(e.state.Cards), tc.size)
		}
	}
}

// TestStartGameInsufficientCardsAborts drives the abort branch directly: a
// sub-seatCount deck returns ErrInsufficientCards and deals no hands. The real
// Base_Deck is always 48 cards, so this guard is unreachable through a normal
// level.
func TestStartGameInsufficientCardsAborts(t *testing.T) {
	e := NewEngine(nil, rand.New(rand.NewSource(1)))
	e.state = &GameState{Phase: PhaseWaiting}

	if err := e.dealIfSufficient([]Card{{ID: 0}, {ID: 1}, {ID: 2}}); !errors.Is(err, ErrInsufficientCards) {
		t.Fatalf("dealIfSufficient(3 cards): err = %v, want ErrInsufficientCards", err)
	}
	for seat := 0; seat < 4; seat++ {
		if len(e.state.HandOrder[seat]) != 0 {
			t.Errorf("seat %d has a dealt hand after abort, want none", seat)
		}
	}
	if len(e.state.Cards) != 0 {
		t.Errorf("Cards non-empty (%d) after abort, want 0", len(e.state.Cards))
	}

	// Exactly four cards is sufficient.
	if err := e.dealIfSufficient([]Card{{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3}}); err != nil {
		t.Fatalf("dealIfSufficient(4 cards): unexpected error %v", err)
	}
	if len(e.state.Cards) != 4 {
		t.Errorf("Cards count = %d, want 4 after a sufficient deal", len(e.state.Cards))
	}
}
