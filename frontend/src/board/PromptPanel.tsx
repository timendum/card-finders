// PromptPanel — bottom-center panel: states what the viewer can do, hosts the
// single confirm control, and shows the latest server error. It is purely
// presentational; the owner holds the selection and performs the send.

import { selfRevealOptions } from "./actions.ts";
import { availableActions } from "./availableActions.ts";
import { rankLabel } from "./cardLabel.ts";
import {
  isSwapChoice,
  jokerDescriptions,
  jokerLabel,
  jokerSeatChoices,
  jokerShape,
} from "./jokers.ts";
import { seatPositionName } from "./seatLayout.ts";
import { canConfirm, confirmLabel } from "./confirm.ts";
import type { SelectionState } from "./selection.ts";
import type { SocketError } from "../api/useGameSocket.ts";
import {
  ActionKind,
  Outcome,
  type LastAction,
  type PlayerView,
  type SeatID,
  type SelfRevealValue,
} from "../types.ts";

export interface PromptPanelProps {
  view: PlayerView;
  selection: SelectionState;
  /** Latest `error` frame; the selection is left untouched so it can be retried. */
  error: SocketError | null;
  onConfirm: () => void;
  onPickSelfValue: (value: SelfRevealValue | null) => void;
  /** Joker flow controls; optional so a panel without jokers needs no wiring. */
  onPickJokerSeat?: (seat: SeatID) => void;
  onPickJokerAsk?: (rank: number | null) => void;
  /** Dismiss the result banner; receives its Seq so the owner can remember it. */
  onDismissResult?: (seq: number) => void;
}

/**
 * The viewer's own resolved guess/safe call that has not been dismissed.
 * A newer Seq supersedes it naturally, and a fresh mount (reconnect) with no
 * dismissal bookkeeping shows the latest one again.
 */
export function visibleResult(view: PlayerView, lastShownSeq: number | null): LastAction | null {
  const last = view.LastAction;
  if (!last || last.Actor !== view.Seat) return null;
  if (last.Kind !== ActionKind.Guess && last.Kind !== ActionKind.SafeCall) return null;
  if (
    last.Outcome !== Outcome.Hit &&
    last.Outcome !== Outcome.Miss &&
    last.Outcome !== Outcome.GoldHit
  ) {
    return null;
  }
  if (lastShownSeq !== null && last.Seq <= lastShownSeq) return null;
  return last;
}

function resultText(last: LastAction): string {
  if (last.Outcome === Outcome.Hit) return "Hit! No life lost.";
  if (last.Outcome === Outcome.GoldHit) return "You hit a gold card. Game over.";
  const n = last.LivesLost;
  return `Miss. You lost ${n} ${n === 1 ? "life" : "lives"}.`;
}

function notActingText(view: PlayerView, actingSeat: number, awaitingChoice: boolean): string {
  const pos = seatPositionName(view.Seat, actingSeat);
  const who = pos === "you" ? "You are" : `The ${pos} player is`;
  const verb = isSwapChoice(view)
    ? "give"
    : view.LastAction?.Outcome === Outcome.AwaitingToken
      ? "token"
      : "reveal";
  return awaitingChoice
    ? `${who} choosing a card to ${verb}.`
    : `Waiting for the ${pos} player to act.`;
}

export function PromptPanel({
  view,
  selection,
  error,
  onConfirm,
  onPickSelfValue,
  onPickJokerSeat,
  onPickJokerAsk,
  onDismissResult,
}: PromptPanelProps) {
  const result = visibleResult(view, selection.lastShownSeq);
  const prompt = availableActions(view);
  const single = prompt.actions.length === 1 ? prompt.actions[0] : null;
  const jokerRank = selection.jokerRank;
  const showConfirm = prompt.variant !== "notActing" || jokerRank !== null;
  const enabled = canConfirm(selection, view);
  const selfOptions = prompt.variant === "standardTurn" ? selfRevealOptions(view) : [];

  return (
    <section className="prompt-panel" aria-label="Your move">
      {prompt.variant === "notActing" && (
        <p className="prompt-waiting">
          {notActingText(
            view,
            prompt.actingSeat ?? view.CurrentSeat,
            prompt.awaitingChoice === true,
          )}
        </p>
      )}

      {single && (
        <div className="prompt-single">
          <div className="prompt-action-name">{single.name}</div>
          <div className="prompt-action-description">{single.description}</div>
        </div>
      )}

      {!single && prompt.actions.length > 0 && (
        <ul className="prompt-actions">
          {prompt.actions
            .filter((action) => action.enabled)
            .map((action) => (
              <li key={action.id} className="prompt-action">
                <span className="prompt-action-name">{action.name}</span>
                <span className="prompt-action-description">{action.description}</span>
              </li>
            ))}
        </ul>
      )}

      {jokerRank !== null && (
        <div className="prompt-joker" role="group" aria-label={`Joker: ${jokerLabel(jokerRank)}`}>
          <div className="prompt-action-name">{jokerLabel(jokerRank)}</div>
          <div className="prompt-action-description">{jokerDescriptions[jokerRank]}</div>
          {jokerShape(view, jokerRank).seats > 0 && (
            <div className="prompt-joker-seats" role="group" aria-label="Player">
              {jokerSeatChoices(view, jokerRank).map((seat) => (
                <button
                  key={seat}
                  type="button"
                  className="prompt-joker-seat"
                  aria-pressed={selection.jokerSeats.includes(seat)}
                  onClick={() => onPickJokerSeat?.(seat)}
                >
                  {seat === view.Seat ? "You" : `The ${seatPositionName(view.Seat, seat)} player`}
                </button>
              ))}
            </div>
          )}
          {jokerShape(view, jokerRank).rank && (
            <div className="prompt-joker-ranks" role="group" aria-label="Rank">
              {Array.from({ length: 12 }, (_, i) => i + 2).map((rank) => (
                <button
                  key={rank}
                  type="button"
                  className="prompt-joker-rank"
                  aria-pressed={selection.jokerAsk === rank}
                  onClick={() => onPickJokerAsk?.(selection.jokerAsk === rank ? null : rank)}
                >
                  {rankLabel(rank)}
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {selfOptions.length > 0 && jokerRank === null && (
        <div className="prompt-self-values" role="group" aria-label="Self-reveal value">
          {selfOptions.map((option) => {
            const selected = selection.selfValue === option.value;
            return (
              <button
                key={String(option.value)}
                type="button"
                className="prompt-self-value"
                aria-pressed={selected}
                onClick={() => onPickSelfValue(selected ? null : option.value)}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      )}

      {showConfirm && (
        <button type="button" className="prompt-confirm" disabled={!enabled} onClick={onConfirm}>
          {confirmLabel(selection, view)}
        </button>
      )}

      {result && (
        <div
          className="prompt-result"
          data-outcome={result.Outcome === Outcome.Hit ? "hit" : "miss"}
          role="status"
        >
          <span className="prompt-result-text">{resultText(result)}</span>
          <button
            type="button"
            className="prompt-result-dismiss"
            onClick={() => onDismissResult?.(result.Seq)}
          >
            Dismiss
          </button>
        </div>
      )}

      {error && (
        <p className="prompt-error" role="alert">
          {error.message}
        </p>
      )}
    </section>
  );
}
