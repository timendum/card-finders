package engine

import (
	"reflect"
	"testing"
)

func TestJokerTripleCallValidationAndHit(t *testing.T) {
	e := jokerEngine(3, JokerTripleCall)
	place(e, 1, 0, suited(7))
	place(e, 2, 0, suited(2))
	place(e, 3, 1, suited(7))
	place(e, 4, 1, suited(9))
	place(e, 5, 1, suited(8))
	place(e, 6, 2, suited(5))
	layout(e)
	e.state.Players[0].PowerUsed = true // independent of Safe_Call

	call := Action{Joker: JokerTripleCall, SourceCard: 1, TargetCard: 3, TargetCard2: 4, TargetCard3: 5}
	wantJokerErr(t, e, 1, call, ErrNotYourTurn)
	bad := call
	bad.TargetCard3 = 6 // a different seat
	wantJokerErr(t, e, 0, bad, ErrJokerInvalid)
	bad = call
	bad.TargetCard3 = 4 // duplicate
	wantJokerErr(t, e, 0, bad, ErrJokerInvalid)
	bad = call
	bad.SourceCard = 3 // not mine
	wantJokerErr(t, e, 0, bad, ErrInvalidSource)

	mustJoker(t, e, 0, call)
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[3].Facing != FaceUp {
		t.Error("source and the matching target should flip")
	}
	if e.state.CurrentSeat != 1 {
		t.Errorf("CurrentSeat = %d, want the turn passed", e.state.CurrentSeat)
	}
	if la := e.View(2).LastAction; la.Kind != ActionKindJoker || la.Joker != JokerTripleCall {
		t.Errorf("last action = %+v, want joker 4", la)
	}
}

func TestJokerSweepMissAndRestrictions(t *testing.T) {
	e := jokerEngine(3, JokerSweep)
	place(e, 1, 0, silver(5))
	place(e, 2, 0, suited(7))
	place(e, 3, 1, suited(8))
	place(e, 4, 1, goldOf(3))
	layout(e)

	wantJokerErr(t, e, 0, Action{Joker: JokerSweep, SourceCard: 1, TargetSeat: 1}, ErrJokerInvalid) // silver source
	wantJokerErr(t, e, 0, Action{Joker: JokerSweep, SourceCard: 2, TargetSeat: 0}, ErrJokerInvalid) // self
	mustJoker(t, e, 0, Action{Joker: JokerSweep, SourceCard: 2, TargetSeat: 1})
	if e.state.LifePool != 2 || e.state.Cards[3].Token == nil || e.state.Cards[4].Token != nil {
		t.Errorf("life %d: a miss costs one life and tokens the non-gold card only", e.state.LifePool)
	}
}

func TestJokerSweepOnlyGoldIsLoss(t *testing.T) {
	e := jokerEngine(3, JokerSweep)
	place(e, 1, 0, suited(7))
	place(e, 2, 1, goldOf(3))
	layout(e)
	res, err := e.Apply(0, Action{Kind: ActionKindJoker, Joker: JokerSweep, SourceCard: 1, TargetSeat: 1})
	if err != nil || res != ResultLoss {
		t.Errorf("res=%d err=%v, want a loss", res, err)
	}
}

func TestJokerSweepSeveralHitsTargetChooses(t *testing.T) {
	e := jokerEngine(3, JokerSweep)
	place(e, 1, 0, suited(7))
	place(e, 2, 0, suited(2))
	place(e, 3, 1, suited(7))
	place(e, 4, 1, suited(7))
	place(e, 5, 1, suited(9))
	layout(e)
	mustJoker(t, e, 0, Action{Joker: JokerSweep, SourceCard: 1, TargetSeat: 1})
	v := e.View(1)
	if v.Phase != PhaseAwaitingResponse || !v.HasResponsePlayer || v.ResponsePlayer != 1 {
		t.Fatalf("view = phase %d response %d, want seat 1 to choose", v.Phase, v.ResponsePlayer)
	}
	if got := v.LastAction.Targets; !reflect.DeepEqual(got, []CardID{3, 4}) {
		t.Errorf("options = %v, want [3 4]", got)
	}
	if v.LastAction.Joker != JokerSweep {
		t.Errorf("last action joker = %d", v.LastAction.Joker)
	}
}

