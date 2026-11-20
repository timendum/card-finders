// Pure display-label helpers shared by the card components and ValueSummary.

import { SpecialKind, type Rank } from "../types.ts";

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

// No suit: a value identity carries none. cardLabel adds the glyph for concrete cards.
export function valueLabel(value: ValueLike): string {
  switch (value.Kind) {
    case SpecialKind.Silver:
      return `${rankLabel(value.Rank)}S`;
    case SpecialKind.Gold:
      return `${rankLabel(value.Rank)}G`;
    default:
      return rankLabel(value.Rank);
  }
}
