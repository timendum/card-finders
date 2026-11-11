import { useCallback, useState } from "react";
import { Table } from "../board/Table.tsx";
import { StatusPanel } from "../status/StatusPanel.tsx";
import { PromptPanel } from "../board/PromptPanel.tsx";
import { GameLog } from "../board/GameLog.tsx";
import { EndBanner } from "../status/EndBanner.tsx";
import { useGameSocket } from "../api/useGameSocket.ts";
import { buildGuess, buildSafeCall, buildSelfReveal } from "../board/actions.ts";
import { matchHints } from "../board/matchHints.ts";
import {
  buildChoice,
  emptySelection,
  inferredAction,
  isSelectable,
  selectionReducer,
  type SelectionEvent,
  type SelectionState,
} from "../board/selection.ts";
import {
  Phase,
  Result,
  type CardID,
  type ClientMessage,
  type PlayerView,
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

/** The single message the current selection confirms, or null if incomplete. */
export function messageFor(selection: SelectionState, view: PlayerView): ClientMessage | null {
  const { source, targets, selfValue } = selection;
  switch (inferredAction(selection, view)) {
    case "openingReveal":
      return source === null ? null : { type: "openingReveal", cardId: source };
    case "guess":
      return source === null ? null : buildGuess(source, targets[0]);
    case "safeCall":
      return source === null ? null : buildSafeCall(source, targets[0], targets[1]);
    case "selfReveal":
      return selfValue === null ? null : buildSelfReveal(selfValue);
    case "safeChooseReveal":
      return buildChoice(selection);
    default:
      return null;
  }
}

export function GameScreen({ token, seat, onLeave }: GameScreenProps) {
  const { view, error, status, sendAction } = useGameSocket(token);
  const [selection, setSelection] = useState<SelectionState>(emptySelection);

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
    selection.targets.includes(card.ID);

  const hints = ended ? null : matchHints(view, selection.source);

  return (
    <main className="game-screen">
      <Table
        view={view}
        status={<StatusPanel view={view} />}
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
              onDismissResult={(seq) => dispatch({ type: "shown", seq })}
            />
          )
        }
        log={<GameLog view={view} />}
        isSelectable={ended ? undefined : (card) => isSelectable(view, card)}
        isSelected={selected}
        matchHint={hints ? (card) => hints.get(card.ID) : undefined}
        onActivate={ended ? undefined : onActivate}
      />
    </main>
  );
}
