package engine

// applyGuess resolves a normal guess.
func (e *Engine) applyGuess(seat SeatID, action Action) (Result, error) {
	s := e.state

	if s.Phase != PhasePlaying {
		return ResultNone, ErrWrongPhase
	}
	if seat != s.CurrentSeat {
		return ResultNone, ErrNotYourTurn
	}
	if e.rejectIfIneligible(seat) {
		return ResultNone, ErrSeatIneligible
	}

	// Source: owned, face-down, non-gold.
	source, ok := s.Cards[action.SourceCard]
	if !ok || source.Owner != seat || source.Facing != FaceDown {
		return ResultNone, ErrInvalidSource
	}
	if source.Card.Kind == SpecialKindGold {
		return ResultNone, ErrGoldSource
	}

	// Target: exists, face-down, owned by another player.
	target, ok := s.Cards[action.TargetCard]
	if !ok {
		return ResultNone, ErrInvalidTarget
	}
	if target.Facing != FaceDown {
		return ResultNone, ErrTargetRevealed
	}
	if target.Owner == seat {
		return ResultNone, ErrTargetOwn
	}
	// Another player's gold is non-targetable (own-target already rejected above).
	if target.Card.Kind == SpecialKindGold {
		return ResultNone, ErrGoldNotTargetable
	}

	if matchesAnnouncedValue(source.Card, target.Card) {
		// Hit: Pair_Reveal flips both face-up.
		source.Facing = FaceUp
		target.Facing = FaceUp
	} else {
		// Miss: lose one life (clamped), token the target face-down.
		if s.LifePool > 0 {
			s.LifePool--
		}
		target.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
	}

	return e.finishTurn(s.CurrentSeat), nil
}

// matchesAnnouncedValue reports whether target matches the value announced by
// source (its Kind+Rank). Shared by the guess, silver-feat, and safe-call paths.
func matchesAnnouncedValue(source, target Card) bool {
	return cardHasValue(target, source.Kind, source.Rank)
}
