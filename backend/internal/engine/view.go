package engine

import "sort"

// Projection is the single chokepoint enforcing visibility:
// Properties 20 and 21.

// CardValue is a card's concrete public value. Suit is meaningful only for
// SpecialKindNone.
type CardValue struct {
	Rank Rank
	Kind SpecialKind
	Suit Suit
}

// Outcome classifies how the last action resolved. OutcomeAwaitingChoice
// marks a double-hit safe call whose Pair_Reveal is still pending the target's
// choice, so no cards are revealed and no life is lost yet.
type Outcome uint8

const (
	OutcomeNone Outcome = iota
	OutcomeHit
	OutcomeMiss
	OutcomeAwaitingChoice
	// OutcomeAwaitingToken marks a double-miss safe call (life already lost)
	// whose token placement is pending the target's choice.
	OutcomeAwaitingToken
	// OutcomeGoldHit marks a targeted gold card: the game ended as a loss.
	OutcomeGoldHit
)

// LastAction records the most recently applied action for display. Seq is a
// monotonic counter so clients can detect a new action even when the resolved
// fields coincide. AnnouncedValue and SourceCard are nil for actions that carry
// neither (e.g. opening reveal); Revealed lists cards flipped face-up by this
// action and LivesLost is the life delta (non-negative).
type LastAction struct {
	Seq            uint64
	Actor          SeatID
	Kind           ActionKind
	AnnouncedValue *CardValue
	Targets        []CardID
	Outcome        Outcome
	Revealed       []CardID
	LivesLost      int
	SourceCard     *CardID
}

// ViewCard is one card as projected for a single viewer. Value is non-nil only
// when the viewer may see the card (face-up or owned). TokenDisplay is the
// token's resolved public value, non-nil exactly when Token != nil, so a hidden
// card still shows its token value without exposing Value.
type ViewCard struct {
	ID           CardID
	Owner        SeatID
	Slot         int
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
	SpecialPool ViewSpecialPool
	LifePool    int
	MaxLife     int // Level.StartLives
	CurrentSeat SeatID
	YourTurn    bool
	PowerUsed   [4]bool
	Connected   [4]bool  // At next Player related field, create a substructure.
	Cards       []ViewCard
	Result      Result
	SeatsFilled int
	LastAction  *LastAction
	Log         []LogEntry

	// ResponsePlayer is the seat that must choose; meaningful only when
	// HasResponsePlayer. Identical for every viewer.
	ResponsePlayer    SeatID
	HasResponsePlayer bool
}

// View projects state into the per-player view for seat. With no game
// (state == nil) it returns a zero-valued view carrying only the seat.
func (e *Engine) View(seat SeatID) PlayerView {
	if e.state == nil {
		return PlayerView{Seat: seat, SpecialPool: normalizePool(SpecialPool{})}
	}
	s := e.state

	v := PlayerView{
		Seat:        seat,
		Phase:       s.Phase,
		Level:       s.Level,
		SpecialPool: normalizePool(s.SpecialPool),
		LifePool:    s.LifePool,
		MaxLife:     s.Level.StartLives,
		CurrentSeat: s.CurrentSeat,
		YourTurn:    seat == s.CurrentSeat && (s.Phase == PhaseOpening || s.Phase == PhasePlaying),
		Result:      s.GameResult,
		SeatsFilled: s.seatsFilled(),
	}
	for i := range 4 {
		if p := s.Players[i]; p != nil {
			v.PowerUsed[i] = p.PowerUsed
			v.Connected[i] = p.Connected
		}
	}
	if s.Phase == PhaseAwaitingResponse && s.PendingSafeChoice != nil {
		v.ResponsePlayer = s.PendingSafeChoice.TargetPlayer
		v.HasResponsePlayer = true
	}
	v.Cards = projectCards(s, seat)
	v.LastAction = projectLastAction(s, seat)
	v.Log = append([]LogEntry{}, s.Log...)
	return v
}

// projectLastAction projects GameState.LastAction for one viewer, gating only
// SourceCard: it is visible to the actor, or to everyone once the source card
// is face-up; otherwise (non-actors on a miss or while the double-hit choice is
// pending) it is hidden. AnnouncedValue and all other fields stay public.
func projectLastAction(s *GameState, seat SeatID) *LastAction {
	if s.LastAction == nil {
		return nil
	}
	// Copy before nulling so the shared authoritative record is never mutated.
	la := *s.LastAction
	if la.SourceCard != nil && seat != la.Actor {
		if pc, ok := s.Cards[*la.SourceCard]; !ok || pc.Facing != FaceUp {
			la.SourceCard = nil
		}
	}
	return &la
}

// ViewSpecialPool is the public Special_Pool as projected into a view: bare
// values with no CardIDs, so the pool cannot be correlated with dealt cards.
type ViewSpecialPool struct {
	Silvers      []CardValue // real and decoy entries, ascending rank
	Golds        []CardValue // real and decoy entries, ascending rank
	SilverInPlay int         // number of real silvers dealt
	GoldInPlay   int         // number of real golds dealt
}

// normalizePool projects p to values only. Real and decoy entries are merged
// and sorted ascending by rank per kind, so list order never reveals which are
// decoys. The result slices are never nil, so the view always marshals
// Silvers/Golds as [] rather than null.
func normalizePool(p SpecialPool) ViewSpecialPool {
	out := ViewSpecialPool{
		Silvers:      make([]CardValue, 0, len(p.Silvers)+len(p.DecoySilvers)),
		Golds:        make([]CardValue, 0, len(p.Golds)+len(p.DecoyGolds)),
		SilverInPlay: len(p.Silvers),
		GoldInPlay:   len(p.Golds),
	}
	for _, c := range p.Silvers {
		out.Silvers = append(out.Silvers, naturalValue(c))
	}
	for _, r := range p.DecoySilvers {
		out.Silvers = append(out.Silvers, CardValue{Rank: r, Kind: SpecialKindSilver})
	}
	for _, c := range p.Golds {
		out.Golds = append(out.Golds, naturalValue(c))
	}
	for _, r := range p.DecoyGolds {
		out.Golds = append(out.Golds, CardValue{Rank: r, Kind: SpecialKindGold})
	}
	byRank := func(s []CardValue) {
		sort.Slice(s, func(i, j int) bool { return s[i].Rank < s[j].Rank })
	}
	byRank(out.Silvers)
	byRank(out.Golds)
	return out
}

// projectCards builds the ViewCard slice, ordered by (Owner, Slot). Ids are
// opaque and nothing relies on Cards order; the sort is a wire-debuggability
// convenience, as the client regroups by owner and sorts by Slot itself.
func projectCards(s *GameState, seat SeatID) []ViewCard {
	cards := make([]ViewCard, 0, len(s.Cards))
	for _, pc := range s.Cards {
		cards = append(cards, projectCard(pc, s.Slot[pc.Card.ID], seat))
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Owner != cards[j].Owner {
			return cards[i].Owner < cards[j].Owner
		}
		return cards[i].Slot < cards[j].Slot
	})
	return cards
}

// projectCard projects a single PlacedCard for one viewer.
func projectCard(pc *PlacedCard, slot int, seat SeatID) ViewCard {
	vc := ViewCard{
		ID:     pc.Card.ID,
		Owner:  pc.Owner,
		Slot:   slot,
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
