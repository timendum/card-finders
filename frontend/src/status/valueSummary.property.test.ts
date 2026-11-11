import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { buildValueSummary, buildValueSummaryOrder, sortValue } from "./ValueSummary.tsx";
import { inPlayTotal, selfRevealOptions } from "../board/actions.ts";
import {
  Face,
  Phase,
  Result,
  SpecialKind,
  type CardValue,
  type PlayerView,
  type ViewCard,
} from "../types.ts";

const RUNS = { numRuns: 200 };
const SEAT = 0;

const uniqueRanks = fc.uniqueArray(fc.integer({ min: 2, max: 12 }), { maxLength: 11 });

interface Spec {
  silvers: number[];
  golds: number[];
  silverInPlay: number;
  goldInPlay: number;
  /** Face-up count (0..4) per suited rank 2..13. */
  suitedUp: number[];
  silverUp: boolean[];
  goldUp: boolean[];
}

// Pool entries are distinct per kind; in-play never exceeds the entry count.
const specArb: fc.Arbitrary<Spec> = fc
  .record({ silvers: uniqueRanks, golds: uniqueRanks })
  .chain(({ silvers, golds }) =>
    fc.record({
      silvers: fc.constant(silvers),
      golds: fc.constant(golds),
      silverInPlay: fc.integer({ min: 0, max: silvers.length }),
      goldInPlay: fc.integer({ min: 0, max: golds.length }),
      suitedUp: fc.array(fc.integer({ min: 0, max: 4 }), { minLength: 12, maxLength: 12 }),
      silverUp: fc.array(fc.boolean(), { minLength: silvers.length, maxLength: silvers.length }),
      goldUp: fc.array(fc.boolean(), { minLength: golds.length, maxLength: golds.length }),
    }),
  );

const val = (Rank: number, Kind: SpecialKind): CardValue => ({ Rank, Kind, Suit: 0 });

