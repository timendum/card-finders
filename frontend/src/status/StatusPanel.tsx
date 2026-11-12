// StatusPanel — top-center panel. Top row: remaining lives on the left, jokers
// (and the Shield flag when armed) on the right. Bottom row: the value summary.

import { JokerBar } from "./JokerBar.tsx";
import { ValueSummary } from "./ValueSummary.tsx";
import type { PlayerView, Rank } from "../types.ts";

export interface LifeDisplayProps {
  lifePool: number;
  maxLife: number;
}

// LifePool may exceed MaxLife (Extra life), so this shows e.g. 5/4 as is.
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

export function ShieldIndicator({ armed }: { armed: boolean }) {
  if (!armed) return null;
  return (
    <span className="shield-indicator" role="status">
      Shield armed
    </span>
  );
}

export interface StatusPanelProps {
  view: PlayerView;
  /** The joker being set up, if any. */
  activeJoker?: Rank | null;
  onActivateJoker?: (rank: Rank) => void;
}

export function StatusPanel({ view, activeJoker, onActivateJoker }: StatusPanelProps) {
  return (
    <section className="status-panel" aria-label="Game status" data-shield={view.Shield}>
      <div className="status-panel-top">
        <LifeDisplay lifePool={view.LifePool} maxLife={view.MaxLife} />
        <div className="status-panel-jokers">
          <ShieldIndicator armed={view.Shield} />
          <JokerBar view={view} active={activeJoker} onActivate={onActivateJoker} />
        </div>
      </div>
      <ValueSummary view={view} />
    </section>
  );
}
