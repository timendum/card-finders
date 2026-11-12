// JokerBar — the game's shared jokers, one button per drawn joker. State comes
// from the view; a usable joker starts its selection flow through onActivate.

import { rankLabel } from "../board/cardLabel.ts";
import { jokerDescriptions, jokerLabel, jokerUsable } from "../board/jokers.ts";
import { JokerState, type PlayerView, type Rank } from "../types.ts";

export interface JokerBarProps {
  view: PlayerView;
  /** The joker currently being set up, if any. */
  active?: Rank | null;
  onActivate?: (rank: Rank) => void;
}

const stateNames = {
  [JokerState.Inactive]: "inactive",
  [JokerState.Ready]: "ready",
  [JokerState.Used]: "used",
} as const;

export function JokerBar({ view, active = null, onActivate }: JokerBarProps) {
  if (view.Jokers.length === 0) return null;
  return (
    <ul className="joker-bar" aria-label="Jokers">
      {view.Jokers.map((joker) => {
        const usable = jokerUsable(view, joker.Rank);
        const state = stateNames[joker.State];
        return (
          <li key={joker.Rank}>
            <button
              type="button"
              className="joker"
              data-state={state}
              data-usable={usable}
              data-active={active === joker.Rank}
              disabled={!usable}
              aria-pressed={usable ? active === joker.Rank : undefined}
              title={jokerDescriptions[joker.Rank]}
              aria-label={`${jokerLabel(joker.Rank)} (${rankLabel(joker.Rank)}), ${state}`}
              onClick={() => onActivate?.(joker.Rank)}
            >
              <span className="joker-rank" aria-hidden="true">
                {rankLabel(joker.Rank)}
              </span>{" "}
              <span className="joker-name" aria-hidden="true">
                {jokerLabel(joker.Rank)}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
