package engine

import "sort"

// handBySlot returns a seat's card ids in ascending Slot order (full hand,
// face-up cards included).
func (s *GameState) handBySlot(seat SeatID) []CardID {
	ids := append([]CardID(nil), s.HandOrder[seat]...)
	sort.Slice(ids, func(i, j int) bool { return s.Slot[ids[i]] < s.Slot[ids[j]] })
	return ids
}

// linkPairs lists the link-token pairs of the board. Within one hand,
// consecutive slots carrying a face-down link token of the same kind pair
// left to right. Runs always consist of whole pairs, because a link is cleared
// whenever one of its cards flips, moves or loses its token.
func (s *GameState) linkPairs() [][2]CardID {
	var pairs [][2]CardID
	for seat := range SeatID(seatCount) {
		var run []CardID
		var kind TokenKind
		flush := func() {
			for i := 0; i+1 < len(run); i += 2 {
				pairs = append(pairs, [2]CardID{run[i], run[i+1]})
			}
			run = nil
		}
		for _, id := range s.handBySlot(seat) {
			pc := s.Cards[id]
			if pc.Facing != FaceDown || !pc.Token.isLink() {
				flush()
				continue
			}
			if len(run) > 0 && pc.Token.Displayed != kind {
				flush()
			}
			kind = pc.Token.Displayed
			run = append(run, id)
		}
		flush()
	}
	return pairs
}

// unlinkCard clears the link token of id and of its peer, if any.
func (s *GameState) unlinkCard(id CardID) {
	if !s.Cards[id].Token.isLink() {
		return
	}
	for _, p := range s.linkPairs() {
		if p[0] == id || p[1] == id {
			s.Cards[p[0]].Token = nil
			s.Cards[p[1]].Token = nil
			return
		}
	}
	s.Cards[id].Token = nil
}

// flipUp turns cards face-up, first clearing any link they belong to. Every
// reveal goes through here so a link never outlives one of its cards.
func (s *GameState) flipUp(ids ...CardID) {
	for _, id := range ids {
		s.unlinkCard(id)
		s.Cards[id].Facing = FaceUp
	}
}

// placeToken puts a natural-identity token on a face-down card, replacing (and
// so dissolving) any link token it carried.
func (s *GameState) placeToken(id CardID) {
	s.unlinkCard(id)
	s.Cards[id].Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
}

// sameIdentity is the link-token notion of "same": suited cards compare by
// rank, specials by kind only (a silver 6 equals a silver 7, never a gold or a
// suited 7).
func sameIdentity(a, b Card) bool {
	if a.Kind != b.Kind {
		return false
	}
	return a.Kind != SpecialKindNone || a.Rank == b.Rank
}

// insertIndex returns where a card of sort value sv goes in the cards ordered
// by slot: after every card of lower or equal Sort_Value.
func (s *GameState) insertIndex(ids []CardID, sv float64) int {
	idx := 0
	for i, id := range ids {
		if s.Cards[id].Card.SortValue() <= sv {
			idx = i + 1
		}
	}
	return idx
}

// swapCards exchanges cards a (owned by seatA) and b (owned by seatB). Neither
// hand is re-sorted: each loses its given card and inserts the received one in
// the last valid spot (see insertIndex). Slots are renumbered, link tokens of
// the moved cards and of any pair split by the insertion are cleared, and
// natural-identity tokens follow their card.
func (s *GameState) swapCards(a, b CardID) {
	s.unlinkCard(a)
	s.unlinkCard(b)
	before := s.linkPairs()

	seatA, seatB := s.Cards[a].Owner, s.Cards[b].Owner
	handA := removeID(s.handBySlot(seatA), a)
	handB := removeID(s.handBySlot(seatB), b)
	s.Cards[a].Owner, s.Cards[b].Owner = seatB, seatA

	handA = insertID(handA, s.insertIndex(handA, s.Cards[b].Card.SortValue()), b)
	handB = insertID(handB, s.insertIndex(handB, s.Cards[a].Card.SortValue()), a)
	s.setHand(seatA, handA)
	s.setHand(seatB, handB)

	for _, p := range before {
		if s.Slot[p[0]]-s.Slot[p[1]] != 1 && s.Slot[p[1]]-s.Slot[p[0]] != 1 {
			s.Cards[p[0]].Token = nil
			s.Cards[p[1]].Token = nil
		}
	}
}

// setHand stores a seat's hand in slot order and renumbers Slot 1..n.
func (s *GameState) setHand(seat SeatID, ids []CardID) {
	s.HandOrder[seat] = ids
	for i, id := range ids {
		s.Slot[id] = i + 1
	}
}

func removeID(ids []CardID, id CardID) []CardID {
	out := make([]CardID, 0, len(ids))
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

func insertID(ids []CardID, at int, id CardID) []CardID {
	out := make([]CardID, 0, len(ids)+1)
	out = append(out, ids[:at]...)
	out = append(out, id)
	return append(out, ids[at:]...)
}
