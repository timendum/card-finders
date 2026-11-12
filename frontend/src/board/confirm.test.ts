import { describe, expect, it } from "vitest";
import { buildChoice, canConfirm, inferredAction } from "./confirm.ts";
import { emptySelection, type SelectionState } from "./selection.ts";
import {
  ActionKind,
  Face,
  JokerState,
  Outcome,
  Phase,
  SpecialKind,
  type PlayerView,
  type ViewCard,
} from "../types.ts";

function card(id: number, owner: number, slot: number): ViewCard {
  return {
    ID: id,
    Owner: owner,
    Slot: slot,
    Facing: Face.Down,
    Value: { Rank: 5, Kind: SpecialKind.None, Suit: 0 },
  } as ViewCard;
}

const cards = [
  card(1, 0, 1),
  card(2, 0, 2),
  card(3, 1, 1),
  card(4, 1, 2),
  card(5, 1, 3),
  card(6, 2, 1),
];

function view(over: Partial<PlayerView> = {}): PlayerView {
  return {
    Seat: 0,
    Phase: Phase.Playing,
    YourTurn: true,
    CurrentSeat: 0,
    PowerUsed: [false, false, false, false],
    Cards: cards,
    Jokers: [2, 4, 9, 11, 12].map((Rank) => ({ Rank, State: JokerState.Ready })),
    ...over,
  } as PlayerView;
}

function selection(over: Partial<SelectionState>): SelectionState {
  return { ...emptySelection, ...over };
}

function responderView(joker: number, targets: number[], outcome: Outcome): PlayerView {
  return view({
    Seat: 1,
    Phase: Phase.AwaitingResponse,
    ResponsePlayer: 1,
    HasResponsePlayer: true,
    LastAction: {
      Seq: 3,
      Actor: 0,
      Kind: joker === 0 ? ActionKind.SafeCall : ActionKind.Joker,
      Targets: targets,
      Outcome: outcome,
      Revealed: [],
      LivesLost: 0,
      Joker: joker,
    },
  });
}

describe("safe-choose-reveal confirm", () => {
  it("is incomplete until a card is chosen, then builds the message", () => {
    const v = responderView(0, [3, 4], Outcome.Hit);
    expect(inferredAction(emptySelection, v)).toBeNull();
    expect(buildChoice(emptySelection)).toBeNull();
    const picked = selection({ chosen: 3 });
    expect(inferredAction(picked, v)).toBe("safeChooseReveal");
    expect(buildChoice(picked)).toEqual({ type: "safeChooseReveal", targetCardIds: [3] });
  });

  it("confirms a swap pick", () => {
    const swap = responderView(3, [], Outcome.AwaitingChoice);
    expect(inferredAction(selection({ chosen: 4 }), swap)).toBe("safeChooseReveal");
  });
});

describe("joker confirm", () => {
  it("confirms an anytime joker off turn once its own cards are picked", () => {
    const off = view({ YourTurn: false });
    const s = selection({ jokerRank: 2, jokerOwn: [1, 2] });
    expect(inferredAction(s, off)).toBe("joker");
    expect(canConfirm(s, off)).toBe(true);
  });

  it("needs all three same-seat targets for Triple call", () => {
    const v = view();
    const partial = selection({ jokerRank: 4, jokerOwn: [1], targets: [3, 4] });
    expect(inferredAction(partial, v)).toBeNull();
    expect(inferredAction({ ...partial, targets: [3, 4, 5] }, v)).toBe("joker");
  });

  it("needs a seat for Pass and a rank for Census", () => {
    const v = view();
    const pass = selection({ jokerRank: 12 });
    expect(canConfirm(pass, v)).toBe(false);
    expect(canConfirm({ ...pass, jokerSeats: [1] }, v)).toBe(true);
    const census = selection({ jokerRank: 9 });
    expect(canConfirm(census, v)).toBe(false);
    expect(canConfirm({ ...census, jokerAsk: 7 }, v)).toBe(true);
  });
});
