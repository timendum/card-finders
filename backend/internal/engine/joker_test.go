package engine

import (
	"math/rand"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// jokerEngine builds a Playing state with four seated players and the given
// jokers drawn and ready (their four suited cards are placed face-up outside
// every hand, so they never disturb slot layouts). Cards placed afterwards are
// slotted by layout.
func jokerEngine(lives int, ranks ...Rank) *Engine {
	e := playingState(lives)
	s := e.state
	s.Slot = map[CardID]int{}
	for i := range s.Players {
		s.Players[i] = &Player{Seat: SeatID(i), Connected: true}
	}
	for n, r := range ranks {
		s.Jokers = append(s.Jokers, JokerSlot{Rank: r})
		for suit := range Suit(suitsPerRank) {
			id := CardID(1000 + n*10 + int(suit))
			s.Cards[id] = &PlacedCard{
				Card:   Card{ID: id, Rank: r, Suit: suit, Kind: SpecialKindNone},
				Owner:  3,
				Facing: FaceUp,
			}
		}
	}
	return e
}

// layout numbers every seat's slots from the order cards were placed in.
func layout(e *Engine) {
	for seat := range e.state.HandOrder {
		e.state.setHand(SeatID(seat), e.state.HandOrder[seat])
	}
}

// Jokers are drawn distinct, in 2..13, in the level's count, ascending.
func TestPropDrawJokerRanks(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, maxJokers).Draw(t, "n")
		got := drawJokerRanks(rand.New(rand.NewSource(rapid.Int64().Draw(t, "seed"))), n)
		if len(got) != n {
			t.Fatalf("drew %d jokers, want %d", len(got), n)
		}
		for i, j := range got {
			if j.Rank < minRank || j.Rank > maxRank || j.Used {
				t.Fatalf("bad joker %+v", j)
			}
			if i > 0 && got[i-1].Rank >= j.Rank {
				t.Fatalf("ranks not strictly ascending: %+v", got)
			}
		}
	})
}

func TestDrawJokerRanksSeedReproducible(t *testing.T) {
	a := drawJokerRanks(rand.New(rand.NewSource(7)), 5)
	b := drawJokerRanks(rand.New(rand.NewSource(7)), 5)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("same seed drew %v and %v", a, b)
	}
}

// Drawing jokers happens after the deal, so the deal does not depend on it.
func TestJokersDoNotChangeDeal(t *testing.T) {
	deal := func(jokers int) [4][]CardID {
		lvl := Level{Number: 1, Name: "J", SilverCount: 2, GoldCount: 1, StartLives: 3, Jokers: jokers}
		return newDealtEngine(t, 42, lvl).state.HandOrder
	}
	if !reflect.DeepEqual(deal(0), deal(6)) {
		t.Error("deal changed with the joker count")
	}
}

func TestStartGameDrawsLevelJokers(t *testing.T) {
	lvl := Level{Number: 1, Name: "J", StartLives: 3, Jokers: 6}
	e := newDealtEngine(t, 1, lvl)
	if got := len(e.state.Jokers); got != 6 {
		t.Errorf("drew %d jokers, want 6", got)
	}
	if v := e.View(0); len(v.Jokers) != 6 || v.Jokers[0].State != JokerStateInactive {
		t.Errorf("view jokers = %+v, want 6 inactive", v.Jokers)
	}
}

func TestJokerStateTransitions(t *testing.T) {
	e := jokerEngine(3, 7)
	s := e.state
	if st := s.jokerStateOf(s.Jokers[0]); st != JokerStateReady {
		t.Fatalf("state = %d, want ready once all four 7s are face-up", st)
	}
	s.Cards[1003].Facing = FaceDown
	if st := s.jokerStateOf(s.Jokers[0]); st != JokerStateInactive {
		t.Errorf("state = %d, want inactive with a 7 face-down", st)
	}
	s.Cards[1003].Facing = FaceUp
	s.Jokers[0].Used = true
	if st := s.jokerStateOf(s.Jokers[0]); st != JokerStateUsed {
		t.Errorf("state = %d, want used", st)
	}
}

func TestJokerUsableOnlyWhenPlaying(t *testing.T) {
	e := jokerEngine(3, 7)
	s := e.state
	if !s.jokerUsable(7) {
		t.Fatal("ready joker should be usable while Playing")
	}
	s.Phase = PhaseAwaitingResponse
	if s.jokerUsable(7) {
		t.Error("joker usable while a choice is pending")
	}
	s.Phase = PhasePlaying
	if s.jokerUsable(8) {
		t.Error("undrawn joker reported usable")
	}
}

