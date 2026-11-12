import { describe, expect, it } from "vitest";
import { linkRoles } from "./links.ts";
import { Face, TokenKind, type ViewCard } from "../types.ts";

function card(id: number, slot: number, kind?: TokenKind, facing = Face.Down): ViewCard {
  return {
    ID: id,
    Owner: 1,
    Slot: slot,
    Facing: facing,
    Token: kind === undefined ? undefined : { Displayed: kind },
  } as ViewCard;
}

describe("linkRoles", () => {
  it("pairs adjacent same-kind link tokens left to right", () => {
    const roles = linkRoles([
      card(4, 4, TokenKind.Same),
      card(1, 1, TokenKind.Different),
      card(2, 2, TokenKind.Different),
      card(3, 3, TokenKind.Same),
    ]);
    expect(roles.get(1)).toEqual({ kind: TokenKind.Different, part: "first" });
    expect(roles.get(2)).toEqual({ kind: TokenKind.Different, part: "second" });
    expect(roles.get(3)).toEqual({ kind: TokenKind.Same, part: "first" });
    expect(roles.get(4)).toEqual({ kind: TokenKind.Same, part: "second" });
  });

  it("does not pair across a gap, a face-up card or a different kind", () => {
    const roles = linkRoles([
      card(1, 1, TokenKind.Same),
      card(2, 2, TokenKind.Different),
      card(3, 4, TokenKind.Different),
      card(4, 5, TokenKind.Different, Face.Up),
    ]);
    expect(roles.get(1)?.part).toBe("single");
    expect(roles.get(2)?.part).toBe("single");
    expect(roles.get(3)?.part).toBe("single");
    expect(roles.has(4)).toBe(false);
  });

  it("ignores natural-identity tokens and untokened cards", () => {
    const roles = linkRoles([card(1, 1, TokenKind.NaturalIdentity), card(2, 2)]);
    expect(roles.size).toBe(0);
  });
});
