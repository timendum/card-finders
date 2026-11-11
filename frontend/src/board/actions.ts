// Pure action-selection logic: which cards/values the viewer
// may pick, and the ClientMessage each choice produces. These helpers offer
// only plausibly-legal choices; the engine is the sole authority and rejects
// anything illegal with an `error` frame.

import { rankLabel } from "./cardLabel.ts";
import {
  Face,
  SpecialKind,
  type CardID,
  type GuessMessage,
  type PlayerView,
  type SafeCallMessage,
  type SeatID,
  type SelfRevealMessage,
  type SelfRevealValue,
  type ViewCard,
} from "../types.ts";

function isFaceDown(card: ViewCard): boolean {
  return card.Facing === Face.Down;
}

function isGold(card: ViewCard): boolean {
  return card.Value != null && card.Value.Kind === SpecialKind.Gold;
}

/**
 * sourceCards returns the viewer's own face-down non-gold cards (gold can never
 * be a source). A silver source makes the guess the silver feat.
 */
export function sourceCards(view: PlayerView): ViewCard[] {
  return view.Cards.filter((card) => card.Owner === view.Seat && isFaceDown(card) && !isGold(card));
}

/**
 * targetCards returns other players' face-down cards targetable by a guess or
 * safe call. Cards known to be gold are excluded; a card whose value the viewer
 * cannot see still appears.
 */
export function targetCards(view: PlayerView): ViewCard[] {
  return view.Cards.filter((card) => card.Owner !== view.Seat && isFaceDown(card) && !isGold(card));
}

/**
 * targetsForSeat returns the targetable face-down cards owned by one other
 * seat. A safe call's two targets must belong to the same other player.
 */
export function targetsForSeat(view: PlayerView, seat: SeatID): ViewCard[] {
  return targetCards(view).filter((card) => card.Owner === seat);
}

/**
 * targetableSeats returns the other seats with at least one targetable
 * face-down card, in ascending seat order.
 */
export function targetableSeats(view: PlayerView): SeatID[] {
  const seats = new Set<SeatID>();
  for (const card of targetCards(view)) {
    seats.add(card.Owner);
  }
  return [...seats].sort((a, b) => a - b);
}

/**
 * inPlayTotal returns how many cards of a special kind are actually in play.
 * The pool may list decoy entries, so never derive this from the entry count.
 */
export function inPlayTotal(
  view: PlayerView,
  kind: typeof SpecialKind.Silver | typeof SpecialKind.Gold,
): number {
  return kind === SpecialKind.Silver ? view.SpecialPool.SilverInPlay : view.SpecialPool.GoldInPlay;
}

/** A self-reveal option: a picker `label` and the wire `value` it sends. */
export interface SelfRevealOption {
  label: string;
  value: SelfRevealValue;
}

/**
 * selfRevealOptions returns the values the viewer may self-reveal: each suited
 * rank (ascending) they hold in full, then "silver", then "gold", matching
 * Value_Summary order. The engine remains the final authority.
 */
export function selfRevealOptions(view: PlayerView): SelfRevealOption[] {
  const heldRanks = new Map<number, number>();
  const faceUpRanks = new Map<number, number>();
  let heldSilver = 0;
  let faceUpSilver = 0;
  let heldGold = 0;
  let heldOther = 0;

  for (const card of view.Cards) {
    const value = card.Value;
    if (value == null) continue;
    const mine = card.Owner === view.Seat && isFaceDown(card);
    if (!mine) {
      if (card.Facing === Face.Up) {
        if (value.Kind === SpecialKind.Silver) faceUpSilver++;
        else if (value.Kind === SpecialKind.None) {
          faceUpRanks.set(value.Rank, (faceUpRanks.get(value.Rank) ?? 0) + 1);
        }
      }
      continue;
    }
    switch (value.Kind) {
      case SpecialKind.Silver:
        heldSilver++;
        heldOther++;
        break;
      case SpecialKind.Gold:
        heldGold++;
        break;
      default:
        heldRanks.set(value.Rank, (heldRanks.get(value.Rank) ?? 0) + 1);
        heldOther++;
        break;
    }
  }

  // Only offer values the viewer holds in full (every remaining face-down card).
  const ranks = [...heldRanks.entries()]
    .filter(([rank, held]) => held + (faceUpRanks.get(rank) ?? 0) === 4)
    .map(([rank]) => rank);
  const hasSilver =
    heldSilver > 0 && heldSilver + faceUpSilver === inPlayTotal(view, SpecialKind.Silver);
  // Gold: only once every other face-down card the viewer holds is gone.
  const hasGold = heldGold > 0 && heldOther === 0;

  const options: SelfRevealOption[] = [...ranks]
    .sort((a, b) => a - b)
    .map((rank) => ({ label: `Rank ${rankLabel(rank)}`, value: rank }));
  if (hasSilver) options.push({ label: "All silver", value: "silver" });
  if (hasGold) options.push({ label: "All gold", value: "gold" });
  return options;
}

export function buildGuess(sourceCardId: CardID, targetCardId: CardID): GuessMessage {
  return { type: "guess", sourceCardId, targetCardId };
}

export function buildSafeCall(
  sourceCardId: CardID,
  targetCardId: CardID,
  targetCardId2: CardID,
): SafeCallMessage {
  return { type: "safeCall", sourceCardId, targetCardId, targetCardId2 };
}

export function buildSelfReveal(value: SelfRevealValue): SelfRevealMessage {
  return { type: "selfReveal", value };
}
