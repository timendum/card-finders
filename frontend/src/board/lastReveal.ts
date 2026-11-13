// Pure derivation of which cards the last action flipped face-up, so every
// player can highlight them. Revealed is public, so the set is the same for
// every viewer.

import type { CardID, PlayerView } from "../types.ts";

export function lastRevealedIds(view: Pick<PlayerView, "LastAction">): ReadonlySet<CardID> {
  return new Set(view.LastAction?.Revealed ?? []);
}
