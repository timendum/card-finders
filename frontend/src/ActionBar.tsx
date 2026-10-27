// ActionBar — the viewing player's context-sensitive turn controls: a guess
// (silver feat when the source is silver), a safe call against two cards of one
// other player, and a self-reveal of a fully-held value. The whole control set
// is disabled when it is not the viewer's turn; the server remains the
// authority and rejects anything illegal with an `error` frame, so the bar only
// has to offer plausibly legal choices.
//
// Selection logic (which cards/seats/values are offerable, and how each choice
// maps to a ClientMessage) is the pure actions.ts module; this component is a
// thin controlled view over that logic plus React state for the in-progress
// selection. On submit it hands the built message to sendAction (threaded down
// from GameScreen's useGameSocket) and resets the selection.

import { useEffect, useMemo, useState } from "react";
import {
  buildGuess,
  buildSafeCall,
  buildSelfReveal,
  selfRevealOptions,
  sourceCards,
  targetableSeats,
  targetsForSeat,
} from "./actions.ts";
import { cardLabel } from "./cardLabel.ts";
import type {
  CardID,
  ClientMessage,
  PlayerView,
  SeatID,
  SelfRevealValue,
  ViewCard,
} from "./types.ts";

/**
 * Parse a <select> value string back into a self-reveal value: the specials
 * "silver"/"gold" pass through, anything else is a suited rank number. "" (the
 * placeholder option) yields null. The option strings come from
 * selfRevealOptions, so this only ever sees those three shapes.
 */
function parseSelfRevealValue(raw: string): SelfRevealValue | null {
  if (raw === "") return null;
  if (raw === "silver" || raw === "gold") return raw;
  return Number(raw);
}

/** The three turn actions the bar can assemble. */
type Mode = "guess" | "safeCall" | "selfReveal";

export interface ActionBarProps {
  /** The current snapshot, driving turn state and legal actions. */
  view: PlayerView;
  /** Serialize and send a built action over the socket (from useGameSocket). */
  sendAction: (msg: ClientMessage) => boolean;
  /** Leave the game and return to the lobby. */
  onLeave: () => void;
}

/** Label a selectable card: the viewer's own cards always carry a value. */
function optionLabel(card: ViewCard): string {
  return card.Value ? cardLabel(card.Value) : `#${card.ID}`;
}

