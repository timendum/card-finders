// Pure click-selection reducer. A click only edits the selection; sending a
// message is the caller's job after an explicit confirm.

import { sourceCards, targetCards } from "./actions.ts";
import {
  Phase,
  type CardID,
  type PlayerView,
  type SeatID,
  type SafeChooseRevealMessage,
  type SelfRevealValue,
  type ViewCard,
} from "../types.ts";

export interface SelectionState {
  source: CardID | null;
  targetSeat: SeatID | null;
  targets: CardID[];
  selfValue: SelfRevealValue | null;
  chosen: CardID | null;
  lastShownSeq: number | null;
}

export type InferredAction =
  | "openingReveal"
  | "guess"
  | "safeCall"
  | "selfReveal"
  | "safeChooseReveal";

export const emptySelection: SelectionState = {
  source: null,
  targetSeat: null,
  targets: [],
  selfValue: null,
  chosen: null,
  lastShownSeq: null,
};

export type SelectionEvent =
  | { type: "clickCard"; cardId: CardID }
  | { type: "pickSelfValue"; value: SelfRevealValue | null }
  | { type: "clear" }
  | { type: "shown"; seq: number | null };

function isResponder(view: PlayerView): boolean {
  return (
    view.Phase === Phase.AwaitingResponse &&
    view.HasResponsePlayer &&
    view.ResponsePlayer === view.Seat
  );
}

/** Cards offered for the safe-choose-reveal pick; empty unless the viewer is the responder. */
export function chooseOptions(view: PlayerView): CardID[] {
  return isResponder(view) ? (view.LastAction?.Targets ?? []) : [];
}

/** Whether the viewer may click this card now; drives the selectable styling. */
export function isSelectable(view: PlayerView, card: ViewCard): boolean {
  if (view.Phase === Phase.AwaitingResponse) return chooseOptions(view).includes(card.ID);
  const acting = view.Phase === Phase.Opening || (view.Phase === Phase.Playing && view.YourTurn);
  if (!acting) return false;
  if (sourceCards(view).some((c) => c.ID === card.ID)) return true;
  return view.Phase === Phase.Playing && targetCards(view).some((c) => c.ID === card.ID);
}

/** The confirm message for the picked card, or null until one is chosen. */
export function buildChoice(state: SelectionState): SafeChooseRevealMessage | null {
  return state.chosen === null
    ? null
    : { type: "safeChooseReveal", chosenTargetCardId: state.chosen };
}

// A card click clears any pending self-reveal value: the two intents are exclusive.
function clickCard(state: SelectionState, view: PlayerView, cardId: CardID): SelectionState {
  if (isResponder(view)) {
    if (!chooseOptions(view).includes(cardId)) return state;
    return { ...state, chosen: state.chosen === cardId ? null : cardId };
  }

  const acting = view.Phase === Phase.Opening || (view.Phase === Phase.Playing && view.YourTurn);
  if (!acting) return state;

  if (sourceCards(view).some((c) => c.ID === cardId)) {
    return {
      ...state,
      selfValue: null,
      source: state.source === cardId ? null : cardId,
    };
  }

  // The opening reveal takes only an own card.
  if (view.Phase !== Phase.Playing) return state;

  const card = targetCards(view).find((c) => c.ID === cardId);
  if (!card) return state;

  if (state.targets.includes(cardId)) {
    const targets = state.targets.filter((id) => id !== cardId);
    return {
      ...state,
      selfValue: null,
      targets,
      targetSeat: targets.length > 0 ? state.targetSeat : null,
    };
  }

  const powerUnused = !view.PowerUsed[view.Seat];
  const joinsExisting =
    state.targetSeat === card.Owner && state.targets.length === 1 && powerUnused;
  return {
    ...state,
    selfValue: null,
    targetSeat: card.Owner,
    targets: joinsExisting ? [...state.targets, cardId] : [cardId],
  };
}

export function selectionReducer(
  state: SelectionState,
  view: PlayerView,
  event: SelectionEvent,
): SelectionState {
  switch (event.type) {
    case "clickCard":
      return clickCard(state, view, event.cardId);
    case "pickSelfValue":
      return event.value === null
        ? { ...state, selfValue: null }
        : { ...state, selfValue: event.value, source: null, targetSeat: null, targets: [] };
    case "clear":
      return { ...emptySelection, lastShownSeq: state.lastShownSeq };
    case "shown":
      return { ...state, lastShownSeq: event.seq };
  }
}

/** The move the current selection amounts to, or null when it is not yet a complete move. */
export function inferredAction(state: SelectionState, view: PlayerView): InferredAction | null {
  if (isResponder(view)) return state.chosen == null ? null : "safeChooseReveal";
  if (view.Phase === Phase.Opening) return state.source == null ? null : "openingReveal";
  if (view.Phase !== Phase.Playing) return null;
  if (state.selfValue !== null) return "selfReveal";
  if (state.source === null) return null;
  if (state.targets.length === 1) return "guess";
  if (state.targets.length === 2) return "safeCall";
  return null;
}

const confirmLabels: Record<InferredAction, string> = {
  openingReveal: "Confirm reveal",
  guess: "Confirm guess",
  safeCall: "Confirm safe call",
  selfReveal: "Confirm self-reveal",
  safeChooseReveal: "Confirm choice",
};

/** Label for the confirm control; a neutral label while nothing is confirmable. */
export function confirmLabel(state: SelectionState, view: PlayerView): string {
  const action = inferredAction(state, view);
  return action === null ? "Confirm" : confirmLabels[action];
}

/** Confirm is enabled only for a complete selection (source plus target(s) for calls). */
export function canConfirm(state: SelectionState, view: PlayerView): boolean {
  return inferredAction(state, view) !== null;
}
