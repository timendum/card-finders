package engine

import (
	"reflect"
	"sort"
	"testing"

	"pgregory.net/rapid"
)

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
			for seat := SeatID(0); seat < seatCount; seat++ {
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