func TestJokerUnavailableRejected(t *testing.T) {
	e := jokerEngine(3, 7)
	e.state.Jokers[0].Used = true
	if _, err := e.Apply(0, Action{Kind: ActionKindJoker, Joker: 7}); err != ErrJokerUnavailable {
		t.Errorf("err = %v, want ErrJokerUnavailable for a used joker", err)
	}
	if _, err := e.Apply(0, Action{Kind: ActionKindJoker, Joker: 9}); err != ErrJokerUnavailable {
		t.Errorf("err = %v, want ErrJokerUnavailable for an undrawn joker", err)
	}
}

// checkJokerInvariants asserts the board invariants the jokers must preserve at
// every reachable state.
func checkJokerInvariants(t *rapid.T, e *Engine) {
	s := e.state
	total := 0
	faceDown := map[[2]int]int{}
	for seat := range SeatID(seatCount) {
		hand := s.handBySlot(seat)
		total += len(hand)
		prev := -1.0
		for i, id := range hand {
			pc := s.Cards[id]
			if pc.Owner != seat || s.Slot[id] != i+1 {
				t.Fatalf("card %d: owner %d slot %d, want seat %d slot %d", id, pc.Owner, s.Slot[id], seat, i+1)
			}
			if sv := pc.Card.SortValue(); sv < prev {
				t.Fatalf("seat %d hand not sorted at slot %d", seat, i+1)
			} else {
				prev = sv
			}
			if pc.Facing == FaceDown && pc.Card.Kind != SpecialKindGold {
				key := [2]int{int(pc.Card.Kind), 0} // specials count by kind only
				if pc.Card.Kind == SpecialKindNone {
					key[1] = int(pc.Card.Rank)
				}
				faceDown[key]++
			}
			if pc.Facing == FaceUp && pc.Token.isLink() {
				t.Fatalf("face-up card %d carries a link token", id)
			}
		}
	}
	if total != len(s.Cards) {
		t.Fatalf("hands hold %d cards, board has %d", total, len(s.Cards))
	}
	for v, n := range faceDown {
		if n%2 != 0 {
			t.Fatalf("value %v has %d face-down cards, want an even count", v, n)
		}
	}

	linked := 0
	for _, p := range s.linkPairs() {
		a, b := s.Cards[p[0]], s.Cards[p[1]]
		if a.Owner != b.Owner || s.Slot[p[1]]-s.Slot[p[0]] != 1 || a.Token.Displayed != b.Token.Displayed {
			t.Fatalf("malformed link pair %v", p)
		}
		if sameIdentity(a.Card, b.Card) != (a.Token.Displayed == TokenKindSame) {
			t.Fatalf("link pair %v claims %d but the cards say otherwise", p, a.Token.Displayed)
		}
		linked += 2
	}
	count := 0
	for _, pc := range s.Cards {
		if pc.Token.isLink() {
			count++
		}
	}
	if count != linked {
		t.Fatalf("%d link tokens but only %d belong to a pair", count, linked)
	}

	for _, j := range s.Jokers {
		if j.Used && !s.rankComplete(j.Rank) {
			t.Fatalf("joker %d used before its rank was complete", j.Rank)
		}
	}
	if v := e.View(0); len(v.Jokers) != len(s.Jokers) || v.Shield != s.Shield {
		t.Fatalf("view jokers/shield out of sync: %+v", v)
	}
}

// Jokers keep the board consistent in whole random games: hands stay sorted and
// slotted, face-down counts stay even, link tokens stay truthful pairs, a
// joker is only used once its rank is complete, and life rises only by joker 7.
func TestPropJokerGameInvariants(t *testing.T) {
	jokerUses := 0
	usesByRank := map[Rank]int{}
	rapid.Check(t, func(t *rapid.T) {
		prevLife := -1
		runRandomGameAt(t, drawJokerLevel(t), gameObs{
			onState: func(t *rapid.T, e *Engine) {
				checkJokerInvariants(t, e)
				if prevLife >= 0 && e.state.LifePool > prevLife && e.state.LastAction.Joker != JokerExtraLife {
					t.Fatalf("life rose %d -> %d without joker 7", prevLife, e.state.LifePool)
				}
				prevLife = e.state.LifePool
			},
			onApply: func(t *rapid.T, e *Engine, seat SeatID, a Action, err error) {
				if a.Kind == ActionKindJoker && err == nil {
					jokerUses++
					usesByRank[a.Joker]++
				}
			},
		})
	})
	t.Logf("joker uses by rank: %v", usesByRank)
	if jokerUses == 0 {
		t.Error("no joker was ever used; the generator is not exercising jokers")
	}
}

// drawJokerLevel is drawLevel with plenty of jokers.
func drawJokerLevel(t *rapid.T) Level {
	l := drawLevel(t)
	l.Jokers = rapid.IntRange(4, maxJokers).Draw(t, "jokers")
	return l
}

