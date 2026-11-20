// the end-of-game result overlay shown once the game has ended
// with a win or a loss.

import { Result } from "../types.ts";

export interface EndBannerProps {
  result: Result;
}

// The admin panel opens in a new tab: sessions are per tab (URL hash), so this
// tab keeps its seat for the replay.
export function EndBanner({ result }: EndBannerProps) {
  const won = result === Result.Win;
  return (
    <div
      className="end-banner"
      role="alertdialog"
      aria-label="Game over"
      data-result={won ? "win" : "loss"}
    >
      <h2>{won ? "You win!" : "Game over"}</h2>
      <p>{won ? "Every card is revealed — the team found them all." : "Better luck next time."}</p>
      <button
        type="button"
        className="end-banner-admin"
        onClick={() => window.open("/?admin", "_blank", "noopener")}
      >
        Admin panel
      </button>
    </div>
  );
}
