import { describe, expect, it } from "vitest";
import { lastRevealedIds } from "./lastReveal.ts";
import type { LastAction } from "../types.ts";

describe("lastRevealedIds", () => {
  it("is empty without a last action", () => {
    expect(lastRevealedIds({}).size).toBe(0);
  });

  it("is empty when the last action revealed nothing", () => {
    const LastAction = { Revealed: [] } as unknown as LastAction;
    expect(lastRevealedIds({ LastAction }).size).toBe(0);
  });

  it("holds exactly the revealed card ids", () => {
    const LastAction = { Revealed: [3, 7] } as unknown as LastAction;
    expect([...lastRevealedIds({ LastAction })]).toEqual([3, 7]);
  });
});
