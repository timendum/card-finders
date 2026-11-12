package engine

import (
	"slices"
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
	if s.Shield {
		return ResultNone, ErrShieldActive
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

	s.Players[seat].PowerUsed = true

	targets := []CardID{action.TargetCard, action.TargetCard2}
	return e.resolveMultiTarget(multiCall{
		seat:         seat,
		source:       source,
		targetPlayer: targetPlayer,
		targets:      targets,
		passTurn:     true,
		text: func(outcome string) string {
			return "asked for a " + announcedText(announcedValueOf(source.Card)) +
				" at cards " + s.slotsText(targets) + " and " + outcome
		},
	})
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

// applySafeChooseReveal resolves the second step of a pending choice: the
// targeted player picks which matching card is revealed (or which card gets the
// token on a miss), or, for a Swap, which of its own cards it gives. A plain
// Safe_Call was already marked spent by applySafeCall.
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
	if pending.Joker == JokerSwap {
		return e.resolveSwap(seat, action, pending)
	}
	if !slices.Contains(pending.Options, action.ChosenTarget) {
		return ResultNone, ErrInvalidChoice
	}

	sourceID := pending.SourceCard
	caller := pending.Caller
	la := &LastAction{
		Actor:      caller,
		Kind:       ActionKindSafeChooseReveal,
		Joker:      pending.Joker,
		Targets:    []CardID{action.ChosenTarget},
		SourceCard: &sourceID,
	}
	if src, ok := s.Cards[sourceID]; ok {
		la.AnnouncedValue = announcedValueOf(src.Card)
	}

	// Leave the response phase first; finishing may then move on to Ended.
	s.PendingSafeChoice = nil
	s.Phase = PhasePlaying

	if pending.Miss {
		// Miss: token only the chosen target; nothing is revealed.
		s.placeToken(action.ChosenTarget)
		la.Outcome = OutcomeMiss
		s.recordAction(la)
		s.logAction(seat, "chose card "+strconv.Itoa(s.Slot[action.ChosenTarget])+" to receive the token")
		return e.endCall(caller, pending.PassTurn), nil
	}

	s.flipUp(sourceID, action.ChosenTarget)
	la.Outcome = OutcomeHit
	la.Revealed = []CardID{sourceID, action.ChosenTarget}
	s.recordAction(la)
	s.logAction(seat, "chose card "+strconv.Itoa(s.Slot[action.ChosenTarget])+" to reveal")
	return e.endCall(caller, pending.PassTurn), nil
}
