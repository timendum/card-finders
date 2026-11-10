package engine

// applySafeCall resolves a Safe_Call: a one-time power used in place of a guess,
// targeting two distinct face-down cards of a single other player. The power is
// spent on every valid call. On a double hit resolution defers to
// applySafeChooseReveal, so PowerUsed is set but the Pair_Reveal and turn pass
// wait for the targeted player's choice.
func (e *Engine) applySafeCall(seat SeatID, action Action) (Result, error) {
	s := e.state

	if s.Phase != PhasePlaying {
		return ResultNone, ErrWrongPhase
	}
	if seat != s.CurrentSeat {
		return ResultNone, ErrNotYourTurn
	}

	// Players[seat] is non-nil during play.
	if p := s.Players[seat]; p == nil || p.PowerUsed {
		return ResultNone, ErrSafeCallSpent
	}

	// Source: owned, face-down, non-gold.
	source, ok := s.Cards[action.SourceCard]
	if !ok || source.Owner != seat || source.Facing != FaceDown {
		return ResultNone, ErrInvalidSource
	}
	if source.Card.Kind == SpecialKindGold {
		return ResultNone, ErrGoldSource
	}

	// Targets: both exist, distinct, face-down, owned by the same other player
	// who holds at least two face-down cards. Any failure is ErrSafeCallTargets.
	t1, ok1 := s.Cards[action.TargetCard]
	t2, ok2 := s.Cards[action.TargetCard2]
	if !ok1 || !ok2 {
		return ResultNone, ErrSafeCallTargets
	}
	if action.TargetCard == action.TargetCard2 {
		return ResultNone, ErrSafeCallTargets
	}
	if t1.Facing != FaceDown || t2.Facing != FaceDown {
		return ResultNone, ErrSafeCallTargets
	}
	targetPlayer := t1.Owner
	if targetPlayer == seat || t2.Owner != targetPlayer {
		return ResultNone, ErrSafeCallTargets
	}
	if faceDownCount(s, targetPlayer) < 2 {
		return ResultNone, ErrSafeCallTargets
	}
	// Another player's gold is non-targetable; reject before spending the power.
	if t1.Card.Kind == SpecialKindGold || t2.Card.Kind == SpecialKindGold {
		return ResultNone, ErrGoldNotTargetable
	}

	match1 := matchesAnnouncedValue(source.Card, t1.Card)
	match2 := matchesAnnouncedValue(source.Card, t2.Card)

	s.Players[seat].PowerUsed = true

	sourceID := source.Card.ID
	la := &LastAction{
		Actor:          seat,
		Kind:           ActionKindSafeCall,
		AnnouncedValue: announcedValueOf(source.Card),
		Targets:        []CardID{action.TargetCard, action.TargetCard2},
		SourceCard:     &sourceID,
	}

	switch {
	case match1 && match2:
		// Double hit: record the pending choice and defer the Pair_Reveal and
		// turn pass to applySafeChooseReveal. The source is still face-down, so
		// projection gates SourceCard; only the two options are announced.
		s.PendingSafeChoice = &PendingSafeChoice{
			Caller:       seat,
			TargetPlayer: targetPlayer,
			SourceCard:   source.Card.ID,
			Option1:      action.TargetCard,
			Option2:      action.TargetCard2,
		}
		// CurrentSeat stays the caller; the chooser is the pending TargetPlayer.
		s.Phase = PhaseAwaitingResponse
		la.Outcome = OutcomeAwaitingChoice
		s.recordAction(la)
		s.logAction(seat, "used safe call: both hit")
		return ResultNone, nil

	case match1 || match2:
		// One hit: Pair_Reveal flips the source and the lone matching target.
		matched := t1
		if match2 {
			matched = t2
		}
		source.Facing = FaceUp
		matched.Facing = FaceUp
		la.Outcome = OutcomeHit
		la.Revealed = []CardID{sourceID, matched.Card.ID}

	default:
		// Miss: lose one life (clamped), token each target face-down, source untouched.
		if s.LifePool > 0 {
			s.LifePool--
			la.LivesLost = 1
		}
		// TODO: allow player to choose?
		t1.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
		t2.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
		la.Outcome = OutcomeMiss
	}

	s.recordAction(la)
	if la.Outcome == OutcomeHit {
		s.logAction(seat, "used safe call: one hit")
	} else {
		s.logAction(seat, "used safe call: miss")
	}

	return e.finishTurn(s.CurrentSeat), nil
}

// faceDownCount returns how many of a seat's cards are still face-down.
func faceDownCount(s *GameState, seat SeatID) int {
	n := 0
	for _, id := range s.HandOrder[seat] {
		if c, ok := s.Cards[id]; ok && c.Facing == FaceDown {
			n++
		}
	}
	return n
}

// applySafeChooseReveal resolves the second step of a double-hit Safe_Call: the
// targeted player picks which matching card is revealed, completing the deferred
// Pair_Reveal and passing the turn from the caller. The power was already spent
// by applySafeCall.
func (e *Engine) applySafeChooseReveal(seat SeatID, action Action) (Result, error) {
	s := e.state

	if s.Phase != PhaseAwaitingResponse {
		return ResultNone, ErrWrongPhase
	}
	pending := s.PendingSafeChoice
	if pending == nil {
		return ResultNone, ErrNoPendingChoice
	}
	if seat != pending.TargetPlayer {
		return ResultNone, ErrNotYourTurn
	}
	if action.ChosenTarget != pending.Option1 && action.ChosenTarget != pending.Option2 {
		return ResultNone, ErrInvalidChoice
	}

	sourceID := pending.SourceCard
	var av *CardValue
	if src, ok := s.Cards[pending.SourceCard]; ok {
		src.Facing = FaceUp
		av = announcedValueOf(src.Card)
	}
	if chosen, ok := s.Cards[action.ChosenTarget]; ok {
		chosen.Facing = FaceUp
	}

	caller := pending.Caller
	srcID := sourceID
	s.recordAction(&LastAction{
		Actor:          caller,
		Kind:           ActionKindSafeChooseReveal,
		AnnouncedValue: av,
		Targets:        []CardID{action.ChosenTarget},
		Outcome:        OutcomeHit,
		Revealed:       []CardID{srcID, action.ChosenTarget},
		SourceCard:     &srcID,
	})

	s.PendingSafeChoice = nil
	// Leave the response phase first; finishTurn may then move on to Ended.
	s.Phase = PhasePlaying
	return e.finishTurn(caller), nil
}
