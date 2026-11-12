import { describe, expect, it } from "vitest";
import {
  chooseOptions,
  emptySelection,
  isSelectable,
  selectionReducer,
  type SelectionEvent,
  type SelectionState,
} from "./selection.ts";
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

function run(v: PlayerView, events: SelectionEvent[]): SelectionState {
  return events.reduce((s, e) => selectionReducer(s, v, e), emptySelection);
}

describe("joker selection", () => {
  it("only starts a usable joker and toggles it off on a second pick", () => {
    const off = view({ YourTurn: false });
    expect(selectionReducer(emptySelection, off, { type: "pickJoker", rank: 4 })).toBe(
      emptySelection,
    );
    const started = run(view(), [{ type: "pickJoker", rank: 2 }]);
    expect(started.jokerRank).toBe(2);
    expect(selectionReducer(started, view(), { type: "pickJoker", rank: 2 }).jokerRank).toBeNull();
  });

  it("lets an anytime joker be set up off turn", () => {
    const off = view({ YourTurn: false });
    const s = run(off, [
      { type: "pickJoker", rank: 2 },
      { type: "clickCard", cardId: 1 },
      { type: "clickCard", cardId: 2 },
    ]);
    expect(isSelectable(off, cards[0], s)).toBe(true);
    expect(isSelectable(off, cards[2], s)).toBe(false);
    expect(s.jokerOwn).toEqual([1, 2]);
  });

  it("collects three same-seat targets for Triple call and restarts on another seat", () => {
    const v = view();
    const s = run(v, [
      { type: "pickJoker", rank: 4 },
      { type: "clickCard", cardId: 1 },
      { type: "clickCard", cardId: 3 },
      { type: "clickCard", cardId: 4 },
    ]);
    expect(s.targets).toEqual([3, 4]);
    const done = selectionReducer(s, v, { type: "clickCard", cardId: 5 });
    expect(done.targets).toEqual([3, 4, 5]);
    const restarted = selectionReducer(done, v, { type: "clickCard", cardId: 6 });
    expect(restarted.targets).toEqual([6]);
  });

  it("records the seat for Pass and the rank for Census", () => {
    const v = view();
    const pass = run(v, [
      { type: "pickJoker", rank: 12 },
      { type: "pickJokerSeat", seat: 1 },
    ]);
    expect(pass.jokerSeats).toEqual([1]);
    const census = run(v, [
      { type: "pickJoker", rank: 9 },
      { type: "pickJokerAsk", rank: 7 },
    ]);
    expect(census.jokerAsk).toBe(7);
  });

  it("clears the joker flow on clear", () => {
    const s = run(view(), [{ type: "pickJoker", rank: 12 }, { type: "clear" }]);
    expect(s.jokerRank).toBeNull();
  });
});

describe("swap pending choice", () => {
  const swap = view({
    Seat: 1,
    Phase: Phase.AwaitingResponse,
    ResponsePlayer: 1,
    HasResponsePlayer: true,
    LastAction: {
      Seq: 3,
      Actor: 0,
      Kind: ActionKind.Joker,
      Targets: [],
      Outcome: Outcome.AwaitingChoice,
      Revealed: [],
      LivesLost: 0,
      Joker: 3,
    },
  });

  it("offers the responder every own face-down card", () => {
    expect(chooseOptions(swap)).toEqual([3, 4, 5]);
    const picked = selectionReducer(emptySelection, swap, { type: "clickCard", cardId: 4 });
    expect(picked.chosen).toBe(4);
  });

  it("offers nothing to other seats", () => {
    expect(chooseOptions({ ...swap, Seat: 2 })).toEqual([]);
  });
});
