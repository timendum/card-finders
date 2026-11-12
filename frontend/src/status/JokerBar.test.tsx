import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { JokerBar } from "./JokerBar.tsx";
import { LifeDisplay, ShieldIndicator } from "./StatusPanel.tsx";
import { JokerState, Phase, type PlayerView } from "../types.ts";

const base = {
  Seat: 0,
  Phase: Phase.Playing,
  YourTurn: false,
  Jokers: [
    { Rank: 2, State: JokerState.Inactive },
    { Rank: 7, State: JokerState.Ready },
    { Rank: 10, State: JokerState.Ready },
    { Rank: 13, State: JokerState.Used },
  ],
} as PlayerView;

describe("JokerBar", () => {
  it("marks each joker with its state", () => {
    const html = renderToStaticMarkup(<JokerBar view={base} />);
    expect(html.match(/data-state="inactive"/gu)).toHaveLength(1);
    expect(html.match(/data-state="ready"/gu)).toHaveLength(2);
    expect(html.match(/data-state="used"/gu)).toHaveLength(1);
  });

  it("enables only jokers the viewer can use now", () => {
    // Off turn: Extra life (anytime) is usable, Shield (current player) is not.
    const html = renderToStaticMarkup(<JokerBar view={base} />);
    expect(html.match(/data-usable="true"/gu)).toHaveLength(1);
    const own = renderToStaticMarkup(<JokerBar view={{ ...base, YourTurn: true }} />);
    expect(own.match(/data-usable="true"/gu)).toHaveLength(2);
  });

  it("renders nothing without jokers", () => {
    expect(renderToStaticMarkup(<JokerBar view={{ ...base, Jokers: [] }} />)).toBe("");
  });
});

describe("status panel pieces", () => {
  it("shows lives above the cap", () => {
    expect(renderToStaticMarkup(<LifeDisplay lifePool={5} maxLife={4} />)).toContain("5/4");
  });

  it("shows the Shield flag only when armed", () => {
    expect(renderToStaticMarkup(<ShieldIndicator armed={true} />)).toContain("shield-indicator");
    expect(renderToStaticMarkup(<ShieldIndicator armed={false} />)).toBe("");
  });
});
