import { Table } from "./Table.tsx";
import { ActionBar } from "./ActionBar.tsx";
import { EndBanner } from "./EndBanner.tsx";
import { useGameSocket } from "./useGameSocket.ts";
import { Phase, Result, type SeatID, type SessionToken } from "./types.ts";

export interface GameScreenProps {
  token: SessionToken;
  /** Shown until the first view arrives; thereafter the view supplies the seat. */
  seat: SeatID;
  onLeave: () => void;
}

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
