// ActionBar — the viewer's turn controls for the three turn actions. Offerable
// choices and message construction live in actions.ts; this is a controlled
// view over that logic plus React state for the in-progress selection.

import { useMemo, useState } from "react";
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

// Placeholder "" yields null; "silver"/"gold" pass through; else a suited rank.
function parseSelfRevealValue(raw: string): SelfRevealValue | null {
  if (raw === "") return null;
  if (raw === "silver" || raw === "gold") return raw;
  return Number(raw);
}

type Mode = "guess" | "safeCall" | "selfReveal";

export interface ActionBarProps {
  view: PlayerView;
  sendAction: (msg: ClientMessage) => boolean;
  onLeave: () => void;
}

function optionLabel(card: ViewCard): string {
  return card.Value ? cardLabel(card.Value) : `#${card.ID}`;
}

export function ActionBar({ view, sendAction, onLeave }: ActionBarProps) {
  const yourTurn = view.YourTurn;

  // Selection holds domain values (null = unset); string conversion stays at
  // each <select>'s edge.
  const [mode, setMode] = useState<Mode>("guess");
  const [source, setSource] = useState<CardID | null>(null);
  const [targetSeat, setTargetSeat] = useState<SeatID | null>(null);
  const [target1, setTarget1] = useState<CardID | null>(null);
  const [target2, setTarget2] = useState<CardID | null>(null);
  const [selfValue, setSelfValue] = useState<SelfRevealValue | null>(null);

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

  // Both adjustments below run during render (adjust-state-on-prop-change)
  // rather than in an effect, to avoid a cascading extra commit.

  // Clear a half-made selection when the turn ends.
  const [prevYourTurn, setPrevYourTurn] = useState(yourTurn);
  if (prevYourTurn !== yourTurn) {
    setPrevYourTurn(yourTurn);
    if (!yourTurn) resetSelection();
  }

  // Drop a target seat that is no longer targetable.
  if (targetSeat !== null && !seats.includes(targetSeat)) {
    setTargetSeat(null);
    setTarget1(null);
    setTarget2(null);
  }

  // Gate only obviously-incomplete submissions; the server decides legality.
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
