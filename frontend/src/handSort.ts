// Pure hand-display ordering for the Card Finders frontend.
//
// Mirrors the backend display-order comparator (backend/internal/engine/order.go
// and Card.SortValue in card.go): cards are ordered by ascending sort value,
// with ties broken deterministically by suit so repeated renders of the same
// hand always produce an identical ordering.
//
// Everything here is pure and I/O-free.

import { SpecialKind, type Suit, type CardValue, type ViewCard } from "./types.ts";

/** The minimal shape the comparator needs from a card: its concrete value. */
export interface CardValueLike {
  Rank: number;
  Kind: SpecialKind;
  Suit: Suit; // meaningful only for SpecialKind.None
}

/**
 * sortValue returns the display-ordering value for a card value, matching the
 * backend `Card.SortValue` exactly: an ordinary (SpecialKind.None) card sorts
 * at its integer Rank, a Silver adds 0.3, and a Gold adds 0.6. So a silver 4
 * is 4.3 and a gold 4 is 4.6, both sitting between the 4 and the 5.
 */
export function sortValue(v: CardValueLike): number {
  switch (v.Kind) {
    case SpecialKind.Silver:
      return v.Rank + 0.3;
    case SpecialKind.Gold:
      return v.Rank + 0.6;
    default:
      return v.Rank;
  }
}

/**
 * compareCards is a pure comparator (usable with Array.prototype.sort) that
 * orders two card values by ascending sort value, breaking ties by ascending
 * Suit.
 *
 * Suit is only meaningful for SpecialKind.None cards. Distinct specials never
 * share a sort value with each other or with a suited card of the same rank
 * (suited 4 = 4.0, silver 4 = 4.3, gold 4 = 4.6), so in practice the suit
 * tiebreak is only ever reached between ordinary suited cards. To keep the
 * comparator total and deterministic for any input, a final tiebreak on Kind
 * follows the suit comparison.
 */
export function compareCards(a: CardValueLike, b: CardValueLike): number {
  const av = sortValue(a);
  const bv = sortValue(b);
  if (av !== bv) {
    return av - bv;
  }
  if (a.Suit !== b.Suit) {
    return a.Suit - b.Suit;
  }
  return a.Kind - b.Kind;
}

/**
 * compareCardValues is compareCards narrowed to the shared CardValue DTO. It is
 * the same pure comparator, exported under the DTO type for call sites that
 * sort `CardValue[]` directly.
 */
export function compareCardValues(a: CardValue, b: CardValue): number {
  return compareCards(a, b);
}

/**
 * compareViewCards orders two ViewCards by reading each card's `Value`. A
 * ViewCard's Value is null when the card is neither face-up nor owned by the
 * viewer; cards with a known value sort ahead of value-less cards, and two
 * value-less cards compare equal. This keeps the comparator total and pure.
 */
export function compareViewCards(a: ViewCard, b: ViewCard): number {
  if (a.Value == null || b.Value == null) {
    if (a.Value == null && b.Value == null) return 0;
    return a.Value == null ? 1 : -1;
  }
  return compareCards(a.Value, b.Value);
}
