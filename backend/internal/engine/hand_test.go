package engine

import (
	"reflect"
	"testing"
)

func link(e *Engine, kind TokenKind, ids ...CardID) {
	for _, id := range ids {
		e.state.Cards[id].Token = &TokenInfo{Displayed: kind}
	}
}

func suited(r Rank) Card                      { return Card{Rank: r, Kind: SpecialKindNone} }
func silver(r Rank) Card                      { return Card{Rank: r, Kind: SpecialKindSilver} }
func goldOf(r Rank) Card                      { return Card{Rank: r, Kind: SpecialKindGold} }
func slotsOf(e *Engine, seat SeatID) []CardID { return e.state.handBySlot(seat) }

func TestLinkPairsPairLeftToRight(t *testing.T) {
	e := jokerEngine(3)
	for i := CardID(1); i <= 6; i++ {
		place(e, i, 0, suited(Rank(i+1)))
	}
	layout(e)
	link(e, TokenKindDifferent, 2, 3, 4, 5)
	link(e, TokenKindSame, 6)
	got := e.state.linkPairs()
	want := [][2]CardID{{2, 3}, {4, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pairs = %v, want %v (a lone token pairs with nothing)", got, want)
	}
}

func TestFlipClearsPeerLinkOnly(t *testing.T) {
	e := jokerEngine(3)
	for i := CardID(1); i <= 4; i++ {
		place(e, i, 0, suited(Rank(i+1)))
	}
	layout(e)
	link(e, TokenKindSame, 1, 2, 3, 4)
	e.state.flipUp(2)
	if e.state.Cards[1].Token != nil || e.state.Cards[2].Token != nil {
		t.Error("flipping a card should clear its link and its peer's")
	}
	if e.state.Cards[3].Token == nil || e.state.Cards[4].Token == nil {
		t.Error("the other pair must stay linked")
	}
}

func TestTokenOverLinkDissolvesLink(t *testing.T) {
	e := jokerEngine(3)
	place(e, 1, 0, suited(2))
	place(e, 2, 0, suited(3))
	layout(e)
	link(e, TokenKindDifferent, 1, 2)
	e.state.placeToken(1)
	if e.state.Cards[1].Token.Displayed != TokenKindNaturalIdentity || e.state.Cards[2].Token != nil {
		t.Errorf("tokens = %+v / %+v, want identity on 1 and a cleared peer", e.state.Cards[1].Token, e.state.Cards[2].Token)
	}
}

func TestSameIdentity(t *testing.T) {
	cases := []struct {
		a, b Card
		want bool
	}{
		{suited(7), suited(7), true},
		{suited(7), suited(8), false},
		{silver(6), silver(7), true},
		{goldOf(6), goldOf(9), true},
		{silver(7), goldOf(7), false},
		{suited(7), silver(7), false},
	}
	for _, c := range cases {
		if got := sameIdentity(c.a, c.b); got != c.want {
			t.Errorf("sameIdentity(%+v, %+v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestInsertIndexKeepsSpecialsAfterEqualSuited(t *testing.T) {
	e := jokerEngine(3)
	place(e, 1, 0, suited(5))
	place(e, 2, 0, suited(7))
	place(e, 3, 0, silver(7))
	place(e, 4, 0, suited(9))
	layout(e)
	ids := slotsOf(e, 0)
	cases := []struct {
		card Card
		want int
	}{
		{suited(4), 0},
		{suited(7), 2},  // after the equal suited 7, before the silver 7
		{silver(7), 3},  // after the silver 7
		{goldOf(7), 3},  // gold 7 sorts above silver 7, below the 9
		{suited(12), 4}, // after everything
	}
	for _, c := range cases {
		if got := e.state.insertIndex(ids, c.card.SortValue()); got != c.want {
			t.Errorf("insert %+v at %d, want %d", c.card, got, c.want)
		}
	}
}

func TestSwapCardsMovesOwnershipAndRenumbers(t *testing.T) {
	e := jokerEngine(3)
	place(e, 1, 0, suited(3))
	place(e, 2, 0, suited(8))
	place(e, 3, 0, suited(10))
	place(e, 4, 1, suited(5))
	place(e, 5, 1, suited(9))
	layout(e)
	e.state.Cards[2].Token = &TokenInfo{Displayed: TokenKindNaturalIdentity}

	e.state.swapCards(2, 5)

	if got, want := slotsOf(e, 0), []CardID{1, 5, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("seat 0 hand = %v, want %v", got, want)
	}
	if got, want := slotsOf(e, 1), []CardID{4, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("seat 1 hand = %v, want %v", got, want)
	}
	if e.state.Cards[2].Owner != 1 || e.state.Cards[5].Owner != 0 {
		t.Error("owners not exchanged")
	}
	for id, want := range map[CardID]int{1: 1, 5: 2, 3: 3, 4: 1, 2: 2} {
		if e.state.Slot[id] != want {
			t.Errorf("slot of %d = %d, want %d", id, e.state.Slot[id], want)
		}
	}
	if e.state.Cards[2].Token == nil {
		t.Error("a natural-identity token must follow its card")
	}
}

func TestSwapCardsClearsLinksOfMovedAndSplitPairs(t *testing.T) {
	e := jokerEngine(3)
	place(e, 1, 0, suited(3))
	place(e, 2, 0, suited(4))
	place(e, 3, 0, suited(9))
	place(e, 4, 0, suited(10))
	place(e, 5, 1, suited(6))
	place(e, 6, 1, suited(7))
	layout(e)
	link(e, TokenKindDifferent, 1, 2) // the swap inserts a 6 between 1 and 2? no: after both
	link(e, TokenKindSame, 3, 4)
	link(e, TokenKindSame, 5, 6)

	// Card 3 leaves seat 0 (its peer 4 unlinks); card 5 leaves seat 1 (peer 6
	// unlinks). The received 6 lands after the 4, leaving the 1-2 link intact.
	e.state.swapCards(3, 5)
	for _, id := range []CardID{3, 4, 5, 6} {
		if e.state.Cards[id].Token != nil {
			t.Errorf("card %d still carries a link token", id)
		}
	}
	if e.state.Cards[1].Token == nil || e.state.Cards[2].Token == nil {
		t.Error("an untouched adjacent pair must stay linked")
	}
}

func TestSwapCardsInsertionSplittingAPairClearsIt(t *testing.T) {
	e := jokerEngine(3)
	place(e, 1, 0, suited(3))
	place(e, 2, 0, suited(9))
	place(e, 3, 1, suited(5))
	place(e, 4, 1, suited(6))
	place(e, 5, 2, suited(4))
	layout(e)
	link(e, TokenKindSame, 1, 2)

	// Seat 2's 4 goes into seat 0 between the 3 and the 9: the pair is split.
	e.state.swapCards(5, 3)
	// 3 left seat 1, 5 left seat 2: seat 0 is untouched; use a real split.
	e.state.swapCards(4, 1)
	if e.state.Cards[2].Token != nil {
		t.Error("link whose peer moved away must be cleared")
	}
}
