package engine

import "strconv"

// applyJoker activates a joker. A joker is usable only while the game waits for
// a user action: Playing, never with a choice pending. "Anytime" jokers (2, 3,
// 5, 7, 8, 9, 13) leave the turn alone; "current player" jokers (4, 6, 10, 11,
// 12) need the actor to be the current seat. Validation always precedes
// mutation, so a rejected activation is a no-op.
func (e *Engine) applyJoker(seat SeatID, action Action) (Result, error) {
	s := e.state
	if !s.jokerUsable(action.Joker) {
		return ResultNone, ErrJokerUnavailable
	}
	slot := s.jokerSlot(action.Joker)

	switch action.Joker {
	case JokerDiffer, JokerMatch:
		return e.jokerLink(seat, action, slot)
	case JokerSwap:
		return e.jokerSwap(seat, action, slot)
	case JokerTripleCall:
		return e.jokerTripleCall(seat, action, slot)
	case JokerSelfMark:
		return e.jokerSelfMark(seat, action, slot)
	case JokerSweep:
		return e.jokerSweep(seat, action, slot)
	case JokerExtraLife:
		return e.jokerExtraLife(seat, slot)
	case JokerSecondWind:
		return e.jokerSecondWind(seat, action, slot)
	case JokerCensus:
		return e.jokerCensus(seat, action, slot)
	case JokerShield:
		return e.jokerShield(seat, slot)
	case JokerReverseCall:
		return e.jokerReverseCall(seat, action, slot)
	case JokerPass:
		return e.jokerPass(seat, action, slot)
	default:
		return ResultNone, ErrJokerUnavailable
	}
}

// jokerAction starts the LastAction record of a joker use.
func jokerAction(seat SeatID, rank Rank) *LastAction {
	return &LastAction{Actor: seat, Kind: ActionKindJoker, Joker: rank}
}

// jokerText prefixes a log line with the joker's numeric rank.
func jokerText(rank Rank, detail string) string {
	return " used joker " + announcedText(&CardValue{Rank: rank}) + ": " + detail
}

// ownFaceDown returns seat's own face-down card id, or false.
func (s *GameState) ownFaceDown(seat SeatID, id CardID) (*PlacedCard, bool) {
	pc, ok := s.Cards[id]
	if !ok || pc.Owner != seat || pc.Facing != FaceDown {
		return nil, false
	}
	return pc, true
}

// validOtherSeat reports whether target is a seat other than seat that still
// holds a face-down card.
func (s *GameState) validOtherSeat(seat, target SeatID) bool {
	return target < seatCount && target != seat && faceDownCount(s, target) > 0
}

// requireCurrentSeat is the gate of the "current player" jokers: the actor
// must be the current seat and still own a face-down card.
func (s *GameState) requireCurrentSeat(seat SeatID) error {
	if seat != s.CurrentSeat {
		return ErrNotYourTurn
	}
	if faceDownCount(s, seat) == 0 {
		return ErrJokerInvalid
	}
	return nil
}

// jokerLink is Differ (2) and Match (13): two adjacent own face-down cards
// without a token get Same/Different link tokens. The claim must be true; the
// activator knows his own cards, so a false claim is rejected, not punished.
func (e *Engine) jokerLink(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	c1, ok1 := s.ownFaceDown(seat, a.SourceCard)
	c2, ok2 := s.ownFaceDown(seat, a.TargetCard)
	if !ok1 || !ok2 || a.SourceCard == a.TargetCard || c1.Token != nil || c2.Token != nil {
		return ResultNone, ErrJokerInvalid
	}
	if d := s.Slot[a.SourceCard] - s.Slot[a.TargetCard]; d != 1 && d != -1 {
		return ResultNone, ErrJokerInvalid
	}
	kind, word := TokenKindDifferent, "Different"
	if a.Joker == JokerMatch {
		kind, word = TokenKindSame, "Same"
	}
	if sameIdentity(c1.Card, c2.Card) != (kind == TokenKindSame) {
		return ResultNone, ErrJokerInvalid
	}

	slot.Used = true
	c1.Token = &TokenInfo{Displayed: kind}
	c2.Token = &TokenInfo{Displayed: kind}
	la := jokerAction(seat, a.Joker)
	la.Targets = []CardID{a.SourceCard, a.TargetCard}
	s.recordAction(la)
	s.logAction(seat, jokerText(a.Joker, "put a "+word+" token on cards "+
		s.slotsText(la.Targets)))
	return ResultNone, nil
}

