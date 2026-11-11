import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { PromptPanel } from "./PromptPanel.tsx";
import { emptySelection } from "./selection.ts";
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
} from "../types.ts";

function card(id: number, owner: number, rank: number, withValue: boolean): ViewCard {
  return {
    ID: id,
    Owner: owner,
    Facing: Face.Down,
    Slot: id,
    ...(withValue ? { Value: { Rank: rank, Kind: SpecialKind.None, Suit: 0 } } : {}),
  } as ViewCard;
}

function playingView(overrides: Partial<PlayerView> = {}): PlayerView {
  return {
    Seat: 0,
    Phase: Phase.Playing,
    Level: { Number: 1, Name: "L", SilverCount: 0, GoldCount: 0, ExtraSilver: 0, ExtraGold: 0, StartLives: 4 },
    SpecialPool: { Silvers: [], Golds: [], SilverInPlay: 0, GoldInPlay: 0 },
    LifePool: 4,
    MaxLife: 4,
    CurrentSeat: 0,
    YourTurn: true,
    PowerUsed: [false, false, false, false],
    Cards: [card(1, 0, 5, true), card(2, 1, 7, false), card(3, 1, 8, false)],
    Result: Result.None,
    SeatsFilled: 4,
    ResponsePlayer: 0,
    HasResponsePlayer: false,
    ...overrides,
  } as PlayerView;
}

const noop = () => {};

describe("PromptPanel", () => {
  it("disables confirm with no selection and lists actions", () => {
    const html = renderToStaticMarkup(
      <PromptPanel
        view={playingView()}
        selection={emptySelection}
        error={null}
        onConfirm={noop}
        onPickSelfValue={noop}
      />,
    );
    expect(html).toMatch(/<button[^>]*disabled=""[^>]*>Confirm<\/button>/u);
    expect(html).toContain("Guess");
    expect(html).toContain("Safe call");
  });

  it("labels confirm for the inferred move and shows the error", () => {
    const html = renderToStaticMarkup(
      <PromptPanel
        view={playingView()}
        selection={{ ...emptySelection, source: 1, targetSeat: 1, targets: [2] }}
        error={{ code: "invalid_target", message: "Bad target" }}
        onConfirm={noop}
        onPickSelfValue={noop}
      />,
    );
    expect(html).toContain("Confirm guess");
    expect(html).not.toMatch(/disabled=""/u);
    expect(html).toContain("Bad target");
  });

  it("shows who is acting and no confirm when not the viewer's turn", () => {
    const html = renderToStaticMarkup(
      <PromptPanel
        view={playingView({ YourTurn: false, CurrentSeat: 2 })}
        selection={emptySelection}
        error={null}
        onConfirm={noop}
        onPickSelfValue={noop}
      />,
    );
    expect(html).toContain("the top player");
    expect(html).not.toContain("prompt-confirm");
  });
});

describe("PromptPanel safe-choose-reveal", () => {
  const base = {
    Phase: Phase.AwaitingResponse,
    HasResponsePlayer: true,
    ResponsePlayer: 1,
    LastAction: {
      Seq: 1,
      Actor: 0,
      Kind: ActionKind.SafeCall,
      Targets: [2, 3],
      Outcome: Outcome.Hit,
      Revealed: [],
      LivesLost: 0,
    } as LastAction,
  };
  const render = (view: PlayerView, chosen: number | null = null, error = null) =>
    renderToStaticMarkup(
      <PromptPanel
        view={view}
        selection={{ ...emptySelection, chosen }}
        error={error}
        onConfirm={noop}
        onPickSelfValue={noop}
      />,
    );

  it("guides the responder and enables confirm once a card is picked", () => {
    const view = playingView({ ...base, Seat: 1, YourTurn: false, CurrentSeat: 0 });
    expect(render(view)).toContain("Choose a card to reveal");
    expect(render(view)).toMatch(/disabled=""/u);
    const html = render(view, 2, { code: "invalid_target", message: "Nope" } as never);
    expect(html).toContain("Confirm choice");
    expect(html).not.toMatch(/disabled=""/u);
    expect(html).toContain("Nope");
  });

  it("tells others the responder is choosing, even the caller on their turn", () => {
    const html = render(playingView({ ...base, Seat: 0, YourTurn: true, CurrentSeat: 0 }));
    expect(html).toContain("The left player is choosing");
    expect(html).not.toContain("prompt-confirm");
  });
});

describe("PromptPanel result", () => {
  const last = (over: Record<string, unknown> = {}) =>
    ({
      Seq: 5,
      Actor: 0,
      Kind: ActionKind.Guess,
      Targets: [2],
      Outcome: Outcome.Miss,
      Revealed: [],
      LivesLost: 1,
      ...over,
    }) as LastAction;
  const render = (view: PlayerView, lastShownSeq: number | null = null) =>
    renderToStaticMarkup(
      <PromptPanel
        view={view}
        selection={{ ...emptySelection, lastShownSeq }}
        error={null}
        onConfirm={noop}
        onPickSelfValue={noop}
      />,
    );

  it("shows a miss with lives lost for the viewer's own guess", () => {
    expect(render(playingView({ LastAction: last() }))).toContain("Miss. You lost 1 life.");
  });

  it("shows a hit for the viewer's own safe call", () => {
    const html = render(
      playingView({
        LastAction: last({ Kind: ActionKind.SafeCall, Outcome: Outcome.Hit, LivesLost: 0 }),
      }),
    );
    expect(html).toContain("Hit!");
  });

  it("hides once dismissed, but a newer Seq shows again", () => {
    const view = playingView({ LastAction: last() });
    expect(render(view, 5)).not.toContain("prompt-result");
    expect(render(playingView({ LastAction: last({ Seq: 6 }) }), 5)).toContain("prompt-result");
  });

  it("ignores other seats' actions and non-guess kinds", () => {
    expect(render(playingView({ LastAction: last({ Actor: 2 }) }))).not.toContain("prompt-result");
    expect(
      render(playingView({ LastAction: last({ Kind: ActionKind.SelfReveal }) })),
    ).not.toContain("prompt-result");
  });

  it("reshows the latest action on a fresh mount", () => {
    expect(render(playingView({ LastAction: last() }), null)).toContain("prompt-result");
  });
});
