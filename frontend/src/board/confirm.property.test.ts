import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { sourceCards, targetsForSeat } from "./actions.ts";
import { inferredAction, messageFor } from "./confirm.ts";
import { emptySelection, selectionReducer, type SelectionState } from "./selection.ts";
import { RUNS, eventsArb, viewArb } from "./selectionArbs.ts";
import { Phase, type ClientMessage, type PlayerView } from "../types.ts";

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
            openingReveal: { type: "openingReveal", targetCardIds: [s.source!] },
            guess: { type: "guess", sourceCardIds: [s.source!], targetCardIds: [s.targets[0]] },
            safeCall: {
              type: "safeCall",
              sourceCardIds: [s.source!],
              targetCardIds: [s.targets[0], s.targets[1]],
            },
            selfReveal: { type: "selfReveal", value: s.selfValue! },
            safeChooseReveal: { type: "safeChooseReveal", targetCardIds: [s.chosen!] },
          };
          // toEqual on the whole object: no extra or missing keys, exact ids.
          expect(msg).toEqual(expected[action]);
        }
      }),
      RUNS,
    );
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
          sourceCardIds: [source.ID],
          targetCardIds: [t1.ID, t2.ID],
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
      sourceCardIds: [1],
      targetCardIds: [2],
    });
    expect(
      messageFor({ ...base, source: 1, targets: [2, 3] }, {
        ...view,
        PowerUsed: [false, false, false, false],
      } as PlayerView),
    ).toEqual({
      type: "safeCall",
      sourceCardIds: [1],
      targetCardIds: [2, 3],
    });
    expect(messageFor({ ...base, selfValue: "gold" }, view)).toEqual({
      type: "selfReveal",
      value: "gold",
    });
    expect(messageFor({ ...base, source: 4 }, { Phase: Phase.Opening } as PlayerView)).toEqual({
      type: "openingReveal",
      targetCardIds: [4],
    });
    const resp = {
      Phase: Phase.AwaitingResponse,
      HasResponsePlayer: true,
      ResponsePlayer: 0,
      Seat: 0,
    } as PlayerView;
    expect(messageFor({ ...base, chosen: 5 }, resp)).toEqual({
      type: "safeChooseReveal",
      targetCardIds: [5],
    });
  });
});
