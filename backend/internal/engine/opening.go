package engine

// applyOpeningReveal reveals the acting seat's single opening card: it tokens
// the card with its natural identity while keeping it face-down, then advances
// the turn. Validation runs before any mutation, so a rejected reveal is a
// no-op.
func (e *Engine) applyOpeningReveal(seat SeatID, action Action) (Result, error) {
	s := e.state

	if s.Phase != PhaseOpening {
		return ResultNone, ErrWrongPhase
	}
	if seat != s.CurrentSeat {
		return ResultNone, ErrNotYourTurn
	}

	pc, ok := s.Cards[action.OpeningCard]
	if !ok || pc.Owner != seat || pc.Facing != FaceDown {
		return ResultNone, ErrInvalidOpeningSelection
	}

	if pc.Card.Kind != SpecialKindNone {
		return ResultNone, ErrSpecialOpening
	}

	pc.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
	s.OpeningDone[seat] = true

	s.recordAction(&LastAction{
		Actor:   seat,
		Kind:    ActionKindOpeningReveal,
		Targets: []CardID{action.OpeningCard},
		Outcome: OutcomeNone,
	})

	s.CurrentSeat = nextSeat(s.CurrentSeat)

	// After the fourth reveal CurrentSeat has wrapped back to the first opener,
	// so normal play resumes there with no extra bookkeeping.
	if e.allOpeningDone() {
		s.Phase = PhasePlaying
	}

	return ResultNone, nil
}

// allOpeningDone reports whether every seat has completed its opening reveal.
func (e *Engine) allOpeningDone() bool {
	for _, done := range e.state.OpeningDone {
		if !done {
			return false
		}
	}
	return true
}
