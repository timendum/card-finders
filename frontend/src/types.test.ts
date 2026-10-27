import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  Face,
  TokenKindValue,
  Phase,
  Result,
  SpecialKind,
  type ClientMessage,
  type PlayerView,
  type ServerMessage,
} from "./types.ts";

// This is a scaffold smoke test: it confirms the shared types module, Vitest,
// and fast-check are all wired up. The real pure-logic tests arrive with the
// comparator (22.2) and other follow-up tasks.
describe("shared types scaffold", () => {
  it("exposes the engine enum integer values the backend serializes", () => {
    expect(SpecialKind.None).toBe(0);
    expect(SpecialKind.Silver).toBe(1);
    expect(SpecialKind.Gold).toBe(2);
    expect(Face.Down).toBe(0);
    expect(Face.Up).toBe(1);
    expect(Phase.Waiting).toBe(0);
    expect(Phase.Ended).toBe(5);
    expect(Result.None).toBe(0);
    expect(Result.Loss).toBe(2);
    expect(TokenKindValue.NaturalIdentity).toBe(1);
  });

  it("narrows the server message union on `type`", () => {
    const view: PlayerView = {
      Seat: 0,
      Phase: Phase.Waiting,
      Level: { Number: 1, Name: "Default", SilverCount: 0, GoldCount: 0, StartLives: 4 },
      SpecialPool: { Silvers: [], Golds: [] },
      LifePool: 4,
      MaxLife: 4,
      CurrentSeat: 0,
      YourTurn: false,
      PowerUsed: [false, false, false, false],
      Cards: [],
      Result: Result.None,
      SeatsFilled: 1,
    };
    const msg: ServerMessage = { type: "state", view };
    expect(msg.type === "state" ? msg.view.Seat : -1).toBe(0);
  });

  it("builds every client message variant for any card id (fast-check)", () => {
    fc.assert(
      fc.property(fc.nat(65535), fc.nat(65535), (a, b) => {
        const msgs: ClientMessage[] = [
          { type: "openingReveal", cardId: a },
          { type: "guess", sourceCardId: a, targetCardId: b },
          { type: "safeCall", sourceCardId: a, targetCardId: b, targetCardId2: a },
          { type: "safeChooseReveal", chosenTargetCardId: b },
          { type: "selfReveal", value: "silver" },
        ];
        return msgs.every((m) => typeof m.type === "string");
      }),
    );
  });
});
