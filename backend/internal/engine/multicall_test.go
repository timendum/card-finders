package engine

import "testing"

// callState: seat 0 owns a suited 7 (id 1) and a spare card; seat 1 owns the
// cards in targets (ids 10..).
func callState(lives int, targets ...Card) *Engine {
	e := jokerEngine(lives)
	place(e, 1, 0, suited(7))
	place(e, 2, 0, suited(2))
	for i, c := range targets {
		place(e, CardID(10+i), 1, c)
	}
	layout(e)
	return e
}

func directCall(e *Engine, ids ...CardID) (Result, error) {
	return e.resolveMultiTarget(multiCall{
		seat:         0,
		source:       e.state.Cards[1],
		targetPlayer: 1,
		targets:      ids,
		passTurn:     true,
		text:         func(o string) string { return o },
	})
}

func TestResolveMultiTargetHitOne(t *testing.T) {
	e := callState(3, suited(7), suited(9), silver(4), suited(8))
	res, err := directCall(e, 10, 11, 12, 13)
	if err != nil || res != ResultNone {
		t.Fatalf("res=%d err=%v", res, err)
	}
	if e.state.Cards[1].Facing != FaceUp || e.state.Cards[10].Facing != FaceUp {
		t.Error("source and the lone match should flip")
	}
	if e.state.Cards[11].Facing != FaceDown || e.state.LifePool != 3 || e.state.CurrentSeat != 1 {
		t.Error("other targets untouched, no life lost, turn passed")
	}
}

func TestResolveMultiTargetSeveralHitsAwaitChoice(t *testing.T) {
	e := callState(3, suited(7), suited(9), suited(7), suited(8))
	if _, err := directCall(e, 10, 11, 12, 13); err != nil {
		t.Fatal(err)
	}
	p := e.state.PendingSafeChoice
	if e.state.Phase != PhaseAwaitingResponse || p == nil || len(p.Options) != 2 {
		t.Fatalf("pending = %+v phase %d, want a choice between the two 7s", p, e.state.Phase)
	}
	la := e.View(0).LastAction
	if la.Outcome != OutcomeAwaitingChoice || len(la.Targets) != 2 {
		t.Errorf("last action = %+v, want the options as Targets", la)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: 12}); err != nil {
		t.Fatal(err)
	}
	if e.state.Cards[12].Facing != FaceUp || e.state.Cards[1].Facing != FaceUp || e.state.Cards[10].Facing != FaceDown {
		t.Error("chosen card and source should flip, the other match stays")
	}
	if e.state.Phase != PhasePlaying || e.state.CurrentSeat != 1 {
		t.Error("choice should resume play and pass the turn")
	}
}

func TestResolveMultiTargetMissAwaitsTokenChoice(t *testing.T) {
	e := callState(3, suited(3), suited(9), goldOf(4), suited(8))
	if _, err := directCall(e, 10, 11, 12, 13); err != nil {
		t.Fatal(err)
	}
	p := e.state.PendingSafeChoice
	if e.state.LifePool != 2 || p == nil || !p.Miss || len(p.Options) != 3 {
		t.Fatalf("life %d pending %+v, want life 2 and a token choice among the 3 non-gold cards", e.state.LifePool, p)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: 12}); err != ErrInvalidChoice {
		t.Errorf("choosing the gold card: err = %v, want ErrInvalidChoice", err)
	}
	if _, err := e.Apply(1, Action{Kind: ActionKindSafeChooseReveal, ChosenTarget: 11}); err != nil {
		t.Fatal(err)
	}
	if e.state.Cards[11].Token == nil || e.state.Cards[10].Token != nil {
		t.Error("exactly the chosen target should carry the token")
	}
}

func TestResolveMultiTargetMissSingleNonGoldAutoTokens(t *testing.T) {
	e := callState(3, goldOf(4), suited(9), goldOf(5))
	if _, err := directCall(e, 10, 11, 12); err != nil {
		t.Fatal(err)
	}
	if e.state.Cards[11].Token == nil || e.state.PendingSafeChoice != nil || e.state.LifePool != 2 {
		t.Error("the lone non-gold target should take the token without a choice")
	}
}

func TestResolveMultiTargetAllGoldLoses(t *testing.T) {
	e := callState(3, goldOf(4), goldOf(5), goldOf(6))
	res, _ := directCall(e, 10, 11, 12)
	if res != ResultLoss || e.state.LifePool != 3 {
		t.Errorf("res=%d life=%d, want a loss with life unchanged", res, e.state.LifePool)
	}
	for _, id := range []CardID{10, 11, 12} {
		if e.state.Cards[id].Facing != FaceUp {
			t.Errorf("gold %d should flip", id)
		}
	}
}

func TestResolveMultiTargetLastLifeEndsWithoutChoice(t *testing.T) {
	e := callState(1, suited(3), suited(9), suited(8))
	res, _ := directCall(e, 10, 11, 12)
	if res != ResultLoss || e.state.PendingSafeChoice != nil {
		t.Errorf("res=%d pending=%+v, want loss with no pending choice", res, e.state.PendingSafeChoice)
	}
}
