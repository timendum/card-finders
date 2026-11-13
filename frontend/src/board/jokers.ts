// Pure joker logic: display names (the backend speaks only numeric ranks), the
// anytime / current-player split, usability, the selection shape each joker
// needs, and the wire message a complete selection produces. The engine stays
// the authority and rejects anything illegal.

import { sourceCards, targetCards } from "./actions.ts";
import {
  Face,
  JokerCensus,
  JokerDiffer,
  JokerExtraLife,
  JokerMatch,
  JokerPass,
  JokerReverseCall,
  JokerSecondWind,
  JokerSelfMark,
  JokerShield,
  JokerState,
  JokerSwap,
  JokerSweep,
  JokerTripleCall,
  Outcome,
  Phase,
  SpecialKind,
  type JokerMessage,
  type PlayerView,
  type Rank,
  type SeatID,
  type ViewCard,
  type CardID,
} from "../types.ts";

export const jokerNames: Readonly<Record<Rank, string>> = {
  [JokerDiffer]: "Differ",
  [JokerSwap]: "Swap",
  [JokerTripleCall]: "Triple call",
  [JokerSelfMark]: "Self mark",
  [JokerSweep]: "Sweep",
  [JokerExtraLife]: "Extra life",
  [JokerSecondWind]: "Second wind",
  [JokerCensus]: "Census",
  [JokerShield]: "Shield",
  [JokerReverseCall]: "Reverse call",
  [JokerPass]: "Pass",
  [JokerMatch]: "Match",
};

export const jokerDescriptions: Readonly<Record<Rank, string>> = {
  [JokerDiffer]: "Pick two adjacent face-down cards of yours that have different ranks.",
  [JokerSwap]: "Pick one of your cards and a player; they give you one of theirs back.",
  [JokerTripleCall]: "Pick one of your cards, then three face-down cards of one opponent.",
  [JokerSelfMark]: "Pick one of your face-down cards to token.",
  [JokerSweep]:
    "Pick one of your suited cards, then a player: it hits every matching card they hold.",
  [JokerExtraLife]: "Gain one life.",
  [JokerSecondWind]: "Restore the double call of players who used it.",
  [JokerCensus]: "Ask every player whether they hold a rank face down.",
  [JokerShield]: "Your next guess costs no life and cannot end the game.",
  [JokerReverseCall]: "Pick two of your cards with different values, then one opponent card.",
  [JokerPass]: "Skip your turn and choose who plays next.",
  [JokerMatch]: "Pick two adjacent face-down cards of yours that have the same rank.",
};

/** Jokers any player may use at any time; the rest need the current turn. */
const anytime: ReadonlySet<Rank> = new Set([
  JokerDiffer,
  JokerSwap,
  JokerSelfMark,
  JokerExtraLife,
  JokerSecondWind,
  JokerCensus,
  JokerMatch,
]);

export function isAnytime(rank: Rank): boolean {
  return anytime.has(rank);
}

export function jokerLabel(rank: Rank): string {
  return jokerNames[rank] ?? `Joker ${rank}`;
}

export function jokerStateOf(view: PlayerView, rank: Rank): JokerState {
  return view.Jokers.find((j) => j.Rank === rank)?.State ?? JokerState.Inactive;
}

/** Ready, in Playing (never while a choice is pending), and the turn rule met. */
export function jokerUsable(view: PlayerView, rank: Rank): boolean {
  if (jokerStateOf(view, rank) !== JokerState.Ready) return false;
  if (view.Phase !== Phase.Playing) return false;
  return isAnytime(rank) || view.YourTurn;
}

/**
 * A Swap awaiting the target's reply: the responder picks any of its own
 * face-down cards (the choice carries no listed targets).
 */
export function isSwapChoice(view: PlayerView): boolean {
  return (
    view.Phase === Phase.AwaitingResponse &&
    view.LastAction?.Joker === JokerSwap &&
    view.LastAction.Outcome === Outcome.AwaitingChoice
  );
}

/** What a joker needs before it can be confirmed. */
export interface JokerShape {
  /** Own cards to pick. */
  own: 0 | 1 | 2;
  /** Other-player cards to pick. */
  targets: 0 | 1 | 3;
  /** Seats to pick; 8 decides this from the view, see seatsNeeded. */
  seats: 0 | 1 | 2;
  /** A rank to name. */
  rank: boolean;
}

