// Pure action-selection logic for the ActionBar.
//
// The ActionBar lets the viewing player assemble one of three turn actions —
// a guess (which is the silver feat when its source card is silver), a
// safe call, or a self-reveal — and send the matching WebSocket ClientMessage.
// All of the "what can I pick?" and "what message does this produce?" logic
// lives here as pure, I/O-free functions so the component stays a thin view and
// this can be unit/property tested without a DOM.
//
// Selection rules mirror the engine's validation (backend engine package); the
// server remains the sole authority and rejects anything illegal with an
// `error` frame, so these helpers only need to offer the plausibly
// legal choices, not re-implement every edge of the rules.

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

/** True when a card is currently face-down (a candidate source or target). */
function isFaceDown(card: ViewCard): boolean {
  return card.Facing === Face.Down;
}

/** True when a card's known value is a gold special. */
function isGold(card: ViewCard): boolean {
  return card.Value != null && card.Value.Kind === SpecialKind.Gold;
}

/**
 * sourceCards returns the viewer's own face-down cards that may be played as a
 * guess/safe-call source: all of the viewer's face-down cards except gold,
 * which can never be a source. A silver source card here is what
 * makes the guess the silver feat. The viewer always sees its own
 * cards' values, so Value is non-null for every owned card.
 */
export function sourceCards(view: PlayerView): ViewCard[] {
  return view.Cards.filter(
    (card) => card.Owner === view.Seat && isFaceDown(card) && !isGold(card),
  );
}

/**
 * targetCards returns the face-down cards of OTHER players that may be targeted
 * by a guess or safe call. Gold is never targetable by another player, so it is excluded; a card whose value the viewer cannot see still
 * appears (it is a legal target) — only cards known to be gold are filtered,
 * which are exactly the gold specials the viewer can see face-up or via token.
 * The engine makes the final legality call.
 */
export function targetCards(view: PlayerView): ViewCard[] {
  return view.Cards.filter(
    (card) => card.Owner !== view.Seat && isFaceDown(card) && !isGold(card),
  );
}

/**
 * targetsForSeat returns the targetable face-down cards owned by one specific
 * other seat. A safe call's two targets must belong to the same single other
 * player, so the UI first picks a target seat, then two of that
 * seat's cards from this list.
 */
export function targetsForSeat(view: PlayerView, seat: SeatID): ViewCard[] {
  return targetCards(view).filter((card) => card.Owner === seat);
}

/**
 * targetableSeats returns the other seats that currently have at least one
 * targetable face-down card, in ascending seat order. Used to populate the
 * target-player picker for both guesses and safe calls.
 */
export function targetableSeats(view: PlayerView): SeatID[] {
  const seats = new Set<SeatID>();
  for (const card of targetCards(view)) {
    seats.add(card.Owner);
  }
  return [...seats].sort((a, b) => a - b);
}

/**
 * A self-reveal option the viewer may choose: a suited rank, all silver, or all
 * gold. `label` is the human text for the picker and `value` is the
 * wire value carried by the selfReveal message.
 */
export interface SelfRevealOption {
  /** Short display label, e.g. "Rank 7", "All silver", "All gold". */
  label: string;
  /** The wire value: a suited rank number, or "silver" / "gold". */
  value: SelfRevealValue;
}

/**
 * selfRevealOptions returns the values the viewer could plausibly self-reveal,
 * derived from the viewer's own face-down cards: each distinct suited rank the
 * viewer still holds face-down, plus "silver" if the viewer holds a face-down
 * silver and "gold" if the viewer holds a face-down gold.
 * Eligibility (holding ALL remaining face-down cards of the value, or all-gold
 * for a gold reveal) is enforced by the engine; the UI
 * offers the viewer's held values and lets the server reject an ineligible one.
 *
 * Suited ranks are listed in ascending order, then silver, then gold, matching
 * the Value_Summary ordering so the picker reads consistently.
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

/**
 * buildGuess builds the `guess` message for a source/target pair. A
 * silver source card makes this the silver feat on the same path, so no
 * separate message type is needed.
 */
export function buildGuess(sourceCardId: CardID, targetCardId: CardID): GuessMessage {
  return { type: "guess", sourceCardId, targetCardId };
}

/**
 * buildSafeCall builds the `safeCall` message for a source card and two distinct
 * targets of one other player. The two targets map to targetCardId and
 * targetCardId2 of the wire envelope.
 */
export function buildSafeCall(
  sourceCardId: CardID,
  targetCardId: CardID,
  targetCardId2: CardID,
): SafeCallMessage {
  return { type: "safeCall", sourceCardId, targetCardId, targetCardId2 };
}

/** buildSelfReveal builds the `selfReveal` message for a chosen value. */
export function buildSelfReveal(value: SelfRevealValue): SelfRevealMessage {
  return { type: "selfReveal", value };
}
