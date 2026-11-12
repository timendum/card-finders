package engine

import (
	"log/slog"
	"sort"
)

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

// applyIDPermutation breaks the id-to-value correlation so an opponent's
// face-down card leaks nothing through its id.
func (e *Engine) applyIDPermutation(deck []Card) {
	perm := e.rng.Perm(len(deck))
	// A permutation is a bijection, so ids stay unique.
	remap := make(map[CardID]CardID, len(deck))
	for i := range deck {
		remap[deck[i].ID] = CardID(perm[i])
	}
	for i := range deck {
		deck[i].ID = remap[deck[i].ID]
	}
	for i := range e.state.SpecialPool.Silvers {
		e.state.SpecialPool.Silvers[i].ID = remap[e.state.SpecialPool.Silvers[i].ID]
	}
	for i := range e.state.SpecialPool.Golds {
		e.state.SpecialPool.Golds[i].ID = remap[e.state.SpecialPool.Golds[i].ID]
	}
}

// startGame runs the fourth-join transition: form pool, deal, pick first player,
// reach PhaseOpening. On any failure it reverts fully to PhaseWaiting with no
// dealt hands, so the triggering Join surfaces the error cleanly.
func (e *Engine) startGame() error {

	// Step one: Build Special Pool

	// Level was validated at CreateGame, so a well-formed level does not error.
	pool, err := BuildSpecialPool(e.rng, e.state.Level)
	if err != nil {
		slog.Error("Invalid Special Pool config", "err", err)
		return err
	}
	e.state.SpecialPool = pool

	deck := e.combinedDeck()

	// Step: shuffle IDs, then Cards

	e.applyIDPermutation(deck)
	e.shuffle(deck)
	e.dealCombinedDeck(deck)

	// Step: randomly pick first player

	e.state.CurrentSeat = SeatID(e.rng.Intn(seatCount))

	// Step: draw the jokers last, so existing seeds keep the same deal

	e.state.Jokers = drawJokerRanks(e.rng, e.state.Level.Jokers)

	e.state.Phase = PhaseOpening
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
	e.state.Slot = make(map[CardID]int, len(deck))

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

	e.computeSlots()
}

// computeSlots assigns each card a 1-based position within its owning seat's
// value-sorted hand: lowest Sort_Value is slot 1, ties resolved stably over the
// shuffled deal order in HandOrder. Called once per deal; slots are never
// mutated afterward, except by a Swap joker (see swapCards).
func (e *Engine) computeSlots() {
	for seat := range e.state.HandOrder {
		ids := append([]CardID(nil), e.state.HandOrder[seat]...)
		sort.SliceStable(ids, func(i, j int) bool {
			return e.state.Cards[ids[i]].Card.SortValue() < e.state.Cards[ids[j]].Card.SortValue()
		})
		for rank, id := range ids {
			e.state.Slot[id] = rank + 1
		}
	}
}
