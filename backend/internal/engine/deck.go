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

// drawKind shuffles the eligible ranks and splits off real then decoy ranks,
// so decoys never collide with real ones. Counts must be pre-validated.
func drawKind(rng *rand.Rand, real, decoy int) (realRanks, decoyRanks []Rank) {
	ranks := eligibleSpecialRankList()
	rng.Shuffle(len(ranks), func(i, j int) { ranks[i], ranks[j] = ranks[j], ranks[i] })
	return ranks[:real:real], ranks[real : real+decoy : real+decoy]
}

// BuildSpecialPool forms the Special_Pool for l using the seeded rng. Special
// CardIDs continue from the Base_Deck, silvers then golds.
func BuildSpecialPool(rng *rand.Rand, l Level) (SpecialPool, error) {
	if err := ValidateLevel(l); err != nil {
		return SpecialPool{}, fmt.Errorf("invalid level for special pool: %w", err)
	}
	silverRanks, decoySilvers := drawKind(rng, l.SilverCount, l.ExtraSilver)
	goldRanks, decoyGolds := drawKind(rng, l.GoldCount, l.ExtraGold)


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
