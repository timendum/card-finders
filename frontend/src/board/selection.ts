// Pure click-selection reducer. A click only edits the selection; sending a
// message is the caller's job after an explicit confirm.

import { sourceCards, targetCards } from "./actions.ts";
import {
  isSwapChoice,
  jokerComplete,
  jokerOwnCards,
  jokerShape,
  jokerTargetCards,
  type JokerPicks,
  jokerUsable,
} from "./jokers.ts";
import {
  Face,
  Phase,
  SpecialKind,
  type CardID,
  type PlayerView,
  type Rank,
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
  /** The joker being set up, or null outside a joker flow. */
  jokerRank: Rank | null;
  /** The joker's own-card picks (one or two). */
  jokerOwn: CardID[];
  /** The joker's seat picks (one or two). */
  jokerSeats: SeatID[];
  /** The rank a Census asks about. */
  jokerAsk: number | null;
}

export type InferredAction =
  | "openingReveal"
  | "guess"
  | "safeCall"
  | "selfReveal"
  | "safeChooseReveal"
  | "joker";

export const emptySelection: SelectionState = {
  source: null,
  targetSeat: null,
  targets: [],
  selfValue: null,
  chosen: null,
  lastShownSeq: null,
  jokerRank: null,
  jokerOwn: [],
  jokerSeats: [],
  jokerAsk: null,
};

export type SelectionEvent =
  | { type: "clickCard"; cardId: CardID }
  | { type: "pickSelfValue"; value: SelfRevealValue | null }
  | { type: "pickJoker"; rank: Rank | null }
  | { type: "pickJokerSeat"; seat: SeatID }
  | { type: "pickJokerAsk"; rank: number | null }
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
  if (!isResponder(view)) return [];
  if (isSwapChoice(view)) {
    return view.Cards.filter((c) => c.Owner === view.Seat && c.Facing === Face.Down).map(
      (c) => c.ID,
    );
  }
  return view.LastAction?.Targets ?? [];
}

/** Cards a joker flow can click right now (own and target candidates). */
function jokerClickable(view: PlayerView, rank: Rank): ViewCard[] {
  return [...jokerOwnCards(view, rank), ...jokerTargetCards(view, rank)];
}

/** Source candidates; in Opening, silver and gold cards are excluded. */
function openingSourceCards(view: PlayerView): ViewCard[] {
  const cards = sourceCards(view);
  if (view.Phase !== Phase.Opening) return cards;
  return cards.filter((c) => c.Value == null || c.Value.Kind === SpecialKind.None);
}

/** Whether the viewer may click this card now; drives the selectable styling. */
export function isSelectable(
  view: PlayerView,
  card: ViewCard,
  selection?: SelectionState,
): boolean {
  if (view.Phase === Phase.AwaitingResponse) return chooseOptions(view).includes(card.ID);
  if (selection?.jokerRank != null) {
    return jokerClickable(view, selection.jokerRank).some((c) => c.ID === card.ID);
  }
  const acting = view.Phase === Phase.Opening || (view.Phase === Phase.Playing && view.YourTurn);
  if (!acting) return false;
  if (openingSourceCards(view).some((c) => c.ID === card.ID)) return true;
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

  if (state.jokerRank !== null) return clickJokerCard(state, view, cardId);

  const acting = view.Phase === Phase.Opening || (view.Phase === Phase.Playing && view.YourTurn);
  if (!acting) return state;

  if (openingSourceCards(view).some((c) => c.ID === cardId)) {
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

// A click while a joker is being set up: toggle an own pick (up to the joker's
// own-card count) or a target pick (all from one seat, up to its count).
function clickJokerCard(state: SelectionState, view: PlayerView, cardId: CardID): SelectionState {
  const rank = state.jokerRank;
  if (rank === null) return state;
  const shape = jokerShape(view, rank);

  if (jokerOwnCards(view, rank).some((c) => c.ID === cardId)) {
    if (state.jokerOwn.includes(cardId)) {
      return { ...state, jokerOwn: state.jokerOwn.filter((id) => id !== cardId) };
    }
    if (shape.own === 1) return { ...state, jokerOwn: [cardId] };
    if (state.jokerOwn.length >= shape.own) return state;
    return { ...state, jokerOwn: [...state.jokerOwn, cardId] };
  }

  const target = jokerTargetCards(view, rank).find((c) => c.ID === cardId);
  if (!target) return state;
  if (state.targets.includes(cardId)) {
    return { ...state, targets: state.targets.filter((id) => id !== cardId) };
  }
  const sameSeat = state.targets.every(
    (id) => view.Cards.find((c) => c.ID === id)?.Owner === target.Owner,
  );
  if (shape.targets === 1 || !sameSeat) return { ...state, targets: [cardId] };
  if (state.targets.length >= shape.targets) return state;
  return { ...state, targets: [...state.targets, cardId] };
}

function pickJoker(state: SelectionState, view: PlayerView, rank: Rank | null): SelectionState {
  const fresh = { ...emptySelection, lastShownSeq: state.lastShownSeq };
  if (rank === null || rank === state.jokerRank) return fresh;
  return jokerUsable(view, rank) ? { ...fresh, jokerRank: rank } : state;
}

function pickJokerSeat(state: SelectionState, view: PlayerView, seat: SeatID): SelectionState {
  if (state.jokerRank === null) return state;
  const need = jokerShape(view, state.jokerRank).seats;
  if (state.jokerSeats.includes(seat)) {
    return { ...state, jokerSeats: state.jokerSeats.filter((s) => s !== seat) };
  }
  if (need === 1) return { ...state, jokerSeats: [seat] };
  if (state.jokerSeats.length >= need) return state;
  return { ...state, jokerSeats: [...state.jokerSeats, seat] };
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
        : {
            ...emptySelection,
            lastShownSeq: state.lastShownSeq,
            selfValue: event.value,
          };
    case "pickJoker":
      return pickJoker(state, view, event.rank);
    case "pickJokerSeat":
      return pickJokerSeat(state, view, event.seat);
    case "pickJokerAsk":
      return state.jokerRank === null ? state : { ...state, jokerAsk: event.rank };
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

/** Label for the confirm control; a neutral label while nothing is confirmable. */
export function confirmLabel(state: SelectionState, view: PlayerView): string {
  const action = inferredAction(state, view);
  return action === null ? "Confirm" : confirmLabels[action];
}

/** Confirm is enabled only for a complete selection (source plus target(s) for calls). */
export function canConfirm(state: SelectionState, view: PlayerView): boolean {
  return inferredAction(state, view) !== null;
}
