package engine

// Unit tests for the fourth-join deal: conservation, at-most-one balance, and
// the undersized-deck abort.

import (
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"pgregory.net/rapid"
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
	for i := range 4 {
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
			for seat := range 4 {
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
		for seat := range 4 {
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
		for viewer := range SeatID(seatCount) {
			for _, vc := range e.View(viewer).Cards {
				if vc.Owner != viewer && vc.Facing == FaceDown && vc.Value != nil {
					t.Fatalf("seat %d sees opponent card %d value", viewer, vc.ID)
				}
			}
		}
	})
}

// Feature: card-finders, Property 5: the pool (real and decoy) is
// deterministic for a given seed and level.
func TestPropPoolSameSeedIdentical(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		seed := rapid.Int64().Draw(t, "seed")
		a, err := BuildSpecialPool(rand.New(rand.NewSource(seed)), lvl)
		if err != nil {
			t.Fatalf("BuildSpecialPool: %v", err)
		}
		b, err := BuildSpecialPool(rand.New(rand.NewSource(seed)), lvl)
		if err != nil {
			t.Fatalf("BuildSpecialPool: %v", err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("pools differ for same seed: %v vs %v", a, b)
		}
	})
}

// Feature: card-finders, Property 5: with zero extras, pool, deal, and first
// player are identical for engines sharing a seed.
func TestPropZeroExtrasSameSeedIdentical(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevel(t) // extras are zero
		seed := rapid.Int64().Draw(t, "seed")
		a := newDealtEngine(t, seed, lvl)
		b := newDealtEngine(t, seed, lvl)
		if !reflect.DeepEqual(a.state.SpecialPool, b.state.SpecialPool) {
			t.Fatalf("special pools differ")
		}
		if !reflect.DeepEqual(a.state.HandOrder, b.state.HandOrder) {
			t.Fatalf("deals differ")
		}
		if a.state.CurrentSeat != b.state.CurrentSeat {
			t.Fatalf("first player differs: %v vs %v", a.state.CurrentSeat, b.state.CurrentSeat)
		}
	})
}

// Feature: card-finders, Property 5: only real specials are dealt.
func TestPropOnlyRealSpecialsDealt(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		seed := rapid.Int64().Draw(t, "seed")
		e := newDealtEngine(t, seed, lvl)
		p := e.state.SpecialPool

		var silvers, golds []Rank
		for seat := range 4 {
			for _, id := range e.state.HandOrder[seat] {
				c := e.state.Cards[id].Card
				switch c.Kind {
				case SpecialKindSilver:
					silvers = append(silvers, c.Rank)
				case SpecialKindGold:
					golds = append(golds, c.Rank)
				}
			}
		}
		if len(silvers) != lvl.SilverCount || len(golds) != lvl.GoldCount {
			t.Fatalf("dealt %d silvers/%d golds, want %d/%d", len(silvers), len(golds), lvl.SilverCount, lvl.GoldCount)
		}
		sortRanks := func(r []Rank) []Rank {
			out := append([]Rank(nil), r...)
			slices.Sort(out)
			return out
		}
		if !reflect.DeepEqual(sortRanks(silvers), sortRanks(cardRanks(p.Silvers))) {
			t.Fatalf("dealt silvers %v != real silvers %v", silvers, cardRanks(p.Silvers))
		}
		if !reflect.DeepEqual(sortRanks(golds), sortRanks(cardRanks(p.Golds))) {
			t.Fatalf("dealt golds %v != real golds %v", golds, cardRanks(p.Golds))
		}
	})
}
