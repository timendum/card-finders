import { useCallback, useEffect, useState } from "react";
import { useGameSocket } from "../api/useGameSocket.ts";
import {
  GameLog,
  PromptPanel,
  Table,
  emptySelection,
  isSelectable,
  matchHints,
  messageFor,
  selectionReducer,
  type SelectionEvent,
  type SelectionState,
} from "../board/index.ts";
import { EndBanner, StatusPanel } from "../status/index.ts";
import {
  Phase,
  Result,
  type CardID,
  type SeatID,
  type SessionToken,
  type ViewCard,
} from "../types.ts";

export interface GameScreenProps {
  token: SessionToken;
  /** Shown until the first view arrives; thereafter the view supplies the seat. */
  seat: SeatID;
  onLeave: () => void;
}

export function GameScreen({ token, seat, onLeave }: GameScreenProps) {
  const { view, error, status, sendAction } = useGameSocket(token);
  const [selection, setSelection] = useState<SelectionState>(emptySelection);

  // The session is dead (seat taken over, lobby seat released, server
  // restarted): retrying cannot help, so go back to the join screen.
  useEffect(() => {
    if (status === "rejected") onLeave();
  }, [status, onLeave]);

  // A new snapshot means the last move was accepted (or state moved on), so
  // any half-built selection is obsolete. Reset during render (React's
  // reset-on-prop-change pattern) instead of in an effect. An error frame
  // leaves the view untouched, so the selection survives it for a retry.
  const [prevView, setPrevView] = useState(view);
  if (prevView !== view) {
    setPrevView(view);
    setSelection((s) => ({ ...emptySelection, lastShownSeq: s.lastShownSeq }));
  }

  const dispatch = useCallback(
    (event: SelectionEvent) => {
      if (!view) return;
      setSelection((s) => selectionReducer(s, view, event));
    },
    [view],
  );

  const onActivate = useCallback(
    (id: CardID) => dispatch({ type: "clickCard", cardId: id }),
    [dispatch],
  );

  const onConfirm = useCallback(() => {
    if (!view) return;
    const msg = messageFor(selection, view);
    if (msg) sendAction(msg);
  }, [view, selection, sendAction]);

  if (!view) {
    return (
      <main className="game-screen">
        <h1>Card Finders</h1>
        <section aria-label="Connecting">
          <p>Seat {seat}</p>
          <p>Connection: {status}</p>
          <p>Waiting for the first game snapshot…</p>
          <button type="button" onClick={onLeave}>
            Leave game
          </button>
        </section>
      </main>
    );
  }

  if (view.Phase === Phase.Waiting) {
    return (
      <main className="game-screen">
        <h1>Card Finders</h1>
        <section aria-label="Lobby">
          <p>
            Waiting for players: {view.SeatsFilled}/4 seated. You are seat {view.Seat}.
          </p>
          <button type="button" onClick={onLeave}>
            Leave game
          </button>
        </section>
      </main>
    );
  }

  const ended = view.Phase === Phase.Ended || view.Result !== Result.None;
  const selected = (card: ViewCard) =>
    card.ID === selection.source ||
    card.ID === selection.chosen ||
    selection.jokerOwn.includes(card.ID) ||
    selection.targets.includes(card.ID);

  const hints = ended || selection.jokerRank !== null ? null : matchHints(view, selection.source);

  return (
    <main className="game-screen">
      {status !== "open" && (
        <p role="status" className="connection-banner">
          Connection lost, reconnecting…
        </p>
      )}
      <Table
        view={view}
        status={
          <StatusPanel
            view={view}
            activeJoker={ended ? null : selection.jokerRank}
            onActivateJoker={ended ? undefined : (rank) => dispatch({ type: "pickJoker", rank })}
          />
        }
        prompt={
          ended ? (
            <EndBanner result={view.Result} onLeave={onLeave} />
          ) : (
            <PromptPanel
              view={view}
              selection={selection}
              error={error}
              onConfirm={onConfirm}
              onPickSelfValue={(value) => dispatch({ type: "pickSelfValue", value })}
              onPickJokerSeat={(seat) => dispatch({ type: "pickJokerSeat", seat })}
              onPickJokerAsk={(rank) => dispatch({ type: "pickJokerAsk", rank })}
              onDismissResult={(seq) => dispatch({ type: "shown", seq })}
            />
          )
        }
        log={<GameLog view={view} />}
        isSelectable={ended ? undefined : (card) => isSelectable(view, card, selection)}
        isSelected={selected}
        matchHint={hints ? (card) => hints.get(card.ID) : undefined}
        onActivate={ended ? undefined : onActivate}
      />
    </main>
  );
}
