package engine

import (
	"math/rand"

	"pgregory.net/rapid"
)

// Shared harness for the frontend-rework property tests: a seeded, dealt game
// plus a random stream of legal actions (see fr_pick_test.go / fr_run_test.go).

// fataler is satisfied by *rapid.T and *testing.T.
type fataler interface {
	Fatalf(format string, args ...any)
}

// drawLevel draws a valid level exercising silvers, golds, and varied lives.
func drawLevel(t *rapid.T) Level {
	return Level{
		Number:      1,
		Name:        "PBT",
		SilverCount: 2 * rapid.IntRange(0, 5).Draw(t, "silverPairs"),
		GoldCount:   rapid.IntRange(0, 11).Draw(t, "gold"),
		StartLives:  rapid.IntRange(1, 25).Draw(t, "lives"),
	}
}

// newDealtEngine creates a game at lvl, seats four players, and returns the
// engine in PhaseOpening (cards dealt, slots computed).
func newDealtEngine(t fataler, seed int64, lvl Level) *Engine {
	e := NewEngine([]Level{lvl}, rand.New(rand.NewSource(seed)))
	code, err := e.CreateGame(lvl.Number)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, _, err := e.Join(code); err != nil {
			t.Fatalf("Join %d: %v", i, err)
		}
	}
	return e
}

// ownFaceDown lists seat's face-down cards in HandOrder (deterministic).
func ownFaceDown(e *Engine, seat SeatID) []CardID {
	var out []CardID
	for _, id := range e.state.HandOrder[seat] {
		if e.state.Cards[id].Facing == FaceDown {
			out = append(out, id)
		}
	}
	return out
}

// hiddenNonGold lists seat's face-down, non-gold cards in HandOrder.
func hiddenNonGold(e *Engine, seat SeatID) []CardID {
	var out []CardID
	for _, id := range ownFaceDown(e, seat) {
		if e.state.Cards[id].Card.Kind != SpecialKindGold {
			out = append(out, id)
		}
	}
	return out
}

// validSelfReveals lists the self-reveal targets the engine would accept now,
// using the engine's own side-effect-free target resolvers.
func validSelfReveals(e *Engine, seat SeatID) []SelfRevealTarget {
	var out []SelfRevealTarget
	if _, err := e.goldSelfRevealTargets(seat); err == nil {
		out = append(out, SelfRevealTarget{Kind: SpecialKindGold})
	}
	if _, err := e.valueSelfRevealTargets(seat, SpecialKindSilver, 0); err == nil {
		out = append(out, SelfRevealTarget{Kind: SpecialKindSilver})
	}
	for r := Rank(2); r <= 13; r++ {
		if _, err := e.valueSelfRevealTargets(seat, SpecialKindNone, r); err == nil {
			out = append(out, SelfRevealTarget{Kind: SpecialKindNone, Rank: r})
		}
	}
	return out
}

// gameObs observes a random legal game. onState fires on the freshly dealt
// engine and after every applied action; onApply fires after every Apply.
type gameObs struct {
	onState func(t *rapid.T, e *Engine)
	onApply func(t *rapid.T, e *Engine, seat SeatID, a Action, err error)
}

// runRandomGame seeds a game, then streams random legal actions until it ends.
// A legal action that Apply rejects fails the test.
func runRandomGame(t *rapid.T, obs gameObs) *Engine {
	return runRandomGameAt(t, drawLevel(t), obs)
}

// runRandomGameAt is runRandomGame on a caller-supplied level.
func runRandomGameAt(t *rapid.T, lvl Level, obs gameObs) *Engine {
	seed := rapid.Int64().Draw(t, "seed")
	e := newDealtEngine(t, seed, lvl)
	if obs.onState != nil {
		obs.onState(t, e)
	}
	for step := 0; step < 600 && e.state.Phase != PhaseEnded; step++ {
		seat, a := pickAction(t, e)
		_, err := e.Apply(seat, a)
		if obs.onApply != nil {
			obs.onApply(t, e, seat, a, err)
		}
		if err != nil {
			t.Fatalf("legal action %+v by seat %d rejected: %v", a, seat, err)
		}
		if obs.onState != nil {
			obs.onState(t, e)
		}
	}
	return e
}

