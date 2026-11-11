package engine

// advanceTurn moves CurrentSeat clockwise to the nearest seat holding a
// face-down card. Returns false and leaves CurrentSeat unchanged when none has
// one (the win, per checkTerminal).
func (e *Engine) advanceTurn() bool {
	s := e.state
	seat := s.CurrentSeat
	for range seatCount {
		seat = nextSeat(seat)
		if faceDownCount(s, seat) > 0 {
			s.CurrentSeat = seat
			return true
		}
	}
	return false
}

// allFaceUp reports whether every dealt card is face-up (the win condition).
func (e *Engine) allFaceUp() bool {
	for _, pc := range e.state.Cards {
		if pc.Facing == FaceDown {
			return false
		}
	}
	return true
}

// checkTerminal ends the game on a terminal condition and returns its Result
// (ResultNone while the game continues). Loss is checked first: life at zero is
// always a loss.
func (e *Engine) checkTerminal() Result {
	s := e.state
	if s.LifePool == 0 {
		s.Phase = PhaseEnded
		s.GameResult = ResultLoss
		return ResultLoss
	}
	if e.allFaceUp() {
		s.Phase = PhaseEnded
		s.GameResult = ResultWin
		return ResultWin
	}
	return ResultNone
}

// endGoldHit ends the game as a loss after a gold card was targeted. Life_Pool
// is left as-is; the caller has already flipped the gold card(s) face-up.
func (e *Engine) endGoldHit() Result {
	s := e.state
	s.PendingSafeChoice = nil
	s.Phase = PhaseEnded
	s.GameResult = ResultLoss
	return ResultLoss
}

// finishTurn ends the action: if the game ended it returns that Result without
// advancing (leaving the final board intact), else advances clockwise from
// advanceFrom. Most handlers pass CurrentSeat; safe-choose-reveal passes the
// caller's seat, since the turn ends relative to the caller, not the chooser.
func (e *Engine) finishTurn(advanceFrom SeatID) Result {
	e.state.noteSilverDone()
	if res := e.checkTerminal(); res != ResultNone {
		return res
	}
	e.state.CurrentSeat = advanceFrom
	e.advanceTurn()
	return ResultNone
}
