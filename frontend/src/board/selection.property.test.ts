import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { sourceCards, targetCards } from "./actions.ts";
import { canConfirm, inferredAction } from "./confirm.ts";
import {
  chooseOptions,
  emptySelection,
  selectionReducer,
  type SelectionState,
} from "./selection.ts";
import { RUNS, eventsArb, viewArb } from "./selectionArbs.ts";
import { Phase, SpecialKind, type PlayerView } from "../types.ts";

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
