import { describe, expect, it } from "vitest";
import { matchHints } from "./matchHints.ts";
import { Face, Phase, SpecialKind, type PlayerView, type ViewCard } from "../types.ts";

let nextId = 1;
/** A card with an optional known value; face-down cards of other seats have none. */
function card(owner: number, slot: number, rank?: number, facing = Face.Down): ViewCard {
  return {
    ID: nextId++,
    Owner: owner,
    Slot: slot,
    Facing: facing,
    Value: rank === undefined ? undefined : { Rank: rank, Kind: SpecialKind.None, Suit: 0 },
  } as ViewCard;
}
const up = (owner: number, slot: number, rank: number) => card(owner, slot, rank, Face.Up);

function view(cards: ViewCard[]): PlayerView {
  return {
    Seat: 0,
    Phase: Phase.Playing,
    YourTurn: true,
    SpecialPool: { Silvers: [], Golds: [], SilverInPlay: 0, GoldInPlay: 0 },
    Cards: cards,
  } as unknown as PlayerView;
}

describe("matchHints placement", () => {
  it("pins the last copy to the lowest slot next to an equal face-up neighbour", () => {
    // Three 5s are accounted for (own source, left face-up, top face-up); the
    // fourth can only be in the left run, directly after the face-up 5.
    const source = card(0, 0, 5);
    const left = [up(1, 0, 5), card(1, 1), card(1, 2), card(1, 3), card(1, 4), up(1, 5, 10)];
    const top = [up(2, 0, 5), up(2, 1, 7), card(2, 2), card(2, 3), up(2, 4, 10)];
    const right = [up(3, 0, 6), up(3, 1, 7), card(3, 2), up(3, 3, 10)];
    const hints = matchHints(view([source, ...left, ...top, ...right]), source.ID)!;
    expect(hints.get(left[1]!.ID)).toBe("sure");
    for (const c of left.slice(2, 5)) expect(hints.get(c.ID)).toBe("excluded");
    for (const c of [top[2]!, top[3]!, right[2]!]) expect(hints.get(c.ID)).toBe("excluded");
  });
});