// othersFaceDown lists the other seats that still hold a face-down card.
func othersFaceDown(e *Engine, seat SeatID) []SeatID {
	var out []SeatID
	for o := range SeatID(seatCount) {
		if o != seat && len(ownFaceDown(e, o)) > 0 {
			out = append(out, o)
		}
	}
	return out
}

// shieldGuess is a guess that may target any face-down card of another seat,
// gold included (legal, and safe only under an armed Shield).
func shieldGuess(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	srcs := hiddenNonGold(e, seat)
	var tgts []CardID
	for _, o := range othersFaceDown(e, seat) {
		tgts = append(tgts, ownFaceDown(e, o)...)
	}
	if len(srcs) == 0 || len(tgts) == 0 {
		return Action{}, false
	}
	return Action{
		Kind:        ActionKindGuess,
		SourceCards: []CardID{rapid.SampledFrom(srcs).Draw(t, "shieldSrc")},
		TargetCards: []CardID{rapid.SampledFrom(tgts).Draw(t, "shieldTgt")},
	}, true
}

// pickJokerAction returns a legal activation of some usable joker, if any.
func pickJokerAction(t *rapid.T, e *Engine, seat SeatID) (Action, bool) {
	s := e.state
	var cands []Action
	add := func(a Action) {
		a.Kind = ActionKindJoker
		cands = append(cands, a)
	}
	own := ownFaceDown(e, seat)
	srcs := hiddenNonGold(e, seat)
	others := othersFaceDown(e, seat)

	for _, j := range s.Jokers {
		if !s.jokerUsable(j.Rank) {
			continue
		}
		switch j.Rank {
		case JokerDiffer, JokerMatch:
			hand := s.handBySlot(seat)
			for i := 0; i+1 < len(hand); i++ {
				a, b := s.Cards[hand[i]], s.Cards[hand[i+1]]
				if a.Facing != FaceDown || b.Facing != FaceDown || a.Token != nil || b.Token != nil {
					continue
				}
				if sameIdentity(a.Card, b.Card) == (j.Rank == JokerMatch) {
					add(Action{Joker: j.Rank, SourceCards: []CardID{hand[i], hand[i+1]}})
				}
			}
		case JokerSwap:
			for _, o := range others {
				for _, c := range own {
					add(Action{Joker: j.Rank, SourceCards: []CardID{c}, TargetSeats: []SeatID{o}})
				}
			}
		case JokerTripleCall:
			for _, o := range others {
				if tg := ownFaceDown(e, o); len(tg) >= 3 {
					for _, src := range srcs {
						add(Action{Joker: j.Rank, SourceCards: []CardID{src}, TargetCards: []CardID{tg[0], tg[1], tg[2]}})
					}
				}
			}
		case JokerSelfMark:
			for _, c := range own {
				if pc := s.Cards[c]; pc.Card.Kind == SpecialKindNone && pc.Token == nil {
					add(Action{Joker: j.Rank, SourceCards: []CardID{c}})
				}
			}
		case JokerSweep:
			for _, o := range others {
				for _, c := range srcs {
					if s.Cards[c].Card.Kind == SpecialKindNone {
						add(Action{Joker: j.Rank, SourceCards: []CardID{c}, TargetSeats: []SeatID{o}})
					}
				}
			}
		case JokerExtraLife:
			add(Action{Joker: j.Rank})
		case JokerSecondWind:
			var spent []SeatID
			for i, p := range s.Players {
				if p.PowerUsed {
					spent = append(spent, SeatID(i))
				}
			}
			if len(spent) <= 2 {
				add(Action{Joker: j.Rank})
			} else {
				add(Action{Joker: j.Rank, TargetSeats: []SeatID{spent[0], spent[1]}})
			}
		case JokerCensus:
			add(Action{Joker: j.Rank, SelfRevealValue: SelfRevealTarget{Rank: Rank(rapid.IntRange(2, 13).Draw(t, "censusRank"))}})
		case JokerShield:
			if len(srcs) > 0 && len(others) > 0 {
				add(Action{Joker: j.Rank})
			}
		case JokerReverseCall:
			for _, o := range others {
				for _, a := range srcs {
					for _, b := range srcs {
						if a < b && !sameIdentity(s.Cards[a].Card, s.Cards[b].Card) {
							add(Action{Joker: j.Rank, SourceCards: []CardID{a, b}, TargetCards: []CardID{ownFaceDown(e, o)[0]}})
						}
					}
				}
			}
		case JokerPass:
			for _, o := range others {
				add(Action{Joker: j.Rank, TargetSeats: []SeatID{o}})
			}
		}
	}
	if len(cands) == 0 {
		return Action{}, false
	}
	return rapid.SampledFrom(cands).Draw(t, "joker"), true
}
