import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { buildValueSummary, ValueSummary } from "./ValueSummary.tsx";
import { Face, Phase, Result, SpecialKind, type PlayerView, type ViewCard } from "../types.ts";

const silver = (rank: number) => ({ Rank: rank, Suit: 0, Kind: SpecialKind.Silver });

function viewWith(silvers: number[], inPlay: number, cards: ViewCard[] = []): PlayerView {
  return {
    Seat: 0,
    Phase: Phase.Playing,
    Level: {
      Number: 1,
      Name: "L",
      SilverCount: 1,
      GoldCount: 0,
      ExtraSilver: 1,
      ExtraGold: 0,
      StartLives: 4,
    },
    SpecialPool: {
      Silvers: silvers.map((c) => silver(c)),
      Golds: [],
      SilverInPlay: inPlay,
      GoldInPlay: 0,
    },
    LifePool: 4,
    MaxLife: 4,
    CurrentSeat: 0,
    YourTurn: true,
    PowerUsed: [false, false, false, false],
    Connected: [true, true, true, true],
    Cards: cards,
    Result: Result.None,
    SeatsFilled: 4,
    ResponsePlayer: 0,
    HasResponsePlayer: false,
    Log: [],
  };
}

const faceUpSilver4: ViewCard = {
  ID: 1,
  Owner: 1,
  Facing: Face.Up,
  Value: silver(4),
  Slot: 0,
};

function silverRow(html: string, rank: number): string {
  const rows = html.split("<li ").filter((r) => r.includes('data-kind="silver"'));
  const idx = [4, 6].indexOf(rank);
  return rows[idx] ?? "";
}

describe("ValueSummary", () => {
  it("shows 0/1 with ? inside <strong> for a possible decoy", () => {
    const view = viewWith([4, 6], 1);
    const rows = buildValueSummary(view).filter((r) => r.Kind === SpecialKind.Silver);
    expect(rows.map((r) => [r.Found, r.Total, r.Uncertain])).toEqual([
      [0, 1, true],
      [0, 1, true],
    ]);
    const row = silverRow(renderToStaticMarkup(<ValueSummary view={view} />), 4);
    expect(row).toContain('data-uncertain="true"');
    expect(row).toMatch(/0\/1(<!-- -->)?<strong[^>]*>\?<\/strong>/u);
  });

  it("shows 1/1 for a found special, without ?", () => {
    const view = viewWith([4, 6], 1, [faceUpSilver4]);
    const row = silverRow(renderToStaticMarkup(<ValueSummary view={view} />), 4);
    expect(row).toContain("1");
    expect(row).toMatch(/1\/<!-- -->1|1\/1/u);
    expect(row).not.toContain("<strong");
    expect(row).toContain('data-count="complete"');
  });

  it("shows completed 0/0 for the remaining entries once all in-play specials are found", () => {
    const view = viewWith([4, 6], 1, [faceUpSilver4]);
    const [, other] = buildValueSummary(view).filter((r) => r.Kind === SpecialKind.Silver);
    expect([other.Found, other.Total, other.Uncertain]).toEqual([0, 0, false]);
    const row = silverRow(renderToStaticMarkup(<ValueSummary view={view} />), 6);
    expect(row).toMatch(/0\/<!-- -->0|0\/0/u);
    expect(row).not.toContain("<strong");
    expect(row).toContain('data-count="complete"');
  });
});
