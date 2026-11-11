package engine

import (
	"fmt"
	"math/rand"
)

const (
	minRank        Rank = 2
	maxRank        Rank = 13 // king
	maxSpecialRank Rank = 12 // queen
	suitsPerRank        = 4
	baseDeckSize        = 48 // 4 suits x 12 ranks 2..13
)

// SpecialPool entries are the same Card values (same CardID) later dealt into
// the combined deck, so the pool is the public announcement of which specials
// exist rather than a parallel type.
type SpecialPool struct {
	Silvers      []Card
	Golds        []Card
	DecoySilvers []Rank
	DecoyGolds   []Rank
}

// BuildBaseDeck returns the 48-card Base_Deck. IDs are assigned in rank-major,
// suit-minor order so the same deck always yields the same identities.
func BuildBaseDeck() []Card {
	deck := make([]Card, 0, baseDeckSize)
	var id CardID
	for r := minRank; r <= maxRank; r++ {
		for s := range Suit(suitsPerRank) {
			deck = append(deck, Card{
				ID:   id,
				Rank: r,
				Suit: s,
				Kind: SpecialKindNone,
			})
			id++
		}
	}
	return deck
}

// eligibleSpecialRankList returns the ranks a special may take (2..queen).
func eligibleSpecialRankList() []Rank {
	ranks := make([]Rank, 0, eligibleSpecialRanks)
	for r := minRank; r <= maxSpecialRank; r++ {
		ranks = append(ranks, r)
	}
	return ranks
}

// remainingRanks returns eligibleSpecialRankList() minus used, ascending.
func remainingRanks(used []Rank) []Rank {
	taken := make(map[Rank]bool, len(used))
	for _, r := range used {
		taken[r] = true
	}
	out := make([]Rank, 0, eligibleSpecialRanks)
	for _, r := range eligibleSpecialRankList() {
		if !taken[r] {
			out = append(out, r)
		}
	}
	return out
}

// drawDistinctRanks returns n distinct eligible special ranks chosen uniformly
// at random. It errors on out-of-range n rather than silently producing
// duplicates.
func drawDistinctRanks(rng *rand.Rand, n int) ([]Rank, error) {
	return drawDistinctFrom(rng, eligibleSpecialRankList(), n)
}

// drawDistinctFrom returns n distinct ranks chosen uniformly from candidates
// by partial Fisher-Yates. It consumes exactly n rng values (none when n == 0)
// and does not modify the caller's slice.
func drawDistinctFrom(rng *rand.Rand, candidates []Rank, n int) ([]Rank, error) {
	ranks := append([]Rank(nil), candidates...)
	if n < 0 || n > len(ranks) {
		return nil, fmt.Errorf("cannot draw %d distinct ranks from %d eligible ranks", n, len(ranks))
	}
	// Partial Fisher-Yates: shuffle the first n positions, then take them.
	for i := range n {
		j := i + rng.Intn(len(ranks)-i)
		ranks[i], ranks[j] = ranks[j], ranks[i]
	}
	return ranks[:n], nil
}

// BuildSpecialPool forms the Special_Pool for l using the seeded rng. Ranks are
// distinct within a kind; a silver and a gold may share a rank across kinds.
// Special CardIDs continue sequentially from the Base_Deck, silvers then golds,
// so pool entries carry the identities the combined deck later uses.
// BuildSpecialPool re-validates l rather than trusting the caller.
func BuildSpecialPool(rng *rand.Rand, l Level) (SpecialPool, error) {
	if err := ValidateLevel(l); err != nil {
		return SpecialPool{}, fmt.Errorf("invalid level for special pool: %w", err)
	}

	silverRanks, err := drawDistinctRanks(rng, l.SilverCount)
	if err != nil {
		return SpecialPool{}, fmt.Errorf("silver: %w", err)
	}
	goldRanks, err := drawDistinctRanks(rng, l.GoldCount)
	if err != nil {
		return SpecialPool{}, fmt.Errorf("gold: %w", err)
	}

	// Decoys are drawn after all real draws so zero extras consume no RNG and
	// real draws stay identical. Each kind's decoys avoid that kind's real ranks.
	decoySilvers, err := drawDistinctFrom(rng, remainingRanks(silverRanks), l.ExtraSilver)
	if err != nil {
		return SpecialPool{}, fmt.Errorf("decoy silver: %w", err)
	}
	decoyGolds, err := drawDistinctFrom(rng, remainingRanks(goldRanks), l.ExtraGold)
	if err != nil {
		return SpecialPool{}, fmt.Errorf("decoy gold: %w", err)
	}

	nextID := CardID(baseDeckSize)
	pool := SpecialPool{
		Silvers:      make([]Card, 0, l.SilverCount),
		Golds:        make([]Card, 0, l.GoldCount),
		DecoySilvers: decoySilvers,
		DecoyGolds:   decoyGolds,
	}
	for _, r := range silverRanks {
		pool.Silvers = append(pool.Silvers, Card{ID: nextID, Rank: r, Kind: SpecialKindSilver})
		nextID++
	}
	for _, r := range goldRanks {
		pool.Golds = append(pool.Golds, Card{ID: nextID, Rank: r, Kind: SpecialKindGold})
		nextID++
	}
	return pool, nil
}
