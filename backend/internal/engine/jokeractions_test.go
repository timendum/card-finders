package engine

import "testing"

func useJoker(e *Engine, seat SeatID, a Action) error {
	a.Kind = ActionKindJoker
	_, err := e.Apply(seat, a)
	return err
}

func mustJoker(t *testing.T, e *Engine, seat SeatID, a Action) {
	t.Helper()
	if err := useJoker(e, seat, a); err != nil {
		t.Fatalf("joker %d: %v", a.Joker, err)
	}
}

func wantJokerErr(t *testing.T, e *Engine, seat SeatID, a Action, want error) {
	t.Helper()
	if err := useJoker(e, seat, a); err != want {
		t.Fatalf("joker %d: err = %v, want %v", a.Joker, err, want)
	}
}

func TestJokerLinkDifferAndMatchClaims(t *testing.T) {
	e := jokerEngine(3, JokerDiffer, JokerMatch)
	place(e, 1, 0, suited(3))
	place(e, 2, 0, suited(9))
	place(e, 3, 0, silver(6))
	place(e, 4, 0, silver(7))
	place(e, 5, 1, suited(2)) // keeps the board unfinished
	layout(e)

	// Match on a false claim is rejected and changes nothing.
	wantJokerErr(t, e, 0, Action{Joker: JokerMatch, SourceCard: 1, TargetCard: 2}, ErrJokerInvalid)
	if e.state.Cards[1].Token != nil || e.state.Jokers[1].Used {
		t.Fatal("rejected joker mutated state")
	}
	// Non-adjacent pair is rejected.
	wantJokerErr(t, e, 0, Action{Joker: JokerDiffer, SourceCard: 1, TargetCard: 3}, ErrJokerInvalid)
	// Another seat's cards are not mine to link.
	wantJokerErr(t, e, 1, Action{Joker: JokerDiffer, SourceCard: 1, TargetCard: 2}, ErrJokerInvalid)

	// Anytime: seat 2 is not the current seat but may use a joker; the pair
	// belongs to seat 0 so seat 0 does it, and the turn does not move.
	mustJoker(t, e, 0, Action{Joker: JokerDiffer, SourceCard: 1, TargetCard: 2})
	for _, id := range []CardID{1, 2} {
		if tk := e.state.Cards[id].Token; tk == nil || tk.Displayed != TokenKindDifferent {
			t.Errorf("card %d token = %+v, want Different", id, tk)
		}
	}
	if e.state.CurrentSeat != 0 || e.state.Phase != PhasePlaying {
		t.Error("an anytime joker must not advance the turn")
	}
	// Silver 6 and silver 7 are the "same": specials compare by kind only.
	mustJoker(t, e, 0, Action{Joker: JokerMatch, SourceCard: 3, TargetCard: 4})
	if tk := e.state.Cards[3].Token; tk == nil || tk.Displayed != TokenKindSame {
		t.Errorf("silver pair token = %+v, want Same", tk)
	}
	if vc := e.View(1).Cards[0]; vc.Token == nil || vc.TokenDisplay != nil {
		t.Errorf("link token view = %+v, want a token with no TokenDisplay", vc)
	}
}

func TestJokerDifferSilverGoldAndTokenedRejected(t *testing.T) {
	e := jokerEngine(3, JokerDiffer)
	place(e, 1, 0, silver(5))
	place(e, 2, 0, goldOf(5))
	place(e, 3, 0, suited(9))
	layout(e)
	e.state.Cards[3].Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}
	wantJokerErr(t, e, 0, Action{Joker: JokerDiffer, SourceCard: 2, TargetCard: 3}, ErrJokerInvalid)
	mustJoker(t, e, 0, Action{Joker: JokerDiffer, SourceCard: 1, TargetCard: 2})
}

func TestJokerSelfMark(t *testing.T) {
	e := jokerEngine(3, JokerSelfMark)
	place(e, 1, 0, silver(5))
	place(e, 2, 0, suited(6))
	layout(e)
	wantJokerErr(t, e, 0, Action{Joker: JokerSelfMark, SourceCard: 1}, ErrJokerInvalid)
	wantJokerErr(t, e, 2, Action{Joker: JokerSelfMark, SourceCard: 2}, ErrJokerInvalid) // not mine
	mustJoker(t, e, 0, Action{Joker: JokerSelfMark, SourceCard: 2})
	if tk := e.state.Cards[2].Token; tk == nil || tk.Displayed != TokenKindNaturalIdentity {
		t.Errorf("token = %+v, want a natural-identity token", tk)
	}
	if e.state.CurrentSeat != 0 {
		t.Error("anytime joker advanced the turn")
	}
}

