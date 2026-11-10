import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  buildGuess,
  buildSafeCall,
  buildSelfReveal,
  selfRevealOptions,
  sourceCards,
  targetableSeats,
  targetCards,
  targetsForSeat,
} from "./actions.ts";
import {
  Face,
  Phase,
  Result,
  SpecialKind,
  type CardValue,
  type PlayerView,
  type SeatID,
  type ViewCard,
} from "../types.ts";

function card(id: number, owner: SeatID, facing: Face, value?: CardValue): ViewCard {
  return { ID: id, Owner: owner, Facing: facing, Value: value, Slot: id };
}

function suited(rank: number): CardValue {
  return { Rank: rank, Kind: SpecialKind.None, Suit: 0 };
}

function viewWith(cards: ViewCard[], seat: SeatID = 0): PlayerView {
  return {
    Seat: seat,
    Phase: Phase.Playing,
    Level: { Number: 1, Name: "L", SilverCount: 2, GoldCount: 1, StartLives: 4 },
    SpecialPool: { Silvers: [], Golds: [] },
    LifePool: 4,
    MaxLife: 4,
    CurrentSeat: seat,
    YourTurn: true,
    PowerUsed: [false, false, false, false],
    Cards: cards,
    Result: Result.None,
    SeatsFilled: 4,
    ResponsePlayer: 0,
    HasResponsePlayer: false,
    Log: [],
  };
}

describe("sourceCards", () => {
  it("returns the viewer's own face-down non-gold cards", () => {
    const view = viewWith([
      card(1, 0, Face.Down, suited(5)), // own, face-down -> source
      card(2, 0, Face.Up, suited(6)), // own but face-up -> excluded
      card(3, 0, Face.Down, { Rank: 7, Kind: SpecialKind.Gold, Suit: 0 }), // gold -> excluded
      card(4, 0, Face.Down, { Rank: 4, Kind: SpecialKind.Silver, Suit: 0 }), // silver -> included (feat)
      card(5, 1, Face.Down, suited(8)), // other seat -> excluded
    ]);
    expect(sourceCards(view).map((c) => c.ID)).toEqual([1, 4]);
  });
});

describe("targetCards / targetsForSeat / targetableSeats", () => {
  const view = viewWith([
    card(1, 0, Face.Down, suited(5)), // own -> never a target
    card(2, 1, Face.Down), // other, hidden -> target
    card(3, 1, Face.Up, suited(9)), // other, face-up -> excluded
    card(4, 2, Face.Down, { Rank: 7, Kind: SpecialKind.Gold, Suit: 0 }), // gold -> excluded
    card(5, 2, Face.Down), // other, hidden -> target
    card(6, 3, Face.Down), // other, hidden -> target
  ]);

  it("returns other players' face-down non-gold cards", () => {
    expect(
      targetCards(view)
        .map((c) => c.ID)
        .sort(),
    ).toEqual([2, 5, 6]);
  });

  it("filters targets to a single seat", () => {
    expect(targetsForSeat(view, 1).map((c) => c.ID)).toEqual([2]);
    expect(targetsForSeat(view, 2).map((c) => c.ID)).toEqual([5]);
    expect(targetsForSeat(view, 0).map((c) => c.ID)).toEqual([]);
  });

  it("lists seats with a targetable card in ascending order", () => {
    expect(targetableSeats(view)).toEqual([1, 2, 3]);
  });
});

