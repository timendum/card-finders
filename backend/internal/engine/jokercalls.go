package engine

import (
	"slices"
)

// jokerSource validates the owned, face-down, non-gold source card shared by
// the call jokers (the same rule as a normal guess).
func (s *GameState) jokerSource(seat SeatID, id CardID) (*PlacedCard, error) {
	source, ok := s.ownFaceDown(seat, id)
	if !ok {
		return nil, ErrInvalidSource
	}
	if source.Card.Kind == SpecialKindGold {
		return nil, ErrGoldSource
	}
	return source, nil
}

// jokerTripleCall (4) is a Safe_Call on three cards of one other seat. It is
// independent of the player's own Safe_Call power.
func (e *Engine) jokerTripleCall(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	if err := s.requireCurrentSeat(seat); err != nil {
		return ResultNone, err
	}
	source, err := s.jokerSource(seat, a.SourceCards[0])
	if err != nil {
		return ResultNone, err
	}
	targets := slices.Clone(a.TargetCards)
	first, ok := s.Cards[targets[0]]
	if !ok || first.Owner == seat {
		return ResultNone, ErrJokerInvalid
	}
	for i, id := range targets {
		pc, ok := s.Cards[id]
		if !ok || pc.Owner != first.Owner || pc.Facing != FaceDown {
			return ResultNone, ErrJokerInvalid
		}
		if slices.Contains(targets[:i], id) {
			return ResultNone, ErrJokerInvalid
		}
	}

	slot.Used = true
	return e.resolveMultiTarget(multiCall{
		seat:         seat,
		source:       source,
		targetPlayer: first.Owner,
		targets:      targets,
		joker:        a.Joker,
		passTurn:     true,
	})
}

// jokerSweep (6) calls the rank of a suited source against every face-down card
// of one other seat.
func (e *Engine) jokerSweep(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	if err := s.requireCurrentSeat(seat); err != nil {
		return ResultNone, err
	}
	source, err := s.jokerSource(seat, a.SourceCards[0])
	if err != nil {
		return ResultNone, err
	}
	targetSeat := a.TargetSeats[0]
	if source.Card.Kind != SpecialKindNone || !s.validOtherSeat(seat, targetSeat) {
		return ResultNone, ErrJokerInvalid
	}
	var targets []CardID
	for _, id := range s.handBySlot(targetSeat) {
		if s.Cards[id].Facing == FaceDown {
			targets = append(targets, id)
		}
	}

	slot.Used = true
	return e.resolveMultiTarget(multiCall{
		seat:         seat,
		source:       source,
		targetPlayer: targetSeat,
		targets:      targets,
		joker:        a.Joker,
		passTurn:     true,
	})
}

// jokerReverseCall (11) plays two own cards of different identity against one
// card of another seat: a hit is a match with either own card (that card and
// the target flip). A miss costs a life and tokens the target; a gold target
// is a loss.
func (e *Engine) jokerReverseCall(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	if err := s.requireCurrentSeat(seat); err != nil {
		return ResultNone, err
	}
	srcID1, srcID2, targetID := a.SourceCards[0], a.SourceCards[1], a.TargetCards[0]
	src1, err := s.jokerSource(seat, srcID1)
	if err != nil {
		return ResultNone, err
	}
	src2, err := s.jokerSource(seat, srcID2)
	if err != nil {
		return ResultNone, err
	}
	if srcID1 == srcID2 || sameIdentity(src1.Card, src2.Card) {
		return ResultNone, ErrJokerInvalid
	}
	target, ok := s.Cards[targetID]
	switch {
	case !ok:
		return ResultNone, ErrInvalidTarget
	case target.Facing != FaceDown:
		return ResultNone, ErrTargetRevealed
	case target.Owner == seat:
		return ResultNone, ErrTargetOwn
	}

	slot.Used = true
	la := jokerAction(seat, a.Joker)
	la.Targets = []CardID{targetID}
	logCall := func(outcome LogOutcome) {
		s.logEntry(LogEntry{
			Actor:   new(seat),
			Target:  new(target.Owner),
			Event:   LogEventCall,
			Joker:   a.Joker,
			Sources: s.slotsOf(srcID1, srcID2),
			Slots:   s.slotsOf(targetID),
			Outcome: outcome,
		})
	}

	var hit *PlacedCard
	for _, src := range []*PlacedCard{src1, src2} {
		if matchesAnnouncedValue(src.Card, target.Card) {
			hit = src
		}
	}

	switch {
	case target.Card.Kind == SpecialKindGold:
		s.flipUp(targetID)
		la.Outcome = OutcomeGoldHit
		la.Revealed = []CardID{targetID}
		s.recordAction(la)
		logCall(LogOutcomeHitGold)
		return e.endGoldHit(), nil
	case hit != nil:
		id := hit.Card.ID
		s.flipUp(id, targetID)
		la.Outcome = OutcomeHit
		la.AnnouncedValue = announcedValueOf(hit.Card)
		la.SourceCard = &id
		la.Revealed = []CardID{id, targetID}
		s.recordAction(la)
		logCall(LogOutcomeHit)
	default:
		if s.LifePool > 0 {
			s.LifePool--
			la.LivesLost = 1
		}
		s.placeToken(targetID)
		la.Outcome = OutcomeMiss
		s.recordAction(la)
		logCall(LogOutcomeMissed)
	}
	return e.finishTurn(s.CurrentSeat), nil
}

// jokerSwap (3) starts an exchange: the activator offers an own face-down card
// to a seat, whose player then picks the card it gives back (pending choice).
func (e *Engine) jokerSwap(seat SeatID, a Action, slot *JokerSlot) (Result, error) {
	s := e.state
	sourceID, targetSeat := a.SourceCards[0], a.TargetSeats[0]
	if _, ok := s.ownFaceDown(seat, sourceID); !ok || !s.validOtherSeat(seat, targetSeat) {
		return ResultNone, ErrJokerInvalid
	}

	slot.Used = true
	s.PendingSafeChoice = &PendingSafeChoice{
		Caller:       seat,
		TargetPlayer: targetSeat,
		SourceCard:   sourceID,
		Joker:        a.Joker,
	}
	s.Phase = PhaseAwaitingResponse
	la := jokerAction(seat, a.Joker)
	la.Outcome = OutcomeAwaitingChoice // empty Targets: pick any own face-down card
	la.SourceCard = &sourceID
	s.recordAction(la)
	s.logJoker(seat, &targetSeat, LogEntry{Joker: a.Joker, Slots: s.slotsOf(sourceID)})
	return ResultNone, nil
}

// resolveSwap completes a Swap: the target player names one of its own
// face-down cards, the two cards change owner and each lands in the receiving
// hand per swapCards. The turn is untouched.
func (e *Engine) resolveSwap(seat SeatID, a Action, p *PendingSafeChoice) (Result, error) {
	s := e.state
	chosen := a.TargetCards[0]
	if _, ok := s.ownFaceDown(seat, chosen); !ok {
		return ResultNone, ErrInvalidChoice
	}
	given, taken := p.SourceCard, chosen
	from, to := s.Slot[given], s.Slot[taken]

	s.PendingSafeChoice = nil
	s.Phase = PhasePlaying
	s.swapCards(given, taken)

	la := jokerAction(p.Caller, p.Joker)
	la.Kind = ActionKindSafeChooseReveal
	la.Targets = []CardID{given, taken}
	s.recordAction(la)
	s.logJoker(p.Caller, &seat, LogEntry{
		Joker:   p.Joker,
		Slots:   []int{from, to},
		Sources: []int{s.Slot[taken], s.Slot[given]},
	})
	return ResultNone, nil
}
