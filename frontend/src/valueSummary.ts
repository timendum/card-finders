// Value_Summary pure logic: ordering the values in play and computing a
// found-over-total count for each.
//
// These are pure, I/O-free functions shared by the UI. Reactivity (recomputing
// on each `state` push) is the component's job; here we simply derive the
// summary from a given PlayerView snapshot, so re-running after a reveal always
// reflects the current card states.
//
// Confirmed against backend/internal/engine/{card,deck}.go: the Base_Deck is 48
// cards — 4 suits x ranks 2..king (13), no ace — so every suited rank has 4
// cards in play. Each Special_Pool entry (one silver or gold identity) is a
// single distinct value, total 1. Sort_Value matches Card.SortValue: suited =
// rank, silver = rank + 0.3, gold = rank + 0.6.

import {
  Face,
  SpecialKind,
  type CardValue,
  type PlayerView,
  type Rank,
} from "./types.ts";

/** Lowest suited/special rank (2) and highest suited rank (king, 13). */
export const MIN_RANK: Rank = 2;
export const MAX_RANK: Rank = 13;

/** Number of cards of each suited rank in the Base_Deck (one per suit). */
export const SUITED_RANK_TOTAL = 4;

/** A value in play: a suited rank (Kind None) or a special identity. */
export interface ValueIdentity {
  Rank: Rank;
  Kind: SpecialKind;
}

/** One Value_Summary entry: a value identity plus its ascending sort value. */
export interface ValueSummaryEntry extends ValueIdentity {
  /** Ascending ordering key (suited = rank, silver = +0.3, gold = +0.6). */
  SortValue: number;
}

/** A Value_Summary entry resolved against a PlayerView: found out of total. */
export interface ValueSummaryCount extends ValueSummaryEntry {
  /** Cards of this value currently face-up in the view. */
  Found: number;
  /** Total cards of this value in play (4 for a suited rank, 1 per special). */
  Total: number;
}

/**
 * sortValue returns the display-ordering value for a value identity, matching
 * the backend Card.SortValue formula: an ordinary suited
 * card sorts at its integer rank; a silver adds 0.3 and a gold adds 0.6, so a
 * silver 4 (4.3) and gold 4 (4.6) both sit between the 4 and the 5.
 */
export function sortValue(value: ValueIdentity): number {
  switch (value.Kind) {
    case SpecialKind.Silver:
      return value.Rank + 0.3;
    case SpecialKind.Gold:
      return value.Rank + 0.6;
    default:
      return value.Rank;
  }
}

/**
 * buildValueSummaryOrder returns every value in play ordered by ascending
 * Sort_Value: the suited ranks 2..king (13) plus each Silver
 * and Gold identity from the view's Special_Pool. Because a silver sorts at
 * rank + 0.3 and a gold at rank + 0.6, each special falls immediately after its
 * base rank and before the next integer rank (e.g. 4, silver 4, gold 4, 5).
 *
 * The result is pure and depends only on the Special_Pool; card facings do not
 * affect ordering.
 */
export function buildValueSummaryOrder(view: PlayerView): ValueSummaryEntry[] {
  const entries: ValueSummaryEntry[] = [];

  // Suited ranks 2..king, one entry each (Kind None).
  for (let rank = MIN_RANK; rank <= MAX_RANK; rank++) {
    entries.push({ Rank: rank, Kind: SpecialKind.None, SortValue: rank });
  }

  // Each Special_Pool identity is a distinct value in play.
  for (const silver of view.SpecialPool.Silvers) {
    const id: ValueIdentity = { Rank: silver.Rank, Kind: SpecialKind.Silver };
    entries.push({ ...id, SortValue: sortValue(id) });
  }
  for (const gold of view.SpecialPool.Golds) {
    const id: ValueIdentity = { Rank: gold.Rank, Kind: SpecialKind.Gold };
    entries.push({ ...id, SortValue: sortValue(id) });
  }

  // Ascending Sort_Value. Ties cannot occur between distinct values here: each
  // kind holds distinct ranks, and silver/gold offsets (0.3/0.6) keep a shared
  // base rank's silver and gold apart.
  entries.sort((a, b) => a.SortValue - b.SortValue);
  return entries;
}

/** True when two value identities name the same value (rank and kind). */
function sameValue(a: ValueIdentity, b: CardValue): boolean {
  return a.Rank === b.Rank && a.Kind === b.Kind;
}

/** Total cards of a value in play: 4 for a suited rank, 1 per special. */
function totalInPlay(value: ValueIdentity): number {
  return value.Kind === SpecialKind.None ? SUITED_RANK_TOTAL : 1;
}

/**
 * countFound returns how many cards of the given value are currently face-up in
 * the view. Only Face_Up_Cards carry a non-null Value, so a
 * face-down or hidden card never contributes to the found count.
 */
export function countFound(view: PlayerView, value: ValueIdentity): number {
  let found = 0;
  for (const card of view.Cards) {
    if (card.Facing === Face.Up && card.Value != null && sameValue(value, card.Value)) {
      found++;
    }
  }
  return found;
}

/**
 * buildValueSummary returns the ordered Value_Summary with a found-over-total
 * count for every value in play. It is pure: recomputing from a
 * later PlayerView snapshot (e.g. after a card becomes face-up) yields the
 * updated counts, which is how the component keeps the display live.
 */
export function buildValueSummary(view: PlayerView): ValueSummaryCount[] {
  return buildValueSummaryOrder(view).map((entry) => ({
    ...entry,
    Found: countFound(view, entry),
    Total: totalInPlay(entry),
  }));
}