export function ActionBar({ view, sendAction, onLeave }: ActionBarProps) {
  const yourTurn = view.YourTurn;

  // In-progress selection. State mirrors the game-domain values (card ids, a
  // seat id, a self-reveal value), not the DOM; null means "nothing picked".
  // The string<->value conversion lives only at each <select>'s edge.
  const [mode, setMode] = useState<Mode>("guess");
  const [source, setSource] = useState<CardID | null>(null);
  const [targetSeat, setTargetSeat] = useState<SeatID | null>(null);
  const [target1, setTarget1] = useState<CardID | null>(null);
  const [target2, setTarget2] = useState<CardID | null>(null);
  const [selfValue, setSelfValue] = useState<SelfRevealValue | null>(null);

  // Offerable choices, recomputed from each snapshot.
  const sources = useMemo(() => sourceCards(view), [view]);
  const seats = useMemo(() => targetableSeats(view), [view]);
  const selfOptions = useMemo(() => selfRevealOptions(view), [view]);
  const seatTargets = useMemo(
    () => (targetSeat === null ? [] : targetsForSeat(view, targetSeat)),
    [view, targetSeat],
  );

  const resetSelection = () => {
    setSource(null);
    setTargetSeat(null);
    setTarget1(null);
    setTarget2(null);
    setSelfValue(null);
  };

  // When it stops being the viewer's turn, clear any half-made selection so the
  // bar doesn't resume mid-action on the next turn.
  useEffect(() => {
    if (!yourTurn) resetSelection();
  }, [yourTurn]);

  // A target seat that is no longer targetable (its cards got revealed) is
  // dropped so stale targets can't be submitted.
  useEffect(() => {
    if (targetSeat !== null && !seats.includes(targetSeat)) {
      setTargetSeat(null);
      setTarget1(null);
      setTarget2(null);
    }
  }, [seats, targetSeat]);

  // Whether the current selection is complete enough to send. Final legality
  // is the server's call; this only gates obviously-incomplete submissions.
  const canGuess = source !== null && target1 !== null;
  const canSafeCall =
    source !== null && target1 !== null && target2 !== null && target1 !== target2;
  const canSelfReveal = selfValue !== null;
  const canSubmit =
    yourTurn &&
    ((mode === "guess" && canGuess) ||
      (mode === "safeCall" && canSafeCall) ||
      (mode === "selfReveal" && canSelfReveal));

  const submit = () => {
    if (!canSubmit) return;
    let msg: ClientMessage;
    if (mode === "guess") {
      msg = buildGuess(source!, target1!);
    } else if (mode === "safeCall") {
      msg = buildSafeCall(source!, target1!, target2!);
    } else {
      msg = buildSelfReveal(selfValue!);
    }
    if (sendAction(msg)) {
      resetSelection();
    }
  };

  return (
    <section className="action-bar" aria-label="Actions" data-your-turn={yourTurn}>
      <p className="turn-status">
        {yourTurn ? "Your turn" : `Waiting for seat ${view.CurrentSeat}`}
      </p>

      <fieldset disabled={!yourTurn} className="action-controls">
        <legend className="sr-only">Turn action</legend>

        <nav aria-label="Action type" className="action-modes">
          <button type="button" aria-pressed={mode === "guess"} onClick={() => setMode("guess")}>
            Guess
          </button>
          <button
            type="button"
            aria-pressed={mode === "safeCall"}
            onClick={() => setMode("safeCall")}
          >
            Safe call
          </button>
          <button
            type="button"
            aria-pressed={mode === "selfReveal"}
            onClick={() => setMode("selfReveal")}
          >
            Self-reveal
          </button>
        </nav>

        {(mode === "guess" || mode === "safeCall") && (
          <div className="action-fields">
            <label>
              Source card
              <select
                value={source === null ? "" : String(source)}
                onChange={(e) => setSource(e.target.value === "" ? null : (Number(e.target.value) as CardID))}
              >
                <option value="">Choose a card…</option>
                {sources.map((c) => (
                  <option key={c.ID} value={c.ID}>
                    {optionLabel(c)}
                  </option>
                ))}
              </select>
            </label>

            <label>
              Target player
              <select
                value={targetSeat === null ? "" : String(targetSeat)}
                onChange={(e) => {
                  setTargetSeat(e.target.value === "" ? null : (Number(e.target.value) as SeatID));
                  setTarget1(null);
                  setTarget2(null);
                }}
              >
                <option value="">Choose a player…</option>
                {seats.map((seat) => (
                  <option key={seat} value={seat}>
                    Seat {seat}
                  </option>
                ))}
              </select>
            </label>

            <label>
              {mode === "safeCall" ? "First target" : "Target card"}
              <select
                value={target1 === null ? "" : String(target1)}
                onChange={(e) => setTarget1(e.target.value === "" ? null : (Number(e.target.value) as CardID))}
                disabled={targetSeat === null}
              >
                <option value="">Choose a card…</option>
                {seatTargets.map((c) => (
                  <option key={c.ID} value={c.ID}>
                    #{c.ID}
                  </option>
                ))}
              </select>
            </label>

            {mode === "safeCall" && (
              <label>
                Second target
                <select
                  value={target2 === null ? "" : String(target2)}
                  onChange={(e) => setTarget2(e.target.value === "" ? null : (Number(e.target.value) as CardID))}
                  disabled={targetSeat === null}
                >
                  <option value="">Choose a card…</option>
                  {seatTargets
                    .filter((c) => c.ID !== target1)
                    .map((c) => (
                      <option key={c.ID} value={c.ID}>
                        #{c.ID}
                      </option>
                    ))}
                </select>
              </label>
            )}
          </div>
        )}

        {mode === "selfReveal" && (
          <div className="action-fields">
            <label>
              Value to reveal
              <select
                value={selfValue === null ? "" : String(selfValue)}
                onChange={(e) => setSelfValue(parseSelfRevealValue(e.target.value))}
              >
                <option value="">Choose a value…</option>
                {selfOptions.map((opt) => (
                  <option key={String(opt.value)} value={String(opt.value)}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}

        <button type="button" className="action-submit" onClick={submit} disabled={!canSubmit}>
          Submit action
        </button>
      </fieldset>

      <button type="button" className="leave" onClick={onLeave}>
        Leave game
      </button>
    </section>
  );
}
