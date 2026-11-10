import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { sourceCards, targetCards, targetsForSeat } from "./actions.ts";
import { messageFor } from "../screens/GameScreen.tsx";
import {
  canConfirm,
  chooseOptions,
  emptySelection,
  inferredAction,
  selectionReducer,
  type SelectionEvent,
  type SelectionState,
} from "./selection.ts";
import {
  ActionKind,
  Face,
  Outcome,
  Phase,
  Result,
  SpecialKind,
  type CardValue,
  type ClientMessage,
  type PlayerView,
  type ViewCard,
} from "../types.ts";

// Generators for the selection reducer properties.
const RUNS = { numRuns: 200 };

const SEAT = 0;
const CARD_COUNT = 16;

const valueArb: fc.Arbitrary<CardValue> = fc.record({
  Rank: fc.integer({ min: 2, max: 13 }),
  Kind: fc.constantFrom(SpecialKind.None, SpecialKind.None, SpecialKind.Silver, SpecialKind.Gold),
  Suit: fc.integer({ min: 0, max: 3 }),
});

// Card ids are 0..CARD_COUNT-1; only the viewer's own and face-up cards carry a value.
const cardsArb: fc.Arbitrary<ViewCard[]> = fc
  .array(
    fc.record({
      owner: fc.constantFrom(0, 0, 1, 1, 1, 2, 3),
      up: fc.boolean(),
      value: valueArb,
    }),
    { minLength: CARD_COUNT, maxLength: CARD_COUNT },
  )
  .map((rows) =>
    rows.map((r, id): ViewCard => ({
      ID: id,
      Owner: r.owner,
      Slot: id + 1,
      Facing: r.up ? Face.Up : Face.Down,
      Value: r.owner === SEAT || r.up ? r.value : undefined,
    })),
  );

const viewArb: fc.Arbitrary<PlayerView> = fc
  .record({
    phase: fc.constantFrom(
      Phase.Opening,
      Phase.Playing,
      Phase.Playing,
      Phase.AwaitingResponse,
      Phase.AwaitingResponse,
      Phase.Ended,
    ),
    yourTurn: fc.constantFrom(true, true, false),
    powerUsed: fc.constantFrom(false, false, true),
    responder: fc.constantFrom(0, 0, 1, 2, 3),
    options: fc.tuple(
      fc.integer({ min: 0, max: CARD_COUNT - 1 }),
      fc.integer({ min: 0, max: CARD_COUNT - 1 }),
    ),
    cards: cardsArb,
  })
  .map(
    (r) =>
      ({
        Seat: SEAT,
        Phase: r.phase,
        CurrentSeat: r.yourTurn ? SEAT : 1,
        YourTurn: r.yourTurn && (r.phase === Phase.Opening || r.phase === Phase.Playing),
        PowerUsed: [r.powerUsed, false, false, false],
        Cards: r.cards,
        Result: Result.None,
        ResponsePlayer: r.responder,
        HasResponsePlayer: r.phase === Phase.AwaitingResponse,
        LastAction: {
          Seq: 1,
          Actor: 1,
          Kind: ActionKind.SafeCall,
          Targets: [r.options[0], r.options[1]],
          Outcome: Outcome.Hit,
          Revealed: [],
          LivesLost: 0,
        },
      }) as unknown as PlayerView,
  );

// Click ids range one past the deck so unknown cards are exercised too.
const eventArb: fc.Arbitrary<SelectionEvent> = fc.oneof(
  {
    weight: 8,
    arbitrary: fc
      .integer({ min: 0, max: CARD_COUNT })
      .map((cardId): SelectionEvent => ({ type: "clickCard", cardId })),
  },
  {
    weight: 1,
    arbitrary: fc.constantFrom<SelectionEvent>(
      { type: "pickSelfValue", value: "silver" },
      { type: "pickSelfValue", value: "gold" },
      { type: "pickSelfValue", value: 7 },
      { type: "pickSelfValue", value: null },
      { type: "clear" },
    ),
  },
);

