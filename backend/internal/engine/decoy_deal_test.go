package engine

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"pgregory.net/rapid"
)

// Feature: card-finders, Property 5: real draws are independent of extras
// (real silver/gold ranks equal those of the same level with no extras).
func TestPropRealDrawsIndependentOfExtras(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		seed := rapid.Int64().Draw(t, "seed")
		base := lvl
		base.ExtraSilver, base.ExtraGold = 0, 0

		withExtras, err := BuildSpecialPool(rand.New(rand.NewSource(seed)), lvl)
		if err != nil {
			t.Fatalf("BuildSpecialPool with extras: %v", err)
		}
		noExtras, err := BuildSpecialPool(rand.New(rand.NewSource(seed)), base)
		if err != nil {
			t.Fatalf("BuildSpecialPool without extras: %v", err)
		}
		if !reflect.DeepEqual(cardRanks(withExtras.Silvers), cardRanks(noExtras.Silvers)) {
			t.Fatalf("real silvers differ: %v vs %v", cardRanks(withExtras.Silvers), cardRanks(noExtras.Silvers))
		}
		if !reflect.DeepEqual(cardRanks(withExtras.Golds), cardRanks(noExtras.Golds)) {
			t.Fatalf("real golds differ: %v vs %v", cardRanks(withExtras.Golds), cardRanks(noExtras.Golds))
		}
		if len(noExtras.DecoySilvers)+len(noExtras.DecoyGolds) != 0 {
			t.Fatalf("zero extras produced decoys")
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
		for seat := 0; seat < 4; seat++ {
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
			sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
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