func TestJokerReverseCall(t *testing.T) {
	setup := func(target Card) *Engine {
		e := jokerEngine(3, JokerReverseCall)
		place(e, 1, 0, suited(7))
		place(e, 2, 0, silver(5))
		place(e, 3, 0, suited(7))
		place(e, 4, 0, goldOf(2))
		place(e, 5, 1, target)
		place(e, 6, 1, suited(12))
		layout(e)
		return e
	}
	call := Action{Joker: JokerReverseCall, SourceCard: 1, SourceCard2: 2, TargetCard: 5}

	e := setup(suited(7))
	wantJokerErr(t, e, 0, Action{Joker: JokerReverseCall, SourceCard: 1, SourceCard2: 3, TargetCard: 5}, ErrJokerInvalid) // same identity
	wantJokerErr(t, e, 0, Action{Joker: JokerReverseCall, SourceCard: 1, SourceCard2: 4, TargetCard: 5}, ErrGoldSource)
	wantJokerErr(t, e, 0, Action{Joker: JokerReverseCall, SourceCard: 1, SourceCard2: 2, TargetCard: 3}, ErrTargetOwn)
	mustJoker(t, e, 0, call)
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[5].Facing != FaceUp || e.state.Cards[2].Facing != FaceDown {
		t.Error("the matching own card and the target flip; the other own card stays")
	}

	e = setup(silver(9))
	mustJoker(t, e, 0, call)
	if e.state.Cards[2].Facing != FaceUp || e.state.Cards[5].Facing != FaceUp || e.state.Cards[1].Facing != FaceDown {
		t.Error("a silver target matches the silver own card")
	}

	e = setup(suited(9))
	mustJoker(t, e, 0, call)
	if e.state.LifePool != 2 || e.state.Cards[5].Token == nil || e.state.Cards[1].Facing != FaceDown {
		t.Error("a miss costs a life and tokens the target only")
	}

	e = setup(goldOf(9))
	if res, err := e.Apply(0, Action{Kind: ActionKindJoker, Joker: JokerReverseCall, SourceCard: 1, SourceCard2: 2, TargetCard: 5}); err != nil || res != ResultLoss {
		t.Errorf("gold target: res=%d err=%v, want a loss", res, err)
	}
}

func TestJokerSwapFlow(t *testing.T) {
	e := jokerEngine(3, JokerSwap)
	place(e, 1, 2, suited(3))
	place(e, 2, 2, suited(9))
	place(e, 3, 1, suited(5))
	place(e, 4, 1, goldOf(8))
	place(e, 5, 0, suited(6))
	layout(e)

	wantJokerErr(t, e, 2, Action{Joker: JokerSwap, SourceCard: 3, TargetSeat: 1}, ErrJokerInvalid) // not mine
	wantJokerErr(t, e, 2, Action{Joker: JokerSwap, SourceCard: 1, TargetSeat: 2}, ErrJokerInvalid) // self
	mustJoker(t, e, 2, Action{Joker: JokerSwap, SourceCard: 1, TargetSeat: 1})

	v := e.View(0)
	if v.Phase != PhaseAwaitingResponse || v.ResponsePlayer != 1 || len(v.LastAction.Targets) != 0 ||
		v.LastAction.Outcome != OutcomeAwaitingChoice {
		t.Fatalf("view = %+v, want seat 1 to pick any own card (empty Targets)", v)
	}
	if e.state.CurrentSeat != 0 {
		t.Error("the swap must not move the turn")
	}
	if _, err := e.Apply(0, Action{Kind: ActionKindGuess, SourceCard: 5, TargetCard: 3}); err != ErrWrongPhase {
		t.Errorf("guess while pending: err = %v, want ErrWrongPhase", err)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: 2}); err != ErrInvalidChoice {
		t.Errorf("choosing another seat's card: err = %v, want ErrInvalidChoice", err)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: 4}); err != nil {
		t.Fatal(err)
	}

	if e.state.Phase != PhasePlaying || e.state.CurrentSeat != 0 {
		t.Errorf("phase %d seat %d, want Playing with the turn untouched", e.state.Phase, e.state.CurrentSeat)
	}
	if e.state.Cards[1].Owner != 1 || e.state.Cards[4].Owner != 2 {
		t.Error("cards did not change owner")
	}
	// The 3 goes before the 5; the gold 8 goes between the 3's old slot gap and the 9.
	if got, want := slotsOf(e, 1), []CardID{1, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("seat 1 hand = %v, want %v", got, want)
	}
	if got, want := slotsOf(e, 2), []CardID{4, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("seat 2 hand = %v, want %v", got, want)
	}
}
