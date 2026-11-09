import { describe, expect, it } from "vitest";
import {
  buildChoice,
  emptySelection,
  inferredAction,
  isSelectable,
  selectionReducer,
} from "./selection.ts";
import {
  ActionKind,
  Face,
  Outcome,
  Phase,
  Result,
  SpecialKind,
  type LastAction,
  type PlayerView,
  type ViewCard,
} from "./types.ts";

function card(id: number, owner: number): ViewCard {
  return {
    ID: id,
    Owner: owner,
    Facing: Face.Down,
    Slot: id,
    Value: { Rank: 5, Kind: SpecialKind.None, Suit: 0 },
  } as ViewCard;
}

function awaiting(seat: number, responder: number): PlayerView {
  return {
    Seat: seat,
    Phase: Phase.AwaitingResponse,
    CurrentSeat: 0,
    YourTurn: seat === 0,
    PowerUsed: [true, false, false, false],
    Cards: [card(1, 0), card(2, 1), card(3, 1), card(4, 1), card(5, 2)],
    Result: Result.None,
    ResponsePlayer: responder,
    HasResponsePlayer: true,
    LastAction: {
      Seq: 1,
      Actor: 0,
      Kind: ActionKind.SafeCall,
      Targets: [2, 3],
      Outcome: Outcome.Hit,
      Revealed: [],
      LivesLost: 0,
    } as LastAction,
  } as PlayerView;
}

describe("safe-choose-reveal selection", () => {
  const click = (view: PlayerView, id: number, s = emptySelection) =>
    selectionReducer(s, view, { type: "clickCard", cardId: id });

  it("makes only the two called cards selectable for the responder", () => {
    const view = awaiting(1, 1);
    const ids = view.Cards.filter((c) => isSelectable(view, c)).map((c) => c.ID);
    expect(ids).toEqual([2, 3]);
  });

  it("makes nothing selectable for others, even the caller on their turn", () => {
    for (const seat of [0, 2, 3]) {
      const view = awaiting(seat, 1);
      expect(view.Cards.some((c) => isSelectable(view, c))).toBe(false);
      expect(click(view, 2)).toBe(emptySelection);
    }
  });

  it("picks one, switches, ignores other cards, and builds the message", () => {
    const view = awaiting(1, 1);
    expect(inferredAction(emptySelection, view)).toBeNull();
    expect(buildChoice(emptySelection)).toBeNull();
    const a = click(view, 2);
    expect(a.chosen).toBe(2);
    expect(click(view, 4, a)).toBe(a);
    const b = click(view, 3, a);
    expect(b.chosen).toBe(3);
    expect(inferredAction(b, view)).toBe("safeChooseReveal");
    expect(buildChoice(b)).toEqual({ type: "safeChooseReveal", chosenTargetCardId: 3 });
  });
});
