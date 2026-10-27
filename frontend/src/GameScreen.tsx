// GameScreen — the in-game root. It owns the live WebSocket for the seat via
// useGameSocket (keyed on the session
// token), and once the first PlayerView snapshot arrives it renders the Table
// (four-seat layout + center ValueSummary) with the ActionBar beneath it.
//
// The socket's sendAction is threaded into the ActionBar so the viewer's turn
// controls (guess / safe call / self-reveal) serialize straight onto the wire.
// When a snapshot reports the game has ended, the EndBanner overlays the table
// with the win/loss result.
//
// Until the first snapshot lands it shows a connection status line so the
// create/join -> play transition stays observable. Connection errors surface in
// an alert region without tearing down the table.

import { Table } from "./Table.tsx";
import { ActionBar } from "./ActionBar.tsx";
import { EndBanner } from "./EndBanner.tsx";
import { useGameSocket } from "./useGameSocket.ts";
import { Phase, Result, type SeatID, type SessionToken } from "./types.ts";

export interface GameScreenProps {
  /** The Session_Token binding this browser to its seat. */
  token: SessionToken;
  /** The seat this token was issued for, for display until the first view arrives. */
  seat: SeatID;
  /** Leave the game and return to the JoinScreen (clears the stored session). */
  onLeave: () => void;
}

/**
 * GameScreen owns the live socket for the seat and renders the table once a
 * snapshot is available. The latest PlayerView drives the whole subtree, so
 * every `state` push re-renders the seats, hands, and value summary.
 */
export function GameScreen({ token, seat, onLeave }: GameScreenProps) {
  const { view, error, status, sendAction } = useGameSocket(token);

  const ended = view != null && (view.Phase === Phase.Ended || view.Result !== Result.None);

  return (
    <main className="game-screen">
      <h1>Card Finders</h1>
      {error && (
        <p role="alert">
          {error.code}: {error.message}
        </p>
      )}
      {view ? (
        <>
          <Table view={view} />
          {ended ? (
            <EndBanner result={view.Result} onLeave={onLeave} />
          ) : (
            <ActionBar view={view} sendAction={sendAction} onLeave={onLeave} />
          )}
        </>
      ) : (
        <section aria-label="Connecting">
          <p>Seat {seat}</p>
          <p>Connection: {status}</p>
          <p>Waiting for the first game snapshot…</p>
          <button type="button" onClick={onLeave}>
            Leave game
          </button>
        </section>
      )}
    </main>
  );
}
