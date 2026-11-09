package engine

// Rank is 2..13 (11=J, 12=Q, 13=K); no rank 1. Suit is 0..3, meaningful only
// for suited (SpecialKindNone) cards and used for sort tie-breaking. SeatID is
// 0..3 clockwise. See GAME.md for terms.
type Suit uint8

type Rank uint8

// 0-4, ordered, the first to play is randomly choose
type SeatID uint8

type SpecialKind uint8

const (
	SpecialKindNone   SpecialKind = iota // suited card
	SpecialKindSilver                    // no suit, ranks 2..Q
	SpecialKindGold                      // no suit, ranks 2..Q
)

// CardID is a stable identity assigned at deck build time.
type CardID uint16

// Card is an immutable card identity.
type Card struct {
	ID   CardID
	Rank Rank
	Suit Suit // meaningful only when Kind == SpecialKindNone
	Kind SpecialKind
}

// cardHasValue is the shared value-identity predicate for guess/safe-call and
// self-reveal matching: silver matches any silver; suited matches same rank.
// Gold is never passed (gold sources are rejected upstream; gold self-reveal
// has its own path), so a gold kind safely matches nothing.
func cardHasValue(c Card, kind SpecialKind, rank Rank) bool {
	if kind == SpecialKindSilver {
		return c.Kind == SpecialKindSilver
	}
	return c.Kind == SpecialKindNone && c.Rank == rank
}

// SortValue is the display-ordering value: rank, +0.3 silver, +0.6 gold.
func (c Card) SortValue() float64 {
	switch c.Kind {
	case SpecialKindSilver:
		return float64(c.Rank) + 0.3
	case SpecialKindGold:
		return float64(c.Rank) + 0.6
	default:
		return float64(c.Rank)
	}
}