function makeView(
  silvers: number[],
  golds: number[],
  silverInPlay: number,
  goldInPlay: number,
  cards: ViewCard[],
): PlayerView {
  return {
    Seat: SEAT,
    Phase: Phase.Playing,
    Level: { Number: 1, Name: "L", SilverCount: 0, GoldCount: 0, ExtraSilver: 0, ExtraGold: 0, StartLives: 4 },
    SpecialPool: {
      // Unsorted on purpose: the summary must order by Sort_Value itself.
      Silvers: silvers.map((r) => val(r, SpecialKind.Silver)),
      Golds: golds.map((r) => val(r, SpecialKind.Gold)),
      SilverInPlay: silverInPlay,
      GoldInPlay: goldInPlay,
    },
    LifePool: 4,
    MaxLife: 4,
    CurrentSeat: SEAT,
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

function faceUp(id: number, value: CardValue): ViewCard {
  return { ID: id, Owner: 1, Slot: id, Facing: Face.Up, Value: value };
}

function viewFromSpec(s: Spec): PlayerView {
  const cards: ViewCard[] = [];
  let id = 0;
  s.suitedUp.forEach((n, i) => {
    for (let k = 0; k < n; k++) cards.push(faceUp(id++, val(i + 2, SpecialKind.None)));
  });
  s.silvers.forEach((r, i) => s.silverUp[i] && cards.push(faceUp(id++, val(r, SpecialKind.Silver))));
  s.golds.forEach((r, i) => s.goldUp[i] && cards.push(faceUp(id++, val(r, SpecialKind.Gold))));
  return makeView(s.silvers, s.golds, s.silverInPlay, s.goldInPlay, cards);
}

// Feature: extra-decoy-specials, Property 8 (AGENTS Property 24): Value_Summary order
describe("Property 8: Value_Summary order", () => {
  it("lists suited 2..K plus every pool entry by Sort_Value, specials right after their rank", () => {
    fc.assert(
      fc.property(specArb, (s) => {
        const rows = buildValueSummaryOrder(viewFromSpec(s));
        expect(rows).toHaveLength(12 + s.silvers.length + s.golds.length);
        for (let i = 1; i < rows.length; i++) {
          expect(rows[i].SortValue).toBeGreaterThan(rows[i - 1].SortValue);
        }
        for (const row of rows) expect(row.SortValue).toBe(sortValue(row));
        rows.forEach((row, i) => {
          if (row.Kind === SpecialKind.None) return;
          // Walk back to the base rank: only same-rank specials may sit between.
          let j = i - 1;
          while (rows[j].Kind !== SpecialKind.None) j--;
          expect(rows[j].Rank).toBe(row.Rank);
        });
        const silverRanks = rows.filter((r) => r.Kind === SpecialKind.Silver).map((r) => r.Rank);
        expect(silverRanks).toEqual([...s.silvers].sort((a, b) => a - b));
        const goldRanks = rows.filter((r) => r.Kind === SpecialKind.Gold).map((r) => r.Rank);
        expect(goldRanks).toEqual([...s.golds].sort((a, b) => a - b));
      }),
      RUNS,
    );
  });
});

// Feature: extra-decoy-specials, Property 9 (AGENTS Property 25): Value_Summary counts and uncertainty
describe("Property 9: Value_Summary counts and uncertainty", () => {
  it("shows face-up counts, 1/1, completed 0/0, plain 0/1 or uncertain 0/1?", () => {
    fc.assert(
      fc.property(specArb, (s) => {
        const view = viewFromSpec(s);
        const rows = buildValueSummary(view);
        const foundSilver = s.silverUp.filter(Boolean).length;
        const foundGold = s.goldUp.filter(Boolean).length;

        for (const row of rows) {
          if (row.Kind === SpecialKind.None) {
            expect([row.Found, row.Total, row.Uncertain]).toEqual([s.suitedUp[row.Rank - 2], 4, false]);
            continue;
          }
          const silverKind = row.Kind === SpecialKind.Silver;
          const list = silverKind ? s.silvers : s.golds;
          const up = silverKind ? s.silverUp : s.goldUp;
          const inPlay = silverKind ? s.silverInPlay : s.goldInPlay;
          const foundSum = silverKind ? foundSilver : foundGold;
          const found = up[list.indexOf(row.Rank)] ? 1 : 0;

          let expected: [number, number, boolean];
          if (found === 1) expected = [1, 1, false];
          else if (foundSum === inPlay) expected = [0, 0, false];
          else if (list.length === inPlay) expected = [0, 1, false];
          else expected = [0, 1, true];
          expect([row.Found, row.Total, row.Uncertain]).toEqual(expected);
        }
      }),
      RUNS,
    );
  });

  it("is a pure function of the latest view", () => {
    fc.assert(
      fc.property(specArb, (s) => {
        const view = viewFromSpec(s);
        expect(buildValueSummary(view)).toEqual(buildValueSummary(structuredClone(view)));
      }),
      RUNS,
    );
  });
});

interface Hands {
  /** Viewer's own face-down cards. */
  own: CardValue[];
  /** Face-up cards anywhere (counted as found). */
  up: CardValue[];
}

const handValueArb: fc.Arbitrary<CardValue> = fc.record({
  Rank: fc.integer({ min: 2, max: 12 }),
  Kind: fc.constantFrom(SpecialKind.None, SpecialKind.Silver, SpecialKind.Silver, SpecialKind.Gold),
  Suit: fc.integer({ min: 0, max: 3 }),
});

const handsArb: fc.Arbitrary<Hands> = fc.record({
  own: fc.array(handValueArb, { maxLength: 8 }),
  up: fc.array(handValueArb, { maxLength: 8 }),
});

function viewFromHands(h: Hands, silvers: number[], golds: number[], silverInPlay: number, goldInPlay: number) {
  const cards: ViewCard[] = [
    ...h.own.map((v, i): ViewCard => ({ ID: i, Owner: SEAT, Slot: i, Facing: Face.Down, Value: v })),
    ...h.up.map((v, i): ViewCard => ({ ID: 100 + i, Owner: 1, Slot: i, Facing: Face.Up, Value: v })),
  ];
  return makeView(silvers, golds, silverInPlay, goldInPlay, cards);
}

// Feature: extra-decoy-specials, Property 10: Frontend self-reveal uses in-play counts
describe("Property 10: self-reveal uses in-play counts", () => {
  it("offers 'All silver' exactly when held + face-up silvers equal SilverInPlay", () => {
    fc.assert(
      fc.property(
        handsArb,
        uniqueRanks,
        uniqueRanks,
        fc.integer({ min: 0, max: 11 }),
        fc.integer({ min: 0, max: 11 }),
        (h, silvers, golds, silverInPlay, goldInPlay) => {
          const view = viewFromHands(h, silvers, golds, silverInPlay, goldInPlay);
          const isSilver = (v: CardValue) => v.Kind === SpecialKind.Silver;
          const held = h.own.filter(isSilver).length;
          const upSilver = h.up.filter(isSilver).length;
          const options = selfRevealOptions(view).map((o) => o.value);

          expect(options.includes("silver")).toBe(held > 0 && held + upSilver === silverInPlay);

          const heldGold = h.own.filter((v) => v.Kind === SpecialKind.Gold).length;
          const heldOther = h.own.filter((v) => v.Kind !== SpecialKind.Gold).length;
          expect(options.includes("gold")).toBe(heldGold > 0 && heldOther === 0);

          expect(inPlayTotal(view, SpecialKind.Silver)).toBe(silverInPlay);
          expect(inPlayTotal(view, SpecialKind.Gold)).toBe(goldInPlay);
        },
      ),
      RUNS,
    );
  });

  it("does not depend on how many pool entries are listed", () => {
    fc.assert(
      fc.property(
        handsArb,
        uniqueRanks,
        uniqueRanks,
        uniqueRanks,
        uniqueRanks,
        fc.integer({ min: 0, max: 11 }),
        fc.integer({ min: 0, max: 11 }),
        (h, s1, g1, s2, g2, silverInPlay, goldInPlay) => {
          const a = selfRevealOptions(viewFromHands(h, s1, g1, silverInPlay, goldInPlay));
          const b = selfRevealOptions(viewFromHands(h, s2, g2, silverInPlay, goldInPlay));
          expect(a).toEqual(b);
        },
      ),
      RUNS,
    );
  });
});
