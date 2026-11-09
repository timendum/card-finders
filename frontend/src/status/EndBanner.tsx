// the end-of-game result overlay shown once the game has ended
// with a win or a loss.

import { Result } from "../types.ts";

export interface EndBannerProps {
  result: Result;
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
