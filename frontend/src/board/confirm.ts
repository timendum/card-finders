// Pure confirm logic: whether the current selection amounts to a complete move,
// how to label it, and the message that sends it.
import { buildGuess, buildSafeCall, buildSelfReveal } from "./actions.ts";
import { buildJoker, jokerComplete, type JokerPicks } from "./jokers.ts";
import { isResponder, type SelectionState } from "./selection.ts";
import {
  Phase,
  type ClientMessage,
  type PlayerView,
  type SafeChooseRevealMessage,
} from "../types.ts";

export type InferredAction =
  | "openingReveal"
  | "guess"
  | "safeCall"
  | "selfReveal"
  | "safeChooseReveal"
  | "joker";

/** The move the current selection amounts to, or null when it is not yet a complete move. */
export function inferredAction(state: SelectionState, view: PlayerView): InferredAction | null {
  if (isResponder(view)) return state.chosen == null ? null : "safeChooseReveal";
  if (view.Phase === Phase.Opening) return state.source == null ? null : "openingReveal";
  if (view.Phase !== Phase.Playing) return null;
  if (state.jokerRank !== null) {
    return jokerComplete(view, state.jokerRank, jokerPicks(state)) ? "joker" : null;
  }
  if (state.selfValue !== null) return "selfReveal";
  if (state.source === null) return null;
  if (state.targets.length === 1) return "guess";
  if (state.targets.length === 2) return "safeCall";
  return null;
}

const confirmLabels: Record<InferredAction, string> = {
  openingReveal: "Confirm reveal",
  guess: "Confirm guess",
  safeCall: "Confirm double call",
  selfReveal: "Confirm self-reveal",
  safeChooseReveal: "Confirm choice",
  joker: "Confirm joker",
};

/** The picks of the joker flow in progress, in the shape the joker helpers take. */
export function jokerPicks(state: SelectionState): JokerPicks {
  return {
    own: state.jokerOwn,
    targets: state.targets,
    seats: state.jokerSeats,
    rank: state.jokerAsk,
  };
}

/** The confirm message for the picked card, or null until one is chosen. */
export function buildChoice(state: SelectionState): SafeChooseRevealMessage | null {
  return state.chosen === null ? null : { type: "safeChooseReveal", targetCardIds: [state.chosen] };
}

/** The single message the current selection confirms, or null if incomplete. */
export function messageFor(selection: SelectionState, view: PlayerView): ClientMessage | null {
  const { source, targets, selfValue } = selection;
  switch (inferredAction(selection, view)) {
    case "openingReveal":
      return source === null ? null : { type: "openingReveal", targetCardIds: [source] };
    case "guess":
      return source === null ? null : buildGuess(source, targets[0]);
    case "safeCall":
      return source === null ? null : buildSafeCall(source, targets[0], targets[1]);
    case "selfReveal":
      return selfValue === null ? null : buildSelfReveal(selfValue);
    case "safeChooseReveal":
      return buildChoice(selection);
    case "joker":
      return selection.jokerRank === null
        ? null
        : buildJoker(view, selection.jokerRank, jokerPicks(selection));
    default:
      return null;
  }
}

/** Label for the confirm control; a neutral label while nothing is confirmable. */
export function confirmLabel(state: SelectionState, view: PlayerView): string {
  const action = inferredAction(state, view);
  return action === null ? "Confirm" : confirmLabels[action];
}

/** Confirm is enabled only for a complete selection (source plus target(s) for calls). */
export function canConfirm(state: SelectionState, view: PlayerView): boolean {
  return inferredAction(state, view) !== null;
}
