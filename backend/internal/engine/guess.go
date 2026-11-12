package engine

// applyGuess resolves a normal guess. Reads SourceCards (exactly one, the
// caller's card announcing the value) and TargetCards (exactly one).
func (e *Engine) applyGuess(seat SeatID, action Action) (Result, error) {
	if err := action.checkLens(exactly(1), exactly(1), anyLen); err != nil {
		return ResultNone, err
	}
	s := e.state
	targetID := action.TargetCards[0]

	if s.Phase != PhasePlaying {
		return ResultNone, ErrWrongPhase
	}
	if seat != s.CurrentSeat {
		return ResultNone, ErrNotYourTurn
	}

	// Source: owned, face-down, non-gold.
	source, ok := s.Cards[action.SourceCards[0]]
	if !ok || source.Owner != seat || source.Facing != FaceDown {
		return ResultNone, ErrInvalidSource
	}
	if source.Card.Kind == SpecialKindGold {
		return ResultNone, ErrGoldSource
	}

	// Target: exists, face-down, owned by another player.
	target, ok := s.Cards[targetID]
	if !ok {
		return ResultNone, ErrInvalidTarget
	}
	if target.Facing != FaceDown {
		return ResultNone, ErrTargetRevealed
	}
	if target.Owner == seat {
		return ResultNone, ErrTargetOwn
	}

	sourceID := source.Card.ID
	la := &LastAction{
		Actor:          seat,
		Kind:           ActionKindGuess,
		AnnouncedValue: announcedValueOf(source.Card),
		Targets:        []CardID{targetID},
		SourceCard:     &sourceID,
	}

	// Targeting another player's gold ends the game immediately: the gold card
	// flips face-up and Life_Pool is untouched.
	// An armed Shield covers this guess only: a gold target gets a token with
	// no flip and no game over, a miss costs no life but still places the token.
	shield := s.Shield
	s.Shield = false

	if target.Card.Kind == SpecialKindGold && shield {
		s.placeToken(targetID)
		la.Outcome = OutcomeMiss
		s.recordAction(la)
		s.logActionOn(seat, target.Owner, "called "+announcedText(la.AnnouncedValue)+" and the shield protected a gold card")
		return e.finishTurn(s.CurrentSeat), nil
	}

	if target.Card.Kind == SpecialKindGold {
		s.flipUp(targetID)
		la.Outcome = OutcomeGoldHit
		la.Revealed = []CardID{targetID}
		s.recordAction(la)
		s.logActionOn(seat, target.Owner, "called "+announcedText(la.AnnouncedValue)+" and hit a gold card")
		return e.endGoldHit(), nil
	}

	if matchesAnnouncedValue(source.Card, target.Card) {
		// Hit: Pair_Reveal flips both face-up.
		s.flipUp(sourceID, targetID)
		la.Outcome = OutcomeHit
		la.Revealed = []CardID{sourceID, targetID}
	} else {
		// Miss: lose one life (clamped, none under the Shield), token the
		// target face-down.
		if s.LifePool > 0 && !shield {
			s.LifePool--
			la.LivesLost = 1
		}
		s.placeToken(targetID)
		la.Outcome = OutcomeMiss
	}

	s.recordAction(la)
	if la.Outcome == OutcomeMiss {
		s.logActionOn(seat, target.Owner, "called "+announcedText(la.AnnouncedValue)+" and missed")
	}

	return e.finishTurn(s.CurrentSeat), nil
}

// matchesAnnouncedValue reports whether target matches the value announced by
// source (its Kind+Rank). Shared by the guess, silver-feat, and safe-call paths.
func matchesAnnouncedValue(source, target Card) bool {
	return cardHasValue(target, source.Kind, source.Rank)
}
