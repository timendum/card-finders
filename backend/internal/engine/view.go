package engine

import "sort"

// Projection is the single chokepoint enforcing visibility: see AGENTS.md
// Properties 20 and 21.

// CardValue is a card's concrete public value. Suit is meaningful only for
// SpecialKindNone.
type CardValue struct {
	Rank Rank
	Kind SpecialKind
	Suit Suit
}

// ViewCard is one card as projected for a single viewer. Value is non-nil only
// when the viewer may see the card (face-up or owned). TokenDisplay is the
// token's resolved public value, non-nil exactly when Token != nil, so a hidden
// card still shows its token value without exposing Value.
type ViewCard struct {
	ID           CardID
	Owner        SeatID
	Facing       Face
	Value        *CardValue
	Token        *TokenInfo
	TokenDisplay *CardValue
}

// PlayerView is the per-seat snapshot the transport sends on connect and after
// every state change. Cards is already projected, so hidden values never appear.
type PlayerView struct {
	Seat        SeatID
	Phase       Phase
	Level       Level
	SpecialPool SpecialPool
	LifePool    int
	MaxLife     int // Level.StartLives
	CurrentSeat SeatID
	YourTurn    bool
	PowerUsed   [4]bool
	Cards       []ViewCard
	Result      Result
	SeatsFilled int
}

// View projects state into the per-player view for seat. With no game
// (state == nil) it returns a zero-valued view carrying only the seat.
func (e *Engine) View(seat SeatID) PlayerView {
	if e.state == nil {
		return PlayerView{Seat: seat}
	}
	s := e.state

	v := PlayerView{
		Seat:        seat,
		Phase:       s.Phase,
		Level:       s.Level,
		SpecialPool: s.SpecialPool,
		LifePool:    s.LifePool,
		MaxLife:     s.Level.StartLives,
		CurrentSeat: s.CurrentSeat,
		YourTurn:    seat == s.CurrentSeat && (s.Phase == PhaseOpening || s.Phase == PhasePlaying),
		Result:      s.GameResult,
		SeatsFilled: s.seatsFilled(),
	}
	for i := 0; i < 4; i++ {
		if p := s.Players[i]; p != nil {
			v.PowerUsed[i] = p.PowerUsed
		}
	}
	v.Cards = projectCards(s, seat)
	return v
}

// projectCards builds the ViewCard slice sorted by CardID for stable output.
func projectCards(s *GameState, seat SeatID) []ViewCard {
	ids := make([]CardID, 0, len(s.Cards))
	for id := range s.Cards {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	cards := make([]ViewCard, 0, len(ids))
	for _, id := range ids {
		cards = append(cards, projectCard(s.Cards[id], seat))
	}
	return cards
}

// projectCard projects a single PlacedCard for one viewer.
func projectCard(pc *PlacedCard, seat SeatID) ViewCard {
	vc := ViewCard{
		ID:     pc.Card.ID,
		Owner:  pc.Owner,
		Facing: pc.Facing,
	}
	if pc.Facing == FaceUp || pc.Owner == seat {
		val := naturalValue(pc.Card)
		vc.Value = &val
	}
	if pc.Token != nil {
		vc.Token = pc.Token
		td := resolveTokenDisplay(pc.Card, *pc.Token)
		vc.TokenDisplay = &td
	}
	return vc
}

// naturalValue is the card's full concrete value.
func naturalValue(c Card) CardValue {
	return CardValue{Rank: c.Rank, Kind: c.Kind, Suit: c.Suit}
}

// resolveTokenDisplay resolves the public value a token shows per its Displayed
// code. Unknown codes fall back to the natural identity.
func resolveTokenDisplay(c Card, t TokenInfo) CardValue {
	switch t.Displayed {
	case TokenKindNaturalIdentity:
		return naturalValue(c)
	default:
		return naturalValue(c)
	}
}
