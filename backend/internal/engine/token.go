package engine

// Face is a card's up/down orientation.
type Face uint8

const (
	FaceDown Face = iota
	FaceUp
)

// TokenKind records which property of the underlying card a token reveals. It
// never copies the card's value, so the shown value is always resolved from the
// card and can never drift.
type TokenKind uint8

const (
	// TokenKindNaturalIdentity shows the card's Rank (SpecialKindNone) or its
	// special kind (silver/gold). Other values are reserved for future modes.
	TokenKindNaturalIdentity TokenKind = 1
)

// TokenInfo is a token placed on a card, storing only which property to display.
type TokenInfo struct {
	Displayed TokenKind
}

// PlacedCard is a card in play: its identity, owning seat, facing, and optional
// token. A token only ever sits on a face-down card (opening reveal or miss
// target); a face-up card's value is already visible to everyone.
type PlacedCard struct {
	Card   Card
	Owner  SeatID
	Facing Face
	Token  *TokenInfo
}
