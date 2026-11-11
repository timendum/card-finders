package engine

import (
	"strconv"
)

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
	gold1 := t1.Card.Kind == SpecialKindGold
	gold2 := t2.Card.Kind == SpecialKindGold

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

	// Both targets gold: game over, both flip face-up, Life_Pool untouched.
	if gold1 && gold2 {
		t1.Facing = FaceUp
		t2.Facing = FaceUp
		la.Outcome = OutcomeGoldHit
		la.Revealed = []CardID{action.TargetCard, action.TargetCard2}
		s.recordAction(la)
		s.logActionOn(seat, targetPlayer, s.safeCallText(action, la.AnnouncedValue, "hit two gold cards"))
		return e.endGoldHit(), nil
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
		s.logActionOn(seat, targetPlayer, s.safeCallText(action, la.AnnouncedValue, "hit both"))
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
		// Miss: lose one life (clamped); exactly one target gets a token and the
		// source is untouched.
		if s.LifePool > 0 {
			s.LifePool--
			la.LivesLost = 1
		}
		la.Outcome = OutcomeMiss
		switch {
		case gold1:
			// Gold is never tokened: the lone non-gold target takes it.
			t2.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
		case gold2:
			t1.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
		case s.LifePool > 0:
			// Two non-gold misses: the targeted player picks which gets the
			// token. The life is already lost; the turn passes after the choice.
			s.PendingSafeChoice = &PendingSafeChoice{
				Caller:       seat,
				TargetPlayer: targetPlayer,
				SourceCard:   source.Card.ID,
				Option1:      action.TargetCard,
				Option2:      action.TargetCard2,
				Miss:         true,
			}
			s.Phase = PhaseAwaitingResponse
			la.Outcome = OutcomeAwaitingToken
			s.recordAction(la)
			s.logActionOn(seat, targetPlayer, s.safeCallText(action, la.AnnouncedValue, "missed"))
			return ResultNone, nil
		}
		// Otherwise the last life is gone and the game ends below.
	}

	s.recordAction(la)
	if la.Outcome == OutcomeHit {
		s.logActionOn(seat, targetPlayer, s.safeCallText(action, la.AnnouncedValue, "hit one"))
	} else {
		s.logActionOn(seat, targetPlayer, s.safeCallText(action, la.AnnouncedValue, "missed"))
	}

	return e.finishTurn(s.CurrentSeat), nil
}

// safeCallText renders a safe call log line: the targeted slots (ascending),
// the announced value, and the outcome phrase (e.g. "hit one").
func (s *GameState) safeCallText(action Action, announced *CardValue, outcome string) string {
	a, b := s.Slot[action.TargetCard], s.Slot[action.TargetCard2]
	if a > b {
		a, b = b, a
	}
	return "asked for a " + announcedText(announced) + " at cards " +
		strconv.Itoa(a) + " and " + strconv.Itoa(b) + " and " + outcome
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

	if pending.Miss {
		// Double miss: token only the chosen target; nothing is revealed.
		var miss *CardValue
		if src, ok := s.Cards[sourceID]; ok {
			miss = announcedValueOf(src.Card)
		}
		if chosen, ok := s.Cards[action.ChosenTarget]; ok {
			chosen.Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
		}
		caller := pending.Caller
		s.recordAction(&LastAction{
			Actor:          caller,
			Kind:           ActionKindSafeChooseReveal,
			AnnouncedValue: miss,
			Targets:        []CardID{action.ChosenTarget},
			Outcome:        OutcomeMiss,
			SourceCard:     &sourceID,
		})
		s.PendingSafeChoice = nil
		s.Phase = PhasePlaying
		s.logAction(seat, "chose card "+strconv.Itoa(s.Slot[action.ChosenTarget])+" to receive the token")
		return e.finishTurn(caller), nil
	}

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
	s.logAction(seat, "chose card "+strconv.Itoa(s.Slot[action.ChosenTarget])+" to reveal")
	return e.finishTurn(caller), nil
}
