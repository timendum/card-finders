package engine

const seatCount = 4

// nextSeat returns the seat clockwise after s: the single definition of
// clockwise order, reused by the opening sequence and turn advancement.
func nextSeat(s SeatID) SeatID {
	return SeatID((uint8(s) + 1) % seatCount)
}

// combinedDeck returns the base deck followed by the formed pool's silvers and
// golds. SpecialPool must already be set.
func (e *Engine) combinedDeck() []Card {
	pool := e.state.SpecialPool
	deck := make([]Card, 0, baseDeckSize+len(pool.Silvers)+len(pool.Golds))
	deck = append(deck, BuildBaseDeck()...)
	deck = append(deck, pool.Silvers...)
	deck = append(deck, pool.Golds...)
	return deck
}

// shuffle reorders deck in place (Fisher-Yates) using the seeded RNG, so a seed
// reproduces the ordering.
func (e *Engine) shuffle(deck []Card) {
	for i := len(deck) - 1; i > 0; i-- {
		j := e.rng.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

// startGame runs the fourth-join transition: form pool, deal, pick first player,
// reach PhaseOpening. On any failure it reverts fully to PhaseWaiting with no
// dealt hands, so the triggering Join surfaces the error cleanly.
func (e *Engine) startGame() error {
	e.state.Phase = PhaseSpecialPool
	// Level was validated at CreateGame, so a well-formed level does not error.
	pool, err := BuildSpecialPool(e.rng, e.state.Level)
	if err != nil {
		e.state.Phase = PhaseWaiting
		return err
	}
	e.state.SpecialPool = pool

	deck := e.combinedDeck()
	e.shuffle(deck)
	if err := e.dealIfSufficient(deck); err != nil {
		e.state.SpecialPool = SpecialPool{}
		e.state.Phase = PhaseWaiting
		return err
	}
	e.state.Phase = PhaseDealing

	e.state.CurrentSeat = SeatID(e.rng.Intn(seatCount))

	e.state.Phase = PhaseOpening
	return nil
}

// dealIfSufficient deals deck across the four seats, or returns
// ErrInsufficientCards without dealing any card when len(deck) < seatCount.
func (e *Engine) dealIfSufficient(deck []Card) error {
	if len(deck) < seatCount {
		return ErrInsufficientCards
	}
	e.dealCombinedDeck(deck)
	return nil
}

// dealCombinedDeck distributes deck round-robin across the four seats as
// face-down, untokened cards. Round-robin gives conservation and at-most-one
// size difference. Caller guarantees len(deck) >= seatCount.
func (e *Engine) dealCombinedDeck(deck []Card) {
	// Reset so a re-deal starts from clean hands.
	e.state.Cards = make(map[CardID]*PlacedCard, len(deck))
	for i := range e.state.HandOrder {
		e.state.HandOrder[i] = nil
	}

	for i, card := range deck {
		seat := SeatID(i % seatCount)
		e.state.Cards[card.ID] = &PlacedCard{
			Card:   card,
			Owner:  seat,
			Facing: FaceDown,
			Token:  nil,
		}
		e.state.HandOrder[seat] = append(e.state.HandOrder[seat], card.ID)
	}
}
