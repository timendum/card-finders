import { SpecialKind, type Suit, type CardValue, type ViewCard } from "./types.ts";

export interface CardValueLike {
  Rank: number;
  Kind: SpecialKind;
  Suit: Suit; // meaningful only for SpecialKind.None
}

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

// Ascending sort value, ties broken by Suit then Kind to stay total and
// deterministic. The suit/kind tiebreaks are only reached between suited cards,
// since distinct specials never share a sort value.
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

// compareCards narrowed to the CardValue DTO.
export function compareCardValues(a: CardValue, b: CardValue): number {
  return compareCards(a, b);
}

// Value-less cards (neither face-up nor owned) sort after cards with a value.
export function compareViewCards(a: ViewCard, b: ViewCard): number {
  if (a.Value == null || b.Value == null) {
    if (a.Value == null && b.Value == null) return 0;
    return a.Value == null ? 1 : -1;
  }
  return compareCards(a.Value, b.Value);
}
