package engine

import "sort"

// cardLess reports whether card a should be displayed before card b: ascending
// SortValue, ties broken by Suit. Distinct specials never share a SortValue, so
// the Suit tie-break is only reached between ordinary suited cards.
func cardLess(a, b Card) bool {
	av, bv := a.SortValue(), b.SortValue()
	if av != bv {
		return av < bv
	}
	return a.Suit < b.Suit
}

// SortForDisplay orders cards in place by cardLess.
func SortForDisplay(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		return cardLess(cards[i], cards[j])
	})
}
