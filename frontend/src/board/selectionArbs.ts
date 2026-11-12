// Shared fast-check generators for the selection and confirm property tests.
import fc from "fast-check";
import {
  ActionKind,
  Face,
  Outcome,
  Phase,
  Result,
  SpecialKind,
  type CardValue,
  type PlayerView,
  type ViewCard,
} from "../types.ts";
import type { SelectionEvent } from "./selection.ts";

export const RUNS = { numRuns: 200 };

export const SEAT = 0;
export const CARD_COUNT = 16;

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

export const viewArb: fc.Arbitrary<PlayerView> = fc
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

export const eventsArb = fc.array(eventArb, { minLength: 0, maxLength: 25 });
