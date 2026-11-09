// PromptPanel — bottom-center panel: states what the viewer can do, hosts the
// single confirm control, and shows the latest server error. It is purely
// presentational; the owner holds the selection and performs the send.

import { selfRevealOptions } from "./actions.ts";
import { availableActions } from "./availableActions.ts";
import { canConfirm, confirmLabel, type SelectionState } from "./selection.ts";
import type { SocketError } from "../api/useGameSocket.ts";
import {
  ActionKind,
  Outcome,
  type LastAction,
  type PlayerView,
  type SelfRevealValue,
} from "../types.ts";

export interface PromptPanelProps {
  view: PlayerView;
  selection: SelectionState;
  /** Latest `error` frame; the selection is left untouched so it can be retried. */
  error: SocketError | null;
  onConfirm: () => void;
  onPickSelfValue: (value: SelfRevealValue | null) => void;
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
  if (last.Outcome !== Outcome.Hit && last.Outcome !== Outcome.Miss) return null;
  if (lastShownSeq !== null && last.Seq <= lastShownSeq) return null;
  return last;
}

function resultText(last: LastAction): string {
  if (last.Outcome === Outcome.Hit) return "Hit! No life lost.";
  const n = last.LivesLost;
  return `Miss. You lost ${n} ${n === 1 ? "life" : "lives"}.`;
}

function notActingText(view: PlayerView, actingSeat: number, awaitingChoice: boolean): string {
  const who = actingSeat === view.Seat ? "You are" : `Seat ${actingSeat} is`;
  return awaitingChoice
    ? `${who} choosing a card to reveal.`
    : `Waiting for seat ${actingSeat} to act.`;
}

export function PromptPanel({
  view,
  selection,
  error,
  onConfirm,
  onPickSelfValue,
  onDismissResult,
}: PromptPanelProps) {
  const result = visibleResult(view, selection.lastShownSeq);
  const prompt = availableActions(view);
  const single = prompt.actions.length === 1 ? prompt.actions[0] : null;
  const showConfirm = prompt.variant !== "notActing";
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
          <p className="prompt-action-name">{single.name}</p>
          <p className="prompt-action-description">{single.description}</p>
        </div>
      )}

      {!single && prompt.actions.length > 0 && (
        <ul className="prompt-actions">
          {prompt.actions.map((action) => (
            <li
              key={action.id}
              className="prompt-action"
              data-enabled={action.enabled}
              aria-disabled={!action.enabled}
            >
              <span className="prompt-action-name">{action.name}</span>
              {action.enabled || !action.disabledReason ? (
                <span className="prompt-action-description">{action.description}</span>
              ) : (
                <span className="prompt-action-reason">{action.disabledReason}</span>
              )}
            </li>
          ))}
        </ul>
      )}

      {selfOptions.length > 0 && (
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
