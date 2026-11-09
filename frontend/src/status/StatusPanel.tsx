// StatusPanel — top-center panel: remaining lives first, then the value
// summary.

import { ValueSummary } from "./ValueSummary.tsx";
import type { PlayerView } from "../types.ts";

export interface LifeDisplayProps {
  lifePool: number;
  maxLife: number;
}

export function LifeDisplay({ lifePool, maxLife }: LifeDisplayProps) {
  return (
    <div className="life-display" aria-label={`Lives: ${lifePool} of ${maxLife}`}>
      <span className="life-display-label" aria-hidden="true">
        Lives
      </span>{" "}
      <span className="life-display-count" aria-hidden="true">
        {lifePool}/{maxLife}
      </span>
    </div>
  );
}

export interface StatusPanelProps {
  view: PlayerView;
}

export function StatusPanel({ view }: StatusPanelProps) {
  return (
    <section className="status-panel" aria-label="Game status">
      <LifeDisplay lifePool={view.LifePool} maxLife={view.MaxLife} />
      <ValueSummary view={view} />
    </section>
  );
}
