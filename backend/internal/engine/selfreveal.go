package engine

func (e *Engine) applySelfReveal(seat SeatID, action Action) (Result, error) {
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

	target := action.SelfRevealValue

	var toReveal []CardID
	var err error
	switch target.Kind {
	case SpecialKindGold:
		toReveal, err = e.goldSelfRevealTargets(seat)
	case SpecialKindSilver:
		toReveal, err = e.valueSelfRevealTargets(seat, SpecialKindSilver, 0)
	default: // suited rank in target.Rank
		toReveal, err = e.valueSelfRevealTargets(seat, SpecialKindNone, target.Rank)
	}
	if err != nil {
		return ResultNone, err
	}

	s.flipUp(toReveal...)

	// Announce the self-revealed value from the chosen target. Use any revealed
	// card as the concrete identity source (all share the announced value).
	av := announcedValueOf(s.Cards[toReveal[0]].Card)
	s.recordAction(&LastAction{
		Actor:          seat,
		Kind:           ActionKindSelfReveal,
		AnnouncedValue: av,
		Outcome:        OutcomeNone,
		Revealed:       append([]CardID(nil), toReveal...),
	})

	return e.finishTurn(s.CurrentSeat), nil
}

// valueSelfRevealTargets handles the suited-rank and silver branches (gold goes
// through goldSelfRevealTargets). Rejects if another player holds a matching
// face-down card, or the seat holds none.
func (e *Engine) valueSelfRevealTargets(seat SeatID, kind SpecialKind, rank Rank) ([]CardID, error) {
	var own []CardID
	for _, pc := range e.state.Cards {
		if pc.Facing != FaceDown {
			continue
		}
		if !cardHasValue(pc.Card, kind, rank) {
			continue
		}
		if pc.Owner != seat {
			return nil, ErrSelfRevealIncomplete
		}
		own = append(own, pc.Card.ID)
	}
	if len(own) == 0 {
		return nil, ErrSelfRevealIncomplete
	}
	return own, nil
}

// goldSelfRevealTargets rejects if the seat still holds any non-gold face-down
// card, or holds no gold.
func (e *Engine) goldSelfRevealTargets(seat SeatID) ([]CardID, error) {
	var golds []CardID
	for _, pc := range e.state.Cards {
		if pc.Owner != seat || pc.Facing != FaceDown {
			continue
		}
		if pc.Card.Kind != SpecialKindGold {
			return nil, ErrGoldRevealIneligible
		}
		golds = append(golds, pc.Card.ID)
	}
	if len(golds) == 0 {
		return nil, ErrGoldRevealIneligible
	}
	return golds, nil
}
