// Pure slot grouping. Order comes only from the backend-assigned Slot, never
// from card value or id.

import type { SeatID, ViewCard } from "../types.ts";

export function cardsBySeat(cards: readonly ViewCard[]): Map<SeatID, ViewCard[]> {
  const bySeat = new Map<SeatID, ViewCard[]>();
  for (const card of cards) {
    const hand = bySeat.get(card.Owner);
    if (hand) hand.push(card);
    else bySeat.set(card.Owner, [card]);
  }
  for (const hand of bySeat.values()) hand.sort((a, b) => a.Slot - b.Slot);
  return bySeat;
}

// One entry per slot 1..N, where N is the hand size; a slot with no card is
// undefined. Slots are contiguous, so the length never changes during a game.
export function slotted(hand: readonly ViewCard[]): (ViewCard | undefined)[] {
  const bySlot = new Map(hand.map((c) => [c.Slot, c]));
  return Array.from({ length: hand.length }, (_, i) => bySlot.get(i + 1));
}