const shapes: Readonly<Record<Rank, JokerShape>> = {
  [JokerDiffer]: { own: 2, targets: 0, seats: 0, rank: false },
  [JokerMatch]: { own: 2, targets: 0, seats: 0, rank: false },
  [JokerSwap]: { own: 1, targets: 0, seats: 1, rank: false },
  [JokerTripleCall]: { own: 1, targets: 3, seats: 0, rank: false },
  [JokerSelfMark]: { own: 1, targets: 0, seats: 0, rank: false },
  [JokerSweep]: { own: 1, targets: 0, seats: 1, rank: false },
  [JokerExtraLife]: { own: 0, targets: 0, seats: 0, rank: false },
  [JokerSecondWind]: { own: 0, targets: 0, seats: 2, rank: false },
  [JokerCensus]: { own: 0, targets: 0, seats: 0, rank: true },
  [JokerShield]: { own: 0, targets: 0, seats: 0, rank: false },
  [JokerReverseCall]: { own: 2, targets: 1, seats: 0, rank: false },
  [JokerPass]: { own: 0, targets: 0, seats: 1, rank: false },
};

const noNeeds: JokerShape = { own: 0, targets: 0, seats: 0, rank: false };

/** Second wind names two seats only when more than two have spent their safe call. */
export function jokerShape(view: PlayerView, rank: Rank): JokerShape {
  const shape = shapes[rank] ?? noNeeds;
  if (rank !== JokerSecondWind) return shape;
  const used = view.PowerUsed.filter(Boolean).length;
  return { ...shape, seats: used > 2 ? 2 : 0 };
}

function seatsWithFaceDown(view: PlayerView): Set<SeatID> {
  const seats = new Set<SeatID>();
  for (const c of view.Cards) if (c.Facing === Face.Down) seats.add(c.Owner);
  return seats;
}

/** Seats the viewer may name for a seat-picking joker, ascending. */
/** Suited ranks 2..K a Census can still ask about (not all four face-up). */
export function censusRanks(view: PlayerView): number[] {
  const faceUp = new Map<number, number>();
  for (const c of view.Cards) {
    if (c.Facing !== Face.Up || c.Value?.Kind !== SpecialKind.None) continue;
    faceUp.set(c.Value.Rank, (faceUp.get(c.Value.Rank) ?? 0) + 1);
  }
  return Array.from({ length: 12 }, (_, i) => i + 2).filter((r) => (faceUp.get(r) ?? 0) < 4);
}

export function jokerSeatChoices(view: PlayerView, rank: Rank): SeatID[] {
  if (rank === JokerSecondWind) {
    return view.PowerUsed.flatMap((used, seat) => (used ? [seat] : []));
  }
  const withCards = seatsWithFaceDown(view);
  return [0, 1, 2, 3].filter((s) => s !== view.Seat && withCards.has(s));
}

/** Own cards a joker may use as its source(s). */
export function jokerOwnCards(view: PlayerView, rank: Rank): ViewCard[] {
  switch (rank) {
    case JokerTripleCall:
    case JokerReverseCall:
      return sourceCards(view);
    case JokerSweep:
    case JokerSelfMark:
      return sourceCards(view).filter((c) => c.Value?.Kind === SpecialKind.None);
    default:
      return view.Cards.filter((c) => c.Owner === view.Seat && c.Facing === Face.Down);
  }
}

/** Other-player cards a joker may target. */
export function jokerTargetCards(view: PlayerView, rank: Rank): ViewCard[] {
  return jokerShape(view, rank).targets > 0 ? targetCards(view) : [];
}

/** The picks made so far for a joker activation. */
export interface JokerPicks {
  own: readonly CardID[];
  targets: readonly CardID[];
  seats: readonly SeatID[];
  rank: number | null;
}

export function jokerComplete(view: PlayerView, rank: Rank, picks: JokerPicks): boolean {
  const shape = jokerShape(view, rank);
  if (picks.own.length !== shape.own) return false;
  if (picks.targets.length !== shape.targets) return false;
  if (picks.seats.length !== shape.seats) return false;
  if (shape.seats === 2 && picks.seats[0] === picks.seats[1]) return false;
  return !shape.rank || picks.rank !== null;
}

/** The joker message for a complete selection, or null while incomplete. */
export function buildJoker(view: PlayerView, rank: Rank, picks: JokerPicks): JokerMessage | null {
  if (!jokerComplete(view, rank, picks)) return null;
  const msg: JokerMessage = { type: "joker", joker: rank };
  if (picks.own.length > 0) msg.sourceCardIds = [...picks.own];
  if (picks.targets.length > 0) msg.targetCardIds = [...picks.targets];
  if (picks.seats.length > 0) msg.targetSeats = [...picks.seats];
  if (rank === JokerCensus) msg.value = picks.rank ?? undefined;
  return msg;
}