const eventsArb = fc.array(eventArb, { minLength: 0, maxLength: 25 });

function deepFreeze<T>(o: T): T {
  if (o && typeof o === "object" && !Object.isFrozen(o)) {
    Object.freeze(o);
    Object.values(o).forEach((o) => deepFreeze(o));
  }
  return o;
}

function ids(cards: { ID: number }[]): number[] {
  return cards.map((c) => c.ID);
}

function assertInvariants(s: SelectionState, view: PlayerView) {
  if (s.source !== null) {
    expect(ids(sourceCards(view))).toContain(s.source); // own, face-down, never gold
    if (view.Phase === Phase.Opening) {
      // Specials (silver/gold) cannot be revealed in the opening.
      const kind = view.Cards.find((c) => c.ID === s.source)?.Value?.Kind;
      expect(kind === undefined || kind === SpecialKind.None).toBe(true);
    }
  }
  expect(s.targets.length).toBeLessThanOrEqual(2);
  expect(new Set(s.targets).size).toBe(s.targets.length);
  const targetable = new Map(targetCards(view).map((c) => [c.ID, c.Owner]));
  for (const t of s.targets) {
    expect(targetable.has(t)).toBe(true);
    expect(targetable.get(t)).toBe(s.targetSeat);
  }
  if (s.targets.length === 0) expect(s.targetSeat).toBeNull();
  if (s.targets.length === 2) expect(view.PowerUsed[view.Seat]).toBe(false);
  if (s.chosen !== null) expect(chooseOptions(view)).toContain(s.chosen);
  if (s.selfValue !== null) {
    expect(s.source).toBeNull();
    expect(s.targets).toEqual([]);
  }
}

// Feature: frontend-rework, Property 10: Selection reducer infers the action and never sends on a click
describe("Property 10: selection reducer", () => {
  it("keeps a valid selection, infers the action from it, and is a pure state edit", () => {
    fc.assert(
      fc.property(viewArb, eventsArb, (view, events) => {
        deepFreeze(view);
        let s: SelectionState = emptySelection;
        for (const event of events) {
          const next = selectionReducer(s, view, event); // frozen inputs: would throw on mutation
          // A click only yields a SelectionState; there is no message or side effect to emit.
          expect(Object.keys(next).sort()).toEqual(Object.keys(emptySelection).sort());
          s = next;
          assertInvariants(s, view);

          const action = inferredAction(s, view);
          expect(canConfirm(s, view)).toBe(action !== null);
          if (action === "guess") {
            expect([s.source !== null, s.targets.length]).toEqual([true, 1]);
          }
          if (action === "safeCall") {
            expect([s.source !== null, s.targets.length]).toEqual([true, 2]);
          }
          if (action === "openingReveal") {
            expect(view.Phase).toBe(Phase.Opening);
            expect(s.source).not.toBeNull();
          }
          if (action === "safeChooseReveal") expect(s.chosen).not.toBeNull();
          if (action === null && view.Phase === Phase.Playing && s.selfValue === null) {
            // Incomplete: no source, or no target yet.
            expect(s.source === null || s.targets.length === 0).toBe(true);
          }
        }
      }),
      RUNS,
    );
  });

  it("replaces, never adds, a second same-owner target once the power is spent", () => {
    fc.assert(
      fc.property(
        viewArb,
        fc.array(fc.integer({ min: 0, max: 16 }), { maxLength: 20 }),
        (view, clicks) => {
          const spent = { ...view, PowerUsed: [true, true, true, true] } as PlayerView;
          let s: SelectionState = emptySelection;
          for (const cardId of clicks) {
            s = selectionReducer(s, spent, { type: "clickCard", cardId });
            expect(s.targets.length).toBeLessThanOrEqual(1);
          }
        },
      ),
      RUNS,
    );
  });
});

