import { describe, expect, it } from "vitest";
import {
  buildJoker,
  isAnytime,
  jokerComplete,
  jokerNames,
  jokerSeatChoices,
  jokerShape,
  jokerUsable,
} from "./jokers.ts";
import {
  Face,
  JokerCensus,
  JokerDiffer,
  JokerReverseCall,
  JokerSecondWind,
  JokerState,
  JokerTripleCall,
  Phase,
  SpecialKind,
  type PlayerView,
  type ViewCard,
} from "../types.ts";

const ranks = Array.from({ length: 12 }, (_, i) => i + 2);

function card(id: number, owner: number, slot: number): ViewCard {
  return {
    ID: id,
    Owner: owner,
    Slot: slot,
    Facing: Face.Down,
    Value: { Rank: 5, Kind: SpecialKind.None, Suit: 0 },
  } as ViewCard;
}

function view(over: Partial<PlayerView> = {}): PlayerView {
  return {
    Seat: 0,
    Phase: Phase.Playing,
    YourTurn: true,
    PowerUsed: [false, false, false, false],
    Cards: [card(1, 0, 1), card(2, 0, 2), card(3, 1, 1), card(4, 1, 2), card(5, 1, 3)],
    Jokers: ranks.map((Rank) => ({ Rank, State: JokerState.Ready })),
    ...over,
  } as PlayerView;
}

describe("joker names and timing", () => {
  it("names every rank 2..13 and nothing else", () => {
    expect(Object.keys(jokerNames).map(Number).sort((a, b) => a - b)).toEqual(ranks);
  });

  it("splits ranks into anytime and current-player jokers", () => {
    const anytime = ranks.filter(isAnytime);
    expect(anytime).toEqual([2, 3, 5, 7, 8, 9, 13]);
  });

  it("is usable only when ready, in Playing, and the turn rule holds", () => {
    const off = view({ YourTurn: false });
    expect(jokerUsable(off, 7)).toBe(true);
    expect(jokerUsable(off, 10)).toBe(false);
    expect(jokerUsable(view(), 10)).toBe(true);
    expect(jokerUsable(view({ Phase: Phase.AwaitingResponse }), 7)).toBe(false);
    const inactive = view({ Jokers: [{ Rank: 7, State: JokerState.Inactive }] });
    expect(jokerUsable(inactive, 7)).toBe(false);
    const used = view({ Jokers: [{ Rank: 7, State: JokerState.Used }] });
    expect(jokerUsable(used, 7)).toBe(false);
  });
});

describe("joker shapes and messages", () => {
  it("asks Second wind for two seats only when more than two used a safe call", () => {
    expect(jokerShape(view(), JokerSecondWind).seats).toBe(0);
    const three = view({ PowerUsed: [true, true, true, false] });
    expect(jokerShape(three, JokerSecondWind).seats).toBe(2);
    expect(jokerSeatChoices(three, JokerSecondWind)).toEqual([0, 1, 2]);
  });

  it("offers other seats that still hold a face-down card", () => {
    expect(jokerSeatChoices(view(), 12)).toEqual([1]);
  });

  it("builds nothing until the picks are complete", () => {
    const picks = { own: [], targets: [], seats: [], rank: null };
    expect(jokerComplete(view(), JokerDiffer, picks)).toBe(false);
    expect(buildJoker(view(), JokerDiffer, picks)).toBeNull();
    expect(buildJoker(view(), JokerCensus, picks)).toBeNull();
    expect(buildJoker(view(), 7, picks)).toEqual({ type: "joker", joker: 7 });
  });

  it("maps picks to the wire fields of each shape", () => {
    expect(buildJoker(view(), JokerDiffer, { own: [1, 2], targets: [], seats: [], rank: null })).toEqual({
      type: "joker",
      joker: JokerDiffer,
      sourceCardId: 1,
      targetCardId: 2,
    });
    expect(
      buildJoker(view(), JokerTripleCall, { own: [1], targets: [3, 4, 5], seats: [], rank: null }),
    ).toEqual({
      type: "joker",
      joker: JokerTripleCall,
      sourceCardId: 1,
      targetCardId: 3,
      targetCardId2: 4,
      targetCardId3: 5,
    });
    expect(
      buildJoker(view(), JokerReverseCall, { own: [1, 2], targets: [3], seats: [], rank: null }),
    ).toMatchObject({ sourceCardId: 1, sourceCardId2: 2, targetCardId: 3 });
    expect(buildJoker(view(), JokerCensus, { own: [], targets: [], seats: [], rank: 9 })).toEqual({
      type: "joker",
      joker: JokerCensus,
      value: 9,
    });
    expect(buildJoker(view(), 12, { own: [], targets: [], seats: [0], rank: null })).toMatchObject({
      targetSeat: 0,
    });
  });
});
