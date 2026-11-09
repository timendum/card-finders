import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { cardAccessibleName, CardElement, PowerIndicator, TokenSquare } from "./CardElement.tsx";
import { Face, SpecialKind } from "../types.ts";

const king = { Rank: 13, Kind: SpecialKind.None, Suit: 0 };
const goldQueen = { Rank: 12, Kind: SpecialKind.Gold, Suit: 0 };

describe("CardElement", () => {
  it("names value and selectable/selected state", () => {
    expect(cardAccessibleName({ facing: Face.Down, value: king, selectable: true })).toBe(
      "K, selectable",
    );
    expect(cardAccessibleName({ facing: Face.Down, value: king, selected: true })).toBe(
      "K, selected",
    );
  });

  it("renders an opponent's face-down card with no value or kind", () => {
    const html = renderToStaticMarkup(<CardElement facing={Face.Down} />);
    expect(html).toContain('role="button"');
    expect(html).toContain('aria-label="Hidden card, not selectable"');
    expect(html).not.toContain("data-kind");
  });

  it("marks specials by kind and shows the suit-free label", () => {
    const html = renderToStaticMarkup(<CardElement facing={Face.Up} value={goldQueen} />);
    expect(html).toContain('data-kind="gold"');
    expect(html).toContain(">GQ<");
  });
});

describe("TokenSquare / PowerIndicator", () => {
  it("shows the token value label", () => {
    expect(renderToStaticMarkup(<TokenSquare value={king} />)).toContain(">K<");
  });

  it("reflects available vs consumed", () => {
    expect(renderToStaticMarkup(<PowerIndicator used={false} />)).toContain(
      'data-power="available"',
    );
    expect(renderToStaticMarkup(<PowerIndicator used />)).toContain('data-power="consumed"');
  });
});
