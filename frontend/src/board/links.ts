// Pairs up the joker link tokens (Same / Different). The wire carries no peer
// id: within one hand, consecutive slots holding the same link kind pair up
// left to right, matching the engine's sweep. Face-up cards carry no token, so
// they break a run.

import { Face, TokenKind, type CardID, type SeatID, type ViewCard } from "../types.ts";

export type LinkKind = TokenKind.Same | TokenKind.Different;

export interface LinkInfo {
  kind: LinkKind;
  /** First or second card of its pair; "single" only if the peer is missing. */
  part: "first" | "second" | "single";
}

function linkKindOf(card: ViewCard): LinkKind | null {
  if (card.Facing !== Face.Down) return null;
  const kind = card.Token?.Displayed;
  return kind === TokenKind.Same || kind === TokenKind.Different ? kind : null;
}

/** Link role per card id, over every hand in `cards`. */
export function linkRoles(cards: readonly ViewCard[]): Map<CardID, LinkInfo> {
  const bySeat = new Map<SeatID, ViewCard[]>();
  for (const c of cards) {
    const hand = bySeat.get(c.Owner);
    if (hand) hand.push(c);
    else bySeat.set(c.Owner, [c]);
  }

  const roles = new Map<CardID, LinkInfo>();
  for (const hand of bySeat.values()) {
    hand.sort((a, b) => a.Slot - b.Slot);
    for (let i = 0; i < hand.length; i++) {
      const kind = linkKindOf(hand[i]);
      if (kind === null) continue;
      const next = hand[i + 1];
      if (next && next.Slot === hand[i].Slot + 1 && linkKindOf(next) === kind) {
        roles.set(hand[i].ID, { kind, part: "first" });
        roles.set(next.ID, { kind, part: "second" });
        i++;
      } else {
        roles.set(hand[i].ID, { kind, part: "single" });
      }
    }
  }
  return roles;
}
