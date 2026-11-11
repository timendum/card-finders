package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// remainingRanks returns the eligible special ranks not in used, ascending.
func remainingRanks(used []Rank) []Rank {
	taken := make(map[Rank]bool, len(used))
	for _, r := range used {
		taken[r] = true
	}
	var out []Rank
	for _, r := range eligibleSpecialRankList() {
		if !taken[r] {
			out = append(out, r)
		}
	}
	return out
}

// overwriteDecoys replaces e's decoy ranks directly with a random subset of the
// ranks not used by a real special of the same kind. It does not touch RNG or
// hands, so two engines with equal hands can differ only in their decoys.
func overwriteDecoys(t *rapid.T, e *Engine, label string) {
	p := &e.state.SpecialPool
	pick := func(used []Rank, name string) []Rank {
		rest := remainingRanks(used)
		if len(rest) == 0 {
			return nil
		}
		return rapid.SliceOfNDistinct(rapid.SampledFrom(rest), 0, len(rest),
			func(r Rank) Rank { return r }).Draw(t, label+name)
	}
	p.DecoySilvers = pick(cardRanks(p.Silvers), "Silvers")
	p.DecoyGolds = pick(cardRanks(p.Golds), "Golds")
}

func snapshotState(t *rapid.T, e *Engine) string {
	b, err := json.Marshal(e.state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	return string(b)
}

// allSelfRevealTargets lists every self-reveal value a client could submit.
func allSelfRevealTargets() []SelfRevealTarget {
	out := []SelfRevealTarget{{Kind: SpecialKindGold}, {Kind: SpecialKindSilver}}
	for r := Rank(2); r <= 13; r++ {
		out = append(out, SelfRevealTarget{Kind: SpecialKindNone, Rank: r})
	}
	return out
}

// Feature: card-finders, Property 7: self-reveal eligibility ignores decoys
// (two states with identical hands but different decoys accept and reject the
// same self-reveals; a rejection leaves state unchanged).
func TestPropSelfRevealIgnoresDecoys(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lvl := drawLevel(t) // zero extras: same seed => identical hands
		seed := rapid.Int64().Draw(t, "seed")
		a := newDealtEngine(t, seed, lvl)
		b := newDealtEngine(t, seed, lvl)
		overwriteDecoys(t, a, "decoyA")
		overwriteDecoys(t, b, "decoyB")

		if !reflect.DeepEqual(a.state.HandOrder, b.state.HandOrder) {
			t.Fatalf("setup: hands differ")
		}

		for step := 0; step < 600 && a.state.Phase != PhaseEnded; step++ {
			probeSelfReveals(t, a, b)
			if a.state.Phase == PhaseEnded {
				break
			}
			seat, act := pickAction(t, a)
			ra, errA := a.Apply(seat, act)
			rb, errB := b.Apply(seat, act)
			if errA != errB {
				t.Fatalf("action %+v: error %v vs %v", act, errA, errB)
			}
			if errA != nil {
				t.Fatalf("legal action %+v rejected: %v", act, errA)
			}
			if ra != rb {
				t.Fatalf("action %+v: result %v vs %v", act, ra, rb)
			}
		}
	})
}

// probeSelfReveals submits every self-reveal value for every seat to both
// engines. Outcomes must match; rejections must leave state untouched. When an
// action is accepted (identically by both) the state has moved on, so probing
// stops for this round.
func probeSelfReveals(t *rapid.T, a, b *Engine) {
	for seat := SeatID(0); seat < seatCount; seat++ {
		for _, tgt := range allSelfRevealTargets() {
			act := Action{Kind: ActionKindSelfReveal, SelfRevealValue: tgt}
			beforeA, beforeB := snapshotState(t, a), snapshotState(t, b)
			ra, errA := a.Apply(seat, act)
			rb, errB := b.Apply(seat, act)
			if errA != errB {
				t.Fatalf("seat %d self-reveal %+v: error %v vs %v", seat, tgt, errA, errB)
			}
			if errA != nil {
				if snapshotState(t, a) != beforeA || snapshotState(t, b) != beforeB {
					t.Fatalf("seat %d self-reveal %+v rejected (%v) but changed state", seat, tgt, errA)
				}
				continue
			}
			if ra != rb {
				t.Fatalf("seat %d self-reveal %+v: result %v vs %v", seat, tgt, ra, rb)
			}
			return
		}
	}
}