describe("selfRevealOptions", () => {
  it("offers only fully-held ranks and silver; gold only when nothing else is held", () => {
    const silver: CardValue = { Rank: 4, Kind: SpecialKind.Silver, Suit: 0 };
    const view = viewWith([
      card(1, 0, Face.Down, suited(5)),
      card(2, 0, Face.Down, suited(5)),
      card(3, 1, Face.Up, suited(5)),
      card(4, 2, Face.Up, suited(5)), // rank 5: 2 held + 2 face-up -> offered
      card(5, 0, Face.Down, suited(2)), // rank 2: others may hold the rest -> not offered
      card(6, 0, Face.Down, silver),
      card(7, 0, Face.Down, { Rank: 9, Kind: SpecialKind.Gold, Suit: 0 }),
    ]);
    view.SpecialPool = {
      Silvers: [{ Rank: 4, Suit: 0, Kind: SpecialKind.Silver }],
      Golds: [],
    };
    expect(selfRevealOptions(view)).toEqual([
      { label: "Rank 5", value: 5 },
      { label: "All silver", value: "silver" },
    ]);

    const goldOnly = viewWith([
      card(1, 0, Face.Down, { Rank: 9, Kind: SpecialKind.Gold, Suit: 0 }),
      card(2, 0, Face.Up, suited(3)),
    ]);
    expect(selfRevealOptions(goldOnly)).toEqual([{ label: "All gold", value: "gold" }]);
  });
  it("is empty when the viewer holds no face-down cards", () => {
    expect(selfRevealOptions(viewWith([card(1, 0, Face.Up, suited(5))]))).toEqual([]);
  });
});

describe("message builders", () => {
  it("builds a guess envelope", () => {
    expect(buildGuess(10, 20)).toEqual({
      type: "guess",
      sourceCardId: 10,
      targetCardId: 20,
    });
  });

  it("builds a safe-call envelope with both targets", () => {
    expect(buildSafeCall(10, 20, 30)).toEqual({
      type: "safeCall",
      sourceCardId: 10,
      targetCardId: 20,
      targetCardId2: 30,
    });
  });

  it("builds a self-reveal envelope for each value kind", () => {
    expect(buildSelfReveal(7)).toEqual({ type: "selfReveal", value: 7 });
    expect(buildSelfReveal("silver")).toEqual({ type: "selfReveal", value: "silver" });
    expect(buildSelfReveal("gold")).toEqual({ type: "selfReveal", value: "gold" });
  });

  it("round-trips any card ids through guess/safeCall (fast-check)", () => {
    fc.assert(
      fc.property(fc.nat(65535), fc.nat(65535), fc.nat(65535), (s, t1, t2) => {
        const g = buildGuess(s, t1);
        expect(g.sourceCardId).toBe(s);
        expect(g.targetCardId).toBe(t1);
        const sc = buildSafeCall(s, t1, t2);
        expect(sc.sourceCardId).toBe(s);
        expect(sc.targetCardId).toBe(t1);
        expect(sc.targetCardId2).toBe(t2);
      }),
    );
  });

  it("never offers gold or face-up cards as a source (fast-check)", () => {
    const facingArb = fc.constantFrom(Face.Down, Face.Up);
    const kindArb = fc.constantFrom(SpecialKind.None, SpecialKind.Silver, SpecialKind.Gold);
    const cardArb = fc.record({
      ID: fc.nat(65535),
      Owner: fc.constantFrom<SeatID>(0, 1, 2, 3),
      Facing: facingArb,
      Kind: kindArb,
      Rank: fc.integer({ min: 2, max: 13 }),
    });
    fc.assert(
      fc.property(fc.array(cardArb, { maxLength: 20 }), (raw) => {
        const cards: ViewCard[] = raw.map((r) => ({
          ID: r.ID,
          Owner: r.Owner,
          Facing: r.Facing,
          Slot: r.ID,
          Value: { Rank: r.Rank, Kind: r.Kind, Suit: 0 },
        }));
        const view = viewWith(cards, 0);
        for (const src of sourceCards(view)) {
          expect(src.Owner).toBe(0);
          expect(src.Facing).toBe(Face.Down);
          expect(src.Value?.Kind).not.toBe(SpecialKind.Gold);
        }
        for (const tgt of targetCards(view)) {
          expect(tgt.Owner).not.toBe(0);
          expect(tgt.Facing).toBe(Face.Down);
          expect(tgt.Value?.Kind).not.toBe(SpecialKind.Gold);
        }
      }),
    );
  });
});