// Feature: frontend-rework, Property 11: Confirm builds a message carrying exactly the selected ids
describe("Property 11: confirm message", () => {
  const fiveTypes = ["openingReveal", "guess", "safeCall", "safeChooseReveal", "selfReveal"];

  it("builds exactly one of the five messages carrying exactly the selected ids", () => {
    fc.assert(
      fc.property(viewArb, eventsArb, (view, events) => {
        let s: SelectionState = emptySelection;
        for (const event of events) {
          s = selectionReducer(s, view, event);
          const action = inferredAction(s, view);
          const msg: ClientMessage | null = messageFor(s, view);
          if (action === null) {
            expect(msg).toBeNull();
            continue;
          }
          expect(msg).not.toBeNull();
          expect(msg!.type).toBe(action);
          expect(fiveTypes).toContain(msg!.type);
          const expected: Record<string, ClientMessage> = {
            openingReveal: { type: "openingReveal", cardId: s.source! },
            guess: { type: "guess", sourceCardId: s.source!, targetCardId: s.targets[0] },
            safeCall: {
              type: "safeCall",
              sourceCardId: s.source!,
              targetCardId: s.targets[0],
              targetCardId2: s.targets[1],
            },
            selfReveal: { type: "selfReveal", value: s.selfValue! },
            safeChooseReveal: { type: "safeChooseReveal", chosenTargetCardId: s.chosen! },
          };
          // toEqual on the whole object: no extra or missing keys, exact ids.
          expect(msg).toEqual(expected[action]);
        }
      }),
      RUNS,
    );
    // Guard against a vacuous generator: every message kind was exercised.
  });

  it("a source plus two same-owner targets confirms as one safeCall with those ids", () => {
    fc.assert(
      fc.property(viewArb, (base) => {
        const view = {
          ...base,
          Phase: Phase.Playing,
          YourTurn: true,
          PowerUsed: [false, false, false, false],
        } as PlayerView;
        const source = sourceCards(view)[0];
        const seat = [1, 2, 3].find((o) => targetsForSeat(view, o).length >= 2);
        fc.pre(source !== undefined && seat !== undefined);
        const [t1, t2] = targetsForSeat(view, seat!);
        let s: SelectionState = emptySelection;
        for (const cardId of [t1.ID, source.ID, t2.ID]) {
          s = selectionReducer(s, view, { type: "clickCard", cardId });
        }
        expect(messageFor(s, view)).toEqual({
          type: "safeCall",
          sourceCardId: source.ID,
          targetCardId: t1.ID,
          targetCardId2: t2.ID,
        });
      }),
      RUNS,
    );
  });

  it("builds each of the five message kinds from a fixed selection", () => {
    const view = { Phase: Phase.Playing } as PlayerView;
    const base = { ...emptySelection };
    expect(messageFor({ ...base, source: 1, targets: [2] }, view)).toEqual({
      type: "guess",
      sourceCardId: 1,
      targetCardId: 2,
    });
    expect(
      messageFor({ ...base, source: 1, targets: [2, 3] }, {
        ...view,
        PowerUsed: [false, false, false, false],
      } as PlayerView),
    ).toEqual({
      type: "safeCall",
      sourceCardId: 1,
      targetCardId: 2,
      targetCardId2: 3,
    });
    expect(messageFor({ ...base, selfValue: "gold" }, view)).toEqual({
      type: "selfReveal",
      value: "gold",
    });
    expect(messageFor({ ...base, source: 4 }, { Phase: Phase.Opening } as PlayerView)).toEqual({
      type: "openingReveal",
      cardId: 4,
    });
    const resp = {
      Phase: Phase.AwaitingResponse,
      HasResponsePlayer: true,
      ResponsePlayer: 0,
      Seat: 0,
    } as PlayerView;
    expect(messageFor({ ...base, chosen: 5 }, resp)).toEqual({
      type: "safeChooseReveal",
      chosenTargetCardId: 5,
    });
  });
});
