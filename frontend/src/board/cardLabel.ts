// Pure display-label helpers shared by the card components and ValueSummary.

import { SpecialKind, SuitValue, type CardValue, type Rank, type Suit } from "../types.ts";

export interface ValueLike {
  Rank: Rank;
  Kind: SpecialKind;
}

// Out-of-range ranks fall back to their number so the function stays total.
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

// No suit: a value identity carries none. cardLabel adds the glyph for concrete cards.
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

// Not for card rendering: cards are suit-free and use valueLabel. Kept for non-card use.
export function cardLabel(value: CardValue): string {
  if (value.Kind === SpecialKind.None) {
    return `${rankLabel(value.Rank)}${suitGlyph(value.Suit)}`;
  }
  return valueLabel(value);
}