// jokerSelfMark (5) puts a natural-identity token on an own face-down suited
// card without a token.
func (e *Engine) jokerSelfMark(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	pc, ok := s.ownFaceDown(seat, a.SourceCard)
	if !ok || pc.Card.Kind != SpecialKindNone || pc.Token != nil {
		return ResultNone, ErrJokerInvalid
	}
	slot.Used = true
	pc.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
	la := jokerAction(seat, a.Joker)
	la.Targets = []CardID{a.SourceCard}
	s.recordAction(la)
	s.logAction(seat, jokerText(a.Joker, "put a token on card "+strconv.Itoa(s.Slot[a.SourceCard])))
	return ResultNone, nil
}

// jokerExtraLife (7) adds one life with no cap; MaxLife is unchanged.
func (e *Engine) jokerExtraLife(seat SeatID, slot *JokerSlot) (Result, error) {
	s := e.state
	slot.Used = true
	s.LifePool++
	s.recordAction(jokerAction(seat, JokerExtraLife))
	s.logAction(seat, jokerText(JokerExtraLife, "gained a life"))
	return ResultNone, nil
}

// jokerSecondWind (8) restores spent Safe_Call powers: automatically when at
// most two seats spent theirs, else the two seats the activator names. With
// none spent the joker is wasted.
func (e *Engine) jokerSecondWind(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	var spent []SeatID
	for i, p := range s.Players {
		if p != nil && p.PowerUsed {
			spent = append(spent, SeatID(i))
		}
	}
	restore := spent
	if len(spent) > 2 {
		t1, t2 := a.TargetSeat, a.TargetSeat2
		if t1 == t2 || t1 >= seatCount || t2 >= seatCount ||
			!s.Players[t1].PowerUsed || !s.Players[t2].PowerUsed {
			return ResultNone, ErrJokerInvalid
		}
		restore = []SeatID{t1, t2}
	}

	slot.Used = true
	for _, r := range restore {
		s.Players[r].PowerUsed = false
	}
	s.recordAction(jokerAction(seat, a.Joker))
	if len(restore) == 0 {
		s.logAction(seat, jokerText(a.Joker, "nobody had spent a Double Call"))
	}
	for _, r := range restore {
		s.logActionOn(seat, r, jokerText(a.Joker, "restored the Double Call"))
	}
	return ResultNone, nil
}

// jokerCensus (9) asks every seat, the asker included, whether it holds a
// face-down suited card of one rank. Answers are public, one log entry each.
func (e *Engine) jokerCensus(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	rank := a.SelfRevealValue.Rank
	if a.SelfRevealValue.Kind != SpecialKindNone || rank < minRank || rank > maxRank {
		return ResultNone, ErrJokerInvalid
	}

	slot.Used = true
	asked := &CardValue{Rank: rank}
	la := jokerAction(seat, a.Joker)
	la.AnnouncedValue = asked
	s.recordAction(la)
	for i := range SeatID(seatCount) {
		answer := "no"
		for _, id := range s.HandOrder[i] {
			pc := s.Cards[id]
			if pc.Facing == FaceDown && cardHasValue(pc.Card, SpecialKindNone, rank) {
				answer = "yes"
				break
			}
		}
		s.logAction(i, jokerText(a.Joker, "answered "+answer+" to a face-down "+announcedText(asked)))
	}
	return ResultNone, nil
}

// jokerShield (10) arms the Shield for the current player's next normal guess
// or silver feat. It does not end the turn.
func (e *Engine) jokerShield(seat SeatID, slot *JokerSlot) (Result, error) {
	s := e.state
	if err := s.requireCurrentSeat(seat); err != nil {
		return ResultNone, err
	}
	slot.Used = true
	s.Shield = true
	s.recordAction(jokerAction(seat, JokerShield))
	s.logAction(seat, jokerText(JokerShield, "armed the shield"))
	return ResultNone, nil
}

// jokerPass (12) skips the turn: the activator picks the next current seat.
func (e *Engine) jokerPass(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	if err := s.requireCurrentSeat(seat); err != nil {
		return ResultNone, err
	}
	if !s.validOtherSeat(seat, a.TargetSeat) {
		return ResultNone, ErrJokerInvalid
	}
	slot.Used = true
	s.CurrentSeat = a.TargetSeat
	s.recordAction(jokerAction(seat, a.Joker))
	s.logActionOn(seat, a.TargetSeat, jokerText(a.Joker, "passed the turn"))
	return ResultNone, nil
}
