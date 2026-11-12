package engine

import (
	"math/rand"
	"sort"
)

// maxJokers is the number of joker ranks (2..13).
const maxJokers = 12

// Joker ranks, named by the power they grant. The backend only ever speaks in
// numeric ranks (J=11, Q=12, K=13); display names live in the frontend.
const (
	JokerDiffer      Rank = 2  // link two adjacent own cards as different
	JokerSwap        Rank = 3  // exchange a face-down card with another seat
	JokerTripleCall  Rank = 4  // safe call on three cards
	JokerSelfMark    Rank = 5  // token on an own face-down card
	JokerSweep       Rank = 6  // call a rank against a whole hand
	JokerExtraLife   Rank = 7  // +1 life
	JokerSecondWind  Rank = 8  // restore Safe_Call powers
	JokerCensus      Rank = 9  // ask every seat about a rank
	JokerShield      Rank = 10 // next guess costs no life and cannot end the game
	JokerReverseCall Rank = 11 // two own cards against one target card
	JokerPass        Rank = 12 // skip the turn, choose the next seat
	JokerMatch       Rank = 13 // link two adjacent own cards as the same
)

// JokerSlot is a joker drawn for the game. Only Used is stored; whether it is
// inactive or ready is derived from the board (see jokerStateOf).
type JokerSlot struct {
	Rank Rank
	Used bool
}

// JokerState is the derived lifecycle of a joker, as shown in the view.
type JokerState uint8

const (
	JokerStateInactive JokerState = iota // its rank is not yet fully revealed
	JokerStateReady                      // rank fully revealed, not yet used
	JokerStateUsed
)

// JokerView is one joker as projected for every viewer (jokers are public).
type JokerView struct {
	Rank  Rank
	State JokerState
}

// drawJokerRanks draws n distinct ranks from 2..13, ascending.
func drawJokerRanks(rng *rand.Rand, n int) []JokerSlot {
	ranks := make([]Rank, 0, maxJokers)
	for r := minRank; r <= maxRank; r++ {
		ranks = append(ranks, r)
	}
	rng.Shuffle(len(ranks), func(i, j int) { ranks[i], ranks[j] = ranks[j], ranks[i] })
	ranks = ranks[:n]
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })
	out := make([]JokerSlot, n)
	for i, r := range ranks {
		out[i] = JokerSlot{Rank: r}
	}
	return out
}

// rankComplete reports whether all four suited cards of rank are face-up.
func (s *GameState) rankComplete(rank Rank) bool {
	n := 0
	for _, pc := range s.Cards {
		if pc.Card.Kind == SpecialKindNone && pc.Card.Rank == rank && pc.Facing == FaceUp {
			n++
		}
	}
	return n == suitsPerRank
}

// jokerStateOf derives a joker's state: used wins, else ready once its rank is
// completely revealed.
func (s *GameState) jokerStateOf(j JokerSlot) JokerState {
	switch {
	case j.Used:
		return JokerStateUsed
	case s.rankComplete(j.Rank):
		return JokerStateReady
	default:
		return JokerStateInactive
	}
}

// jokerSlot returns the drawn joker of rank, or nil when it was not drawn.
func (s *GameState) jokerSlot(rank Rank) *JokerSlot {
	for i := range s.Jokers {
		if s.Jokers[i].Rank == rank {
			return &s.Jokers[i]
		}
	}
	return nil
}

// jokerUsable reports whether rank may be activated now: drawn, ready, and the
// game is waiting for a user action (Playing, never while a choice is pending).
func (s *GameState) jokerUsable(rank Rank) bool {
	j := s.jokerSlot(rank)
	return j != nil && s.Phase == PhasePlaying && s.jokerStateOf(*j) == JokerStateReady
}
