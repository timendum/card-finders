// Pure display-label helpers for card values.
//
// These turn an engine CardValue / value identity into the short human label
// the table shows: a suited rank as its pip ("2".."10") or face letter
// (J/Q/K), a silver or gold special as "S"/"G" with its rank. Everything here
// is pure and I/O-free so HandArea and ValueSummary can
// share one formatting source of truth.

import { SpecialKind, SuitValue, type CardValue, type Rank, type Suit } from "./types.ts";

/** Minimal shape needed to label a value: its rank and special kind. */
export interface ValueLike {
  Rank: Rank;
  Kind: SpecialKind;
}

/**
 * rankLabel renders a rank as the symbol shown on a card: 2..10 as their
 * digits, and the three court ranks as J (11), Q (12), K (13). Ranks outside
 * 2..13 fall back to their number so the function stays total.
 */
export function rankLabel(rank: Rank): string {
  switch (rank) {
    case 11:
      return "J";
    case 12:
      return "Q";
    case 13:
      return "K";
    default:
      return String(rank);
  }
}

/** One-character suit glyph for a suited card (Spades/Hearts/Diamonds/Clubs). */
export function suitGlyph(suit: Suit): string {
  switch (suit) {
    case SuitValue.Spades:
      return "\u2660"; // ♠
    case SuitValue.Hearts:
      return "\u2665"; // ♥
    case SuitValue.Diamonds:
      return "\u2666"; // ♦
    case SuitValue.Clubs:
      return "\u2663"; // ♣
    default:
      return "";
  }
}

/**
 * valueLabel renders a value identity (rank + kind) as a short label: a silver
 * prefixes "S", a gold prefixes "G", and an ordinary suited value is just its
 * rank label. Suit is not included here because a value identity does not carry
 * a meaningful suit; cardLabel below adds the suit glyph for concrete cards.
 */
export function valueLabel(value: ValueLike): string {
  switch (value.Kind) {
    case SpecialKind.Silver:
      return `S${rankLabel(value.Rank)}`;
    case SpecialKind.Gold:
      return `G${rankLabel(value.Rank)}`;
    default:
      return rankLabel(value.Rank);
  }
}

/**
 * cardLabel renders a concrete CardValue for display on a card face: a suited
 * card shows its rank plus suit glyph (e.g. "10♥"); a silver or gold shows its
 * special label (e.g. "S4", "G7"), where Suit is not meaningful.
 */
export function cardLabel(value: CardValue): string {
  if (value.Kind === SpecialKind.None) {
    return `${rankLabel(value.Rank)}${suitGlyph(value.Suit)}`;
  }
  return valueLabel(value);
}
