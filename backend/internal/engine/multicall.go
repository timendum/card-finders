package engine

import "strconv"

// multiCall describes a call against several face-down cards of one other
// seat: Safe_Call (2 targets), joker 4 (3 targets) and joker 6 (the whole hand).
// Everything is already validated; resolveMultiTarget only applies the rules.
type multiCall struct {
	seat         SeatID
	source       *PlacedCard
	targetPlayer SeatID
	targets      []CardID
	joker        Rank // 0 for a plain Safe_Call
	passTurn     bool
	// text renders the log line for an outcome phrase such as "hit one".
	text func(outcome string) string
}

// resolveMultiTarget applies the Safe_Call rules to N targets:
//   - every target gold: loss, all flip face-up, no life change;
//   - one match: Pair_Reveal of the source and that target;
//   - several matches: the target player chooses (pending choice);
//   - no match: life -1, one token on a non-gold target (the target player
//     chooses when several, unless the life pool just emptied).
func (e *Engine) resolveMultiTarget(mc multiCall) (Result, error) {
	s := e.state
	sourceID := mc.source.Card.ID

	la := &LastAction{
		Actor:          mc.seat,
		Kind:           ActionKindSafeCall,
		Joker:          mc.joker,
		AnnouncedValue: announcedValueOf(mc.source.Card),
		Targets:        append([]CardID(nil), mc.targets...),
		SourceCard:     &sourceID,
	}
	if mc.joker != 0 {
		la.Kind = ActionKindJoker
	}

	var matches, nonGold []CardID
	for _, id := range mc.targets {
		c := s.Cards[id].Card
		if c.Kind != SpecialKindGold {
			nonGold = append(nonGold, id)
		}
		if matchesAnnouncedValue(mc.source.Card, c) {
			matches = append(matches, id)
		}
	}

	if len(nonGold) == 0 {
		s.flipUp(mc.targets...)
		la.Outcome = OutcomeGoldHit
		la.Revealed = append([]CardID(nil), mc.targets...)
		s.recordAction(la)
		phrase := "hit a gold card"
		if len(mc.targets) > 1 {
			phrase = "hit " + countWord(len(mc.targets)) + " gold cards"
		}
		s.logActionOn(mc.seat, mc.targetPlayer, mc.text(phrase))
		return e.endGoldHit(), nil
	}

	switch {
	case len(matches) >= 2:
		// Several hits: defer the Pair_Reveal and turn pass to the target
		// player's choice. The source is still face-down, so projection gates
		// SourceCard; the options are announced as the action's targets.
		s.PendingSafeChoice = &PendingSafeChoice{
			Caller:       mc.seat,
			TargetPlayer: mc.targetPlayer,
			SourceCard:   sourceID,
			Options:      matches,
			Joker:        mc.joker,
			PassTurn:     mc.passTurn,
		}
		s.Phase = PhaseAwaitingResponse
		la.Outcome = OutcomeAwaitingChoice
		la.Targets = append([]CardID(nil), matches...)
		s.recordAction(la)
		phrase := "hit several"
		if len(matches) == 2 && len(mc.targets) == 2 {
			phrase = "hit both"
		}
		s.logActionOn(mc.seat, mc.targetPlayer, mc.text(phrase))
		return ResultNone, nil

	case len(matches) == 1:
		s.flipUp(sourceID, matches[0])
		la.Outcome = OutcomeHit
		la.Revealed = []CardID{sourceID, matches[0]}
		s.recordAction(la)
		s.logActionOn(mc.seat, mc.targetPlayer, mc.text("hit one"))

	default:
		if s.LifePool > 0 {
			s.LifePool--
			la.LivesLost = 1
		}
		la.Outcome = OutcomeMiss
		switch {
		case len(nonGold) == 1:
			// Gold is never tokened: the lone non-gold target takes it.
			s.placeToken(nonGold[0])
		case s.LifePool > 0:
			// The life is already lost; the target player picks which of the
			// non-gold targets gets the token, then the turn passes.
			s.PendingSafeChoice = &PendingSafeChoice{
				Caller:       mc.seat,
				TargetPlayer: mc.targetPlayer,
				SourceCard:   sourceID,
				Options:      nonGold,
				Miss:         true,
				Joker:        mc.joker,
				PassTurn:     mc.passTurn,
			}
			s.Phase = PhaseAwaitingResponse
			la.Outcome = OutcomeAwaitingToken
			la.Targets = append([]CardID(nil), nonGold...)
			s.recordAction(la)
			s.logActionOn(mc.seat, mc.targetPlayer, mc.text("missed"))
			return ResultNone, nil
		}
		// Otherwise the last life is gone and the game ends below.
		s.recordAction(la)
		s.logActionOn(mc.seat, mc.targetPlayer, mc.text("missed"))
	}

	return e.endCall(mc.seat, mc.passTurn), nil
}

// endCall ends a resolved call: it passes the turn from caller when passTurn,
// else only checks for a terminal state.
func (e *Engine) endCall(caller SeatID, passTurn bool) Result {
	if passTurn {
		return e.finishTurn(caller)
	}
	e.state.noteSilverDone()
	return e.checkTerminal()
}

// slotsText renders ascending slot numbers as "a, b and c" for the log.
func (s *GameState) slotsText(ids []CardID) string {
	slots := make([]int, len(ids))
	for i, id := range ids {
		slots[i] = s.Slot[id]
	}
	for i := range slots {
		for j := i + 1; j < len(slots); j++ {
			if slots[j] < slots[i] {
				slots[i], slots[j] = slots[j], slots[i]
			}
		}
	}
	out := ""
	for i, n := range slots {
		switch {
		case i == 0:
		case i == len(slots)-1:
			out += " and "
		default:
			out += ", "
		}
		out += strconv.Itoa(n)
	}
	return out
}

func countWord(n int) string {
	switch n {
	case 2:
		return "two"
	case 3:
		return "three"
	}
	return strconv.Itoa(n)
}