// pickAction draws the next legal action for whichever seat must act. Playing
// turns are biased by an omniscient mode so hits, double-hit safe calls and
// self-reveals occur often, not just misses.
func pickAction(t *rapid.T, e *Engine) (SeatID, Action) {
	s := e.state
	switch s.Phase {
	case PhaseOpening:
		var suited []CardID
		for _, id := range ownFaceDown(e, s.CurrentSeat) {
			if e.state.Cards[id].Card.Kind == SpecialKindNone {
				suited = append(suited, id)
			}
		}
		id := rapid.SampledFrom(suited).Draw(t, "openCard")
		return s.CurrentSeat, Action{Kind: ActionKindOpeningReveal, OpeningCard: id}
	case PhaseAwaitingResponse:
		p := s.PendingSafeChoice
		c := rapid.SampledFrom([]CardID{p.Option1, p.Option2}).Draw(t, "choice")
		return p.TargetPlayer, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: c}
	}
	seat := s.CurrentSeat
	mode := rapid.IntRange(0, 4).Draw(t, "mode")
	switch mode {
	case 1:
		if a, ok := hitGuess(t, e, seat); ok {
			return seat, a
		}
	case 2:
		if a, ok := doubleHitSafeCall(t, e, seat); ok {
			return seat, a
		}
	case 3:
		if a, ok := randomSafeCall(t, e, seat); ok {
			return seat, a
		}
	case 4:
		if a, ok := pickSelfReveal(t, e, seat); ok {
			return seat, a
		}
	}
	if a, ok := randomGuess(t, e, seat); ok {
		return seat, a
	}
	if a, ok := pickSelfReveal(t, e, seat); ok {
		return seat, a
	}
	t.Fatalf("no legal action for seat %d in phase %d", seat, s.Phase)
	return seat, Action{}
}

// targetsOf lists other seats' face-down non-gold cards, in seat/HandOrder order.
func targetsOf(e *Engine, seat SeatID) []CardID {
	var out []CardID
	for o := SeatID(0); o < seatCount; o++ {
		if o != seat {
			out = append(out, hiddenNonGold(e, o)...)
		}
	}
	return out
}

func randomGuess(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	srcs, tgts := hiddenNonGold(e, seat), targetsOf(e, seat)
	if len(srcs) == 0 || len(tgts) == 0 {
		return Action{}, false
	}
	return Action{
		Kind:       ActionKindGuess,
		SourceCard: rapid.SampledFrom(srcs).Draw(t, "src"),
		TargetCard: rapid.SampledFrom(tgts).Draw(t, "tgt"),
	}, true
}

func hitGuess(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	var hits []Action
	for _, src := range hiddenNonGold(e, seat) {
		for _, tgt := range targetsOf(e, seat) {
			if matchesAnnouncedValue(e.state.Cards[src].Card, e.state.Cards[tgt].Card) {
				hits = append(hits, Action{Kind: ActionKindGuess, SourceCard: src, TargetCard: tgt})
			}
		}
	}
	if len(hits) == 0 {
		return Action{}, false
	}
	return rapid.SampledFrom(hits).Draw(t, "hit"), true
}

func canSafeCall(e *Engine, seat SeatID) bool {
	p := e.state.Players[seat]
	return p != nil && !p.PowerUsed && len(hiddenNonGold(e, seat)) > 0
}

// doubleHitSafeCall finds a source matching two of one other seat's cards.
func doubleHitSafeCall(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	if !canSafeCall(e, seat) {
		return Action{}, false
	}
	var found []Action
	for _, src := range hiddenNonGold(e, seat) {
		for o := SeatID(0); o < seatCount; o++ {
			if o == seat {
				continue
			}
			var m []CardID
			for _, tgt := range hiddenNonGold(e, o) {
				if matchesAnnouncedValue(e.state.Cards[src].Card, e.state.Cards[tgt].Card) {
					m = append(m, tgt)
				}
			}
			if len(m) >= 2 {
				found = append(found, Action{Kind: ActionKindSafeCall, SourceCard: src, TargetCard: m[0], TargetCard2: m[1]})
			}
		}
	}
	if len(found) == 0 {
		return Action{}, false
	}
	return rapid.SampledFrom(found).Draw(t, "double"), true
}

// randomSafeCall picks any source and two distinct non-gold cards of one seat.
func randomSafeCall(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	if !canSafeCall(e, seat) {
		return Action{}, false
	}
	var seats []SeatID
	for o := SeatID(0); o < seatCount; o++ {
		if o != seat && len(hiddenNonGold(e, o)) >= 2 {
			seats = append(seats, o)
		}
	}
	if len(seats) == 0 {
		return Action{}, false
	}
	tg := hiddenNonGold(e, rapid.SampledFrom(seats).Draw(t, "scSeat"))
	i := rapid.IntRange(0, len(tg)-1).Draw(t, "i")
	j := rapid.IntRange(0, len(tg)-2).Draw(t, "j")
	if j >= i {
		j++
	}
	return Action{
		Kind:        ActionKindSafeCall,
		SourceCard:  rapid.SampledFrom(hiddenNonGold(e, seat)).Draw(t, "scSrc"),
		TargetCard:  tg[i],
		TargetCard2: tg[j],
	}, true
}

func pickSelfReveal(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	opts := validSelfReveals(e, seat)
	if len(opts) == 0 {
		return Action{}, false
	}
	return Action{
		Kind:            ActionKindSelfReveal,
		SelfRevealValue: rapid.SampledFrom(opts).Draw(t, "selfValue"),
	}, true
}