func TestJokerExtraLifeExceedsStart(t *testing.T) {
	e := jokerEngine(4, JokerExtraLife)
	place(e, 1, 0, suited(2))
	layout(e)
	mustJoker(t, e, 3, Action{Joker: JokerExtraLife})
	if e.state.LifePool != 5 {
		t.Errorf("LifePool = %d, want 5", e.state.LifePool)
	}
	if e.state.CurrentSeat != 0 {
		t.Error("anytime joker advanced the turn")
	}
}

func TestJokerSecondWind(t *testing.T) {
	setup := func(spent ...SeatID) *Engine {
		e := jokerEngine(3, JokerSecondWind)
		place(e, 1, 0, suited(2))
		layout(e)
		for _, sd := range spent {
			e.state.Players[sd].PowerUsed = true
		}
		return e
	}

	// None spent: the joker is wasted.
	e := setup()
	mustJoker(t, e, 0, Action{Joker: JokerSecondWind})
	if !e.state.Jokers[0].Used {
		t.Error("joker not consumed")
	}

	// At most two spent: restored automatically.
	e = setup(1, 3)
	mustJoker(t, e, 0, Action{Joker: JokerSecondWind})
	if e.state.Players[1].PowerUsed || e.state.Players[3].PowerUsed {
		t.Error("spent powers not restored")
	}

	// More than two spent: the activator picks two valid, distinct seats.
	e = setup(0, 1, 2)
	wantJokerErr(t, e, 0, Action{Joker: JokerSecondWind, TargetSeat: 1, TargetSeat2: 1}, ErrJokerInvalid)
	wantJokerErr(t, e, 0, Action{Joker: JokerSecondWind, TargetSeat: 1, TargetSeat2: 3}, ErrJokerInvalid)
	mustJoker(t, e, 0, Action{Joker: JokerSecondWind, TargetSeat: 0, TargetSeat2: 2})
	if e.state.Players[0].PowerUsed || e.state.Players[2].PowerUsed || !e.state.Players[1].PowerUsed {
		t.Error("only the two named seats should be restored")
	}
}

func TestJokerCensus(t *testing.T) {
	e := jokerEngine(3, JokerCensus)
	place(e, 1, 0, suited(5))
	layout(e)
	wantJokerErr(t, e, 0, Action{Joker: JokerCensus, SelfRevealValue: SelfRevealTarget{Kind: SpecialKindSilver}}, ErrJokerInvalid)
	wantJokerErr(t, e, 0, Action{Joker: JokerCensus}, ErrJokerInvalid)
	before := len(e.state.Log)
	mustJoker(t, e, 2, Action{Joker: JokerCensus, SelfRevealValue: SelfRevealTarget{Rank: 5}})
	if got := len(e.state.Log) - before; got != seatCount {
		t.Errorf("census logged %d entries, want one per player (%d)", got, seatCount)
	}
	if e.state.CurrentSeat != 0 {
		t.Error("census advanced the turn")
	}
}

func TestJokerShieldArmsAndRestrictsActions(t *testing.T) {
	e := jokerEngine(3, JokerShield)
	place(e, 1, 0, suited(5))
	place(e, 2, 1, suited(6))
	layout(e)
	wantJokerErr(t, e, 1, Action{Joker: JokerShield}, ErrNotYourTurn)
	mustJoker(t, e, 0, Action{Joker: JokerShield})
	if !e.state.Shield || !e.View(2).Shield || e.state.CurrentSeat != 0 {
		t.Fatal("shield not armed, or the turn moved")
	}
	if _, err := e.Apply(0, Action{Kind: ActionKindSelfReveal, SelfRevealValue: SelfRevealTarget{Rank: 5}}); err != ErrShieldActive {
		t.Errorf("self-reveal err = %v, want ErrShieldActive", err)
	}
	if _, err := e.Apply(0, Action{Kind: ActionKindSafeCall, SourceCard: 1, TargetCard: 2, TargetCard2: 2}); err != ErrShieldActive {
		t.Errorf("safe call err = %v, want ErrShieldActive", err)
	}
}

func TestJokerPass(t *testing.T) {
	e := jokerEngine(3, JokerPass)
	place(e, 1, 0, suited(5))
	place(e, 2, 2, suited(6))
	layout(e)
	wantJokerErr(t, e, 1, Action{Joker: JokerPass, TargetSeat: 2}, ErrNotYourTurn)
	wantJokerErr(t, e, 0, Action{Joker: JokerPass, TargetSeat: 0}, ErrJokerInvalid)
	wantJokerErr(t, e, 0, Action{Joker: JokerPass, TargetSeat: 1}, ErrJokerInvalid) // no face-down card
	mustJoker(t, e, 0, Action{Joker: JokerPass, TargetSeat: 2})
	if e.state.CurrentSeat != 2 {
		t.Errorf("CurrentSeat = %d, want 2", e.state.CurrentSeat)
	}
}
