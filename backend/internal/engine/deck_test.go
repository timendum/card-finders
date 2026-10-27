package engine

import (
	"math/rand"
	"testing"

	"pgregory.net/rapid"
)

func TestBuildBaseDeckComposition(t *testing.T) {
	deck := BuildBaseDeck()
	if len(deck) != baseDeckSize {
		t.Fatalf("base deck size = %d, want %d", len(deck), baseDeckSize)
	}

	// One card per rank+suit, no ace, all SpecialKindNone.
	perRank := map[Rank]map[Suit]int{}
	for _, c := range deck {
		if c.Kind != SpecialKindNone {
			t.Fatalf("base card %v has kind %d, want SpecialKindNone", c, c.Kind)
		}
		if c.Rank < minRank || c.Rank > maxRank {
			t.Fatalf("base card rank %d out of range 2..13", c.Rank)
		}
		if perRank[c.Rank] == nil {
			perRank[c.Rank] = map[Suit]int{}
		}
		perRank[c.Rank][c.Suit]++
	}
	for r := minRank; r <= maxRank; r++ {
		for s := Suit(0); s < suitsPerRank; s++ {
			if perRank[r][s] != 1 {
				t.Errorf("rank %d suit %d appears %d times, want 1", r, s, perRank[r][s])
			}
		}
	}
	if _, ok := perRank[1]; ok {
		t.Error("base deck must not contain rank 1 (ace)")
	}
}

func TestBuildBaseDeckStableUniqueIDs(t *testing.T) {
	d1 := BuildBaseDeck()
	d2 := BuildBaseDeck()
	seen := map[CardID]bool{}
	for i := range d1 {
		if d1[i] != d2[i] {
			t.Fatalf("deck not stable at index %d: %v vs %v", i, d1[i], d2[i])
		}
		if seen[d1[i].ID] {
			t.Fatalf("duplicate CardID %d", d1[i].ID)
		}
		seen[d1[i].ID] = true
	}
	if len(seen) != baseDeckSize {
		t.Fatalf("got %d distinct IDs, want %d", len(seen), baseDeckSize)
	}
}

func TestBuildSpecialPoolCountsAndKinds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	l := Level{Number: 2, Name: "L", SilverCount: 4, GoldCount: 3, StartLives: 5}
	pool, err := BuildSpecialPool(rng, l)
	if err != nil {
		t.Fatalf("BuildSpecialPool: %v", err)
	}
	if len(pool.Silvers) != l.SilverCount {
		t.Errorf("silvers = %d, want %d", len(pool.Silvers), l.SilverCount)
	}
	if len(pool.Golds) != l.GoldCount {
		t.Errorf("golds = %d, want %d", len(pool.Golds), l.GoldCount)
	}
	for _, c := range pool.Silvers {
		if c.Kind != SpecialKindSilver {
			t.Errorf("silver entry kind = %d", c.Kind)
		}
	}
	for _, c := range pool.Golds {
		if c.Kind != SpecialKindGold {
			t.Errorf("gold entry kind = %d", c.Kind)
		}
	}
}

func TestBuildSpecialPoolSeedReproducible(t *testing.T) {
	l := Level{Number: 3, Name: "L", SilverCount: 6, GoldCount: 5, StartLives: 5}
	a, err := BuildSpecialPool(rand.New(rand.NewSource(42)), l)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildSpecialPool(rand.New(rand.NewSource(42)), l)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Silvers) != len(b.Silvers) || len(a.Golds) != len(b.Golds) {
		t.Fatal("length mismatch across identical seeds")
	}
	for i := range a.Silvers {
		if a.Silvers[i] != b.Silvers[i] {
			t.Fatalf("silver %d differs: %v vs %v", i, a.Silvers[i], b.Silvers[i])
		}
	}
	for i := range a.Golds {
		if a.Golds[i] != b.Golds[i] {
			t.Fatalf("gold %d differs: %v vs %v", i, a.Golds[i], b.Golds[i])
		}
	}
}

func TestBuildSpecialPoolRejectsInvalidLevel(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// GoldCount 12 exceeds the 11 eligible ranks.
	l := Level{Number: 4, Name: "L", SilverCount: 0, GoldCount: 12, StartLives: 5}
	if _, err := BuildSpecialPool(rng, l); err == nil {
		t.Fatal("expected error for gold count exceeding eligible ranks")
	}
}

// Feature: card-finders, Property 5: Special_Pool is well-formed and public before dealing.
func TestPropSpecialPoolWellFormed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		silver := rapid.IntRange(0, eligibleSpecialRanks).Filter(func(n int) bool { return n%2 == 0 }).Draw(t, "silver")
		gold := rapid.IntRange(0, eligibleSpecialRanks).Draw(t, "gold")
		seed := rapid.Int64().Draw(t, "seed")

		l := Level{Number: 1, Name: "L", SilverCount: silver, GoldCount: gold, StartLives: 10}
		if err := ValidateLevel(l); err != nil {
			t.Fatalf("generated level should be valid: %v", err)
		}

		pool, err := BuildSpecialPool(rand.New(rand.NewSource(seed)), l)
		if err != nil {
			t.Fatalf("BuildSpecialPool: %v", err)
		}

		if len(pool.Silvers) != silver {
			t.Fatalf("silver count = %d, want %d", len(pool.Silvers), silver)
		}
		if len(pool.Golds) != gold {
			t.Fatalf("gold count = %d, want %d", len(pool.Golds), gold)
		}

		silverRanks := checkKind(t, pool.Silvers, SpecialKindSilver, 0.3)
		goldRanks := checkKind(t, pool.Golds, SpecialKindGold, 0.6)

		// A shared rank must yield two distinct cards (sort values already differ by kind).
		for r, silverID := range silverRanks {
			if goldID, shared := goldRanks[r]; shared {
				if silverID == goldID {
					t.Fatalf("shared rank %d produced identical card IDs", r)
				}
			}
		}
	})
}

// checkKind asserts kind, rank range, distinct ranks, and sort-value offset,
// returning each rank mapped to its CardID.
func checkKind(t *rapid.T, cards []Card, kind SpecialKind, offset float64) map[Rank]CardID {
	ranks := map[Rank]CardID{}
	for _, c := range cards {
		if c.Kind != kind {
			t.Fatalf("entry kind = %d, want %d", c.Kind, kind)
		}
		if c.Rank < minRank || c.Rank > maxSpecialRank {
			t.Fatalf("special rank %d out of range 2..queen", c.Rank)
		}
		if _, dup := ranks[c.Rank]; dup {
			t.Fatalf("duplicate rank %d within kind %d", c.Rank, kind)
		}
		if got, want := c.SortValue(), float64(c.Rank)+offset; got != want {
			t.Fatalf("sort value = %v, want %v", got, want)
		}
		ranks[c.Rank] = c.ID
	}
	return ranks
}
