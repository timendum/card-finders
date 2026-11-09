// ValueSummary — center panel listing every value in play with a
// found-over-total count. Rows come from buildValueSummary; the list is a
// live region so assistive tech announces reveals.

import { useMemo } from "react";
import { valueLabel } from "../board/cardLabel.ts";
import type { PlayerView } from "../types.ts";
import { buildValueSummary } from "./valueSummary.ts";

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
          const complete = row.Found === row.Total;
          return (
            <li
              key={`${row.Kind}-${row.Rank}`}
              className="value-summary-row"
              data-complete={complete}
              data-partial={row.Found > 0 && !complete}
            >
              <span className="value-summary-label">{valueLabel(row)}</span>
              <span className="value-summary-count">
                {row.Found}/{row.Total}
                {complete ? (
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
