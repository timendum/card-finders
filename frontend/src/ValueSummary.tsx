// ValueSummary — the center panel listing every value in play with a
// found-over-total count.
//
// The ordered rows and their found/total counts come from the shared pure
// buildValueSummary logic, so the panel is a thin view: every `state` push
// re-runs the derivation against the new snapshot and the counts move
// accordingly. Values appear in ascending Sort_Value, which places
// each silver/gold identity immediately after its base rank, and the
// list is marked as a live region so assistive tech announces reveals.

import { useMemo } from "react";
import { valueLabel } from "./cardLabel.ts";
import type { PlayerView } from "./types.ts";
import { buildValueSummary } from "./valueSummary.ts";

export interface ValueSummaryProps {
  /** The current snapshot the summary is derived from. */
  view: PlayerView;
}

export function ValueSummary({ view }: ValueSummaryProps) {
  // Recomputed per snapshot so a newly face-up card updates its row.
  const rows = useMemo(() => buildValueSummary(view), [view]);

  return (
    <section className="value-summary" aria-label="Value summary">
      <h2 className="value-summary-title">Values in play</h2>
      <ul className="value-summary-list" aria-live="polite">
        {rows.map((row) => {
          const complete = row.Found === row.Total;
          return (
            <li
              key={`${row.Kind}-${row.Rank}`}
              className="value-summary-row"
              data-complete={complete}
            >
              <span className="value-summary-label">{valueLabel(row)}</span>
              <span className="value-summary-count">
                {row.Found}/{row.Total}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
