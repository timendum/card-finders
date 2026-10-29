// Pure action-selection logic for the ActionBar: which cards/values the viewer
// may pick, and the ClientMessage each choice produces. These helpers offer
// only plausibly-legal choices; the engine is the sole authority and rejects
// anything illegal with an `error` frame.

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
} from "./types.ts";

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

/** A self-reveal option: a picker `label` and the wire `value` it sends. */
export interface SelfRevealOption {
  label: string;
  value: SelfRevealValue;
}

/**
 * selfRevealOptions returns the values the viewer holds face-down: each distinct
 * suited rank (ascending), then "silver", then "gold", matching Value_Summary
 * order. Eligibility is enforced by the engine.
 */
export function selfRevealOptions(view: PlayerView): SelfRevealOption[] {
  const ranks = new Set<number>();
  let hasSilver = false;
  let hasGold = false;

  for (const card of view.Cards) {
    if (card.Owner !== view.Seat || !isFaceDown(card) || card.Value == null) {
      continue;
    }
    switch (card.Value.Kind) {
      case SpecialKind.Silver:
        hasSilver = true;
        break;
      case SpecialKind.Gold:
        hasGold = true;
        break;
      default:
        ranks.add(card.Value.Rank);
        break;
    }
  }

  const options: SelfRevealOption[] = [...ranks]
    .sort((a, b) => a - b)
    .map((rank) => ({ label: `Rank ${rank}`, value: rank }));
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
