package engine

import (
	"testing"

	"pgregory.net/rapid"
)

// drawLevelWithExtras draws a valid level with decoy extras in
// 0..11-count, biased toward zero extras so the legacy path stays covered.
func drawLevelWithExtras(t *rapid.T) Level {
	l := drawLevel(t)
	l.ExtraSilver = drawExtra(t, "extraSilver", eligibleSpecialRanks-l.SilverCount)
	l.ExtraGold = drawExtra(t, "extraGold", eligibleSpecialRanks-l.GoldCount)
	return l
}

func drawExtra(t *rapid.T, label string, max int) int {
	if rapid.IntRange(0, 2).Draw(t, label+"Zero") == 0 {
		return 0
	}
	return rapid.IntRange(0, max).Draw(t, label)
}

func rankSet(t *rapid.T, what string, ranks []Rank) map[Rank]bool {
	set := map[Rank]bool{}
	for _, r := range ranks {
		if r < minRank || r > maxSpecialRank {
			t.Fatalf("%s rank %d out of range 2..queen", what, r)
		}
		if set[r] {
			t.Fatalf("%s duplicate rank %d", what, r)
		}
		set[r] = true
	}
	return set
}

func cardRanks(cards []Card) []Rank {
	out := make([]Rank, 0, len(cards))
	for _, c := range cards {
		out = append(out, c.Rank)
	}
	return out
}

// Feature: card-finders, Property 5: Special_Pool is well-formed and public before dealing
// (revised for decoys: real plus decoy entries per kind).
func TestPropSpecialPoolCompositionWithDecoys(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		seed := rapid.Int64().Draw(t, "seed")
		e := newDealtEngine(t, seed, lvl)
		p := e.state.SpecialPool

		if got, want := len(p.Silvers)+len(p.DecoySilvers), lvl.SilverCount+lvl.ExtraSilver; got != want {
			t.Fatalf("silver entries = %d, want %d", got, want)
		}
		if got, want := len(p.Golds)+len(p.DecoyGolds), lvl.GoldCount+lvl.ExtraGold; got != want {
			t.Fatalf("gold entries = %d, want %d", got, want)
		}
		// Ranks in 2..Q and distinct within a kind (real + decoy together).
		rankSet(t, "silver", append(cardRanks(p.Silvers), p.DecoySilvers...))
		rankSet(t, "gold", append(cardRanks(p.Golds), p.DecoyGolds...))
		// Shared rank across kinds yields separate entries (separate slices).
		if len(p.Silvers) != lvl.SilverCount || len(p.Golds) != lvl.GoldCount {
			t.Fatalf("real special counts changed by extras")
		}
	})
}

// Feature: card-finders, Property 5: decoys never collide with real specials of the same kind.
func TestPropDecoysDisjointFromReal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevelWithExtras(t)
		seed := rapid.Int64().Draw(t, "seed")
		e := newDealtEngine(t, seed, lvl)
		p := e.state.SpecialPool

		realS := rankSet(t, "real silver", cardRanks(p.Silvers))
		for _, r := range p.DecoySilvers {
			if realS[r] {
				t.Fatalf("decoy silver rank %d collides with a real silver", r)
			}
		}
		realG := rankSet(t, "real gold", cardRanks(p.Golds))
		for _, r := range p.DecoyGolds {
			if realG[r] {
				t.Fatalf("decoy gold rank %d collides with a real gold", r)
			}
		}
	})
}
