// ValueSummary — center panel listing every value in play with a
// found-over-total count. Rows come from buildValueSummary; the list is a
// live region so assistive tech announces reveals.

import { useMemo } from "react";
import { valueLabel } from "../board/cardLabel.ts";
import { Face, SpecialKind, type CardValue, type PlayerView, type Rank } from "../types.ts";

export const MIN_RANK: Rank = 2;
export const MAX_RANK: Rank = 13;

export const SUITED_RANK_TOTAL = 4;

/** A value in play: a suited rank (Kind None) or a special identity. */
export interface ValueIdentity {
  Rank: Rank;
  Kind: SpecialKind;
}

export interface ValueSummaryEntry extends ValueIdentity {
  SortValue: number;
}

export interface ValueSummaryCount extends ValueSummaryEntry {
  Found: number;
  Total: number;
}

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
 * Every value in play, ordered by Sort_Value so each special lands right after
 * its base rank (e.g. 4, silver 4, gold 4, 5). Depends only on the Special_Pool.
 */
export function buildValueSummaryOrder(view: PlayerView): ValueSummaryEntry[] {
  const entries: ValueSummaryEntry[] = [];

  for (let rank = MIN_RANK; rank <= MAX_RANK; rank++) {
    entries.push({ Rank: rank, Kind: SpecialKind.None, SortValue: rank });
  }

  for (const silver of view.SpecialPool.Silvers) {
    const id: ValueIdentity = { Rank: silver.Rank, Kind: SpecialKind.Silver };
    entries.push({ ...id, SortValue: sortValue(id) });
  }
  for (const gold of view.SpecialPool.Golds) {
    const id: ValueIdentity = { Rank: gold.Rank, Kind: SpecialKind.Gold };
    entries.push({ ...id, SortValue: sortValue(id) });
  }

  entries.sort((a, b) => a.SortValue - b.SortValue);
  return entries;
}

function sameValue(a: ValueIdentity, b: CardValue): boolean {
  return a.Rank === b.Rank && a.Kind === b.Kind;
}

/** Total in play: 4 for a suited rank, 1 per special. */
function totalInPlay(value: ValueIdentity): number {
  return value.Kind === SpecialKind.None ? SUITED_RANK_TOTAL : 1;
}

/** Face-up cards of this value; only face-up cards carry a non-null Value. */
export function countFound(view: PlayerView, value: ValueIdentity): number {
  let found = 0;
  for (const card of view.Cards) {
    if (card.Facing === Face.Up && card.Value != null && sameValue(value, card.Value)) {
      found++;
    }
  }
  return found;
}

/** Ordered Value_Summary with a found-over-total count for each value. */
export function buildValueSummary(view: PlayerView): ValueSummaryCount[] {
  return buildValueSummaryOrder(view).map((entry) => ({
    ...entry,
    Found: countFound(view, entry),
    Total: totalInPlay(entry),
  }));
}

function kindName(kind: SpecialKind): "silver" | "gold" | "suited" {
  if (kind === SpecialKind.Silver) return "silver";
  if (kind === SpecialKind.Gold) return "gold";
  return "suited";
}

export interface ValueSummaryProps {
  view: PlayerView;
}

export function ValueSummary({ view }: ValueSummaryProps) {
  const rows = useMemo(() => buildValueSummary(view), [view]);

  return (
    <section className="value-summary" aria-label="Value summary">
      <h2 className="value-summary-title">Values in play</h2>
      <ul className="value-summary-list" aria-live="polite">
        {rows.map((row) => {
          const dataCount =
            row.Found === row.Total ? "complete" : row.Found > 0 ? "partial" : "none";
          return (
            <li
              key={`${row.Kind}-${row.Rank}`}
              className="value-summary-row"
              data-count={dataCount}
              data-kind={kindName(row.Kind)}
            >
              <span className="value-summary-label">{valueLabel(row)}</span>
              <span className="value-summary-count">
                {row.Found}/{row.Total}
                {dataCount === "complete" ? (
                  <span className="value-summary-done" role="img" aria-label="complete">
                    {" ✓"}
                  </span>
                ) : null}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
