// EndBanner — the end-of-game result overlay shown once the game has ended
// with a win or a loss. The engine ends the game and
// records the Result in every seat's PlayerView, so each client simply reflects
// view.Result here when the phase is Ended.
//
// It is a pure presentational component: GameScreen decides when to render it
// (phase Ended / Result != None) and passes the result and a leave handler.

import { Result } from "./types.ts";

export interface EndBannerProps {
  /** The terminal result: Win or Loss. */
  result: Result;
  /** Leave the game and return to the lobby. */
  onLeave: () => void;
}

export function EndBanner({ result, onLeave }: EndBannerProps) {
  const won = result === Result.Win;
  return (
    <div
      className="end-banner"
      role="alertdialog"
      aria-label="Game over"
      data-result={won ? "win" : "loss"}
    >
      <h2>{won ? "You win!" : "Game over"}</h2>
      <p>
        {won
          ? "Every card is revealed — the team found them all."
          : "The life pool is empty. Better luck next time."}
      </p>
      <button type="button" onClick={onLeave}>
        Back to lobby
      </button>
    </div>
  );
}
