// GameLog — scrollable list of public events with no visual clue (wrong calls,
// safe call uses, ...). Newest entry is at the bottom and kept in view.
// Wording: actor first; seats bold with on (call target), to (handover), of
// (owner); cards by "card N"; every call is "called"; "double call" lowercase.
import { useEffect, useRef, type ReactNode } from "react";
import {
  SpecialKind,
  type CardValue,
  type PlayerView,
  JokerCensus,
  JokerDiffer,
  JokerExtraLife,
  JokerMatch,
  JokerPass,
  JokerReverseCall,
  JokerSecondWind,
  JokerSelfMark,
  JokerShield,
  JokerSwap,
  JokerSweep,
  LogEvent,
  LogOutcome,
  type LogEntry,
} from "../types.ts";
import { valueLabel } from "./cardLabel.ts";
import { jokerLabel } from "./jokers.ts";
import { seatName } from "./seatLayout.ts";

type SeatNames = Pick<PlayerView, "Seat" | "Players">;

// A seat shown by its player's name ("You" for the viewer).
function Seat({ view, seat }: { view: SeatNames; seat: number }) {
  return <strong>{seatName(view, seat)}</strong>;
}

// Go serializes an absent seat as null, which the generated optional type misses.
function hasSeat(seat: number | null | undefined): seat is number {
  return seat != null;
}

export function GameLog({
  view,
  onFocusEntry,
}: {
  view: PlayerView;
  // Called with the entry under the mouse, or null when it leaves.
  onFocusEntry?: (entry: LogEntry | null) => void;
}) {
  const endRef = useRef<HTMLLIElement>(null);
  const log = view.Log ?? [];

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ block: "nearest" });
  }, [log.length]);

  return (
    <section className="game-log" aria-label="Game log">
      <ol className="game-log-list">
        {log.map((entry, i) => (
          <li
            key={`${entry.Seq}-${i}`}
            className="game-log-entry"
            onMouseEnter={() => onFocusEntry?.(entry)}
            onMouseLeave={() => onFocusEntry?.(null)}
          >
            {hasSeat(entry.Actor) && (
              <>
                <Seat view={view} seat={entry.Actor} />{" "}
              </>
            )}
            {logText(entry, view)}
          </li>
        ))}
        <li ref={endRef} aria-hidden="true" />
      </ol>
    </section>
  );
}

// "card 3", "cards 3 and 4", "cards 3, 4 and 5": cards are always named by position.
function cardsText(slots: readonly number[]): string {
  if (slots.length < 2) return `card ${slots.join("")}`;
  return `cards ${slots.slice(0, -1).join(", ")} and ${slots.at(slots.length - 1)}`;
}

function outcomePhrase(entry: LogEntry, slots: readonly number[]): string {
  switch (entry.Outcome) {
    case LogOutcome.Hit:
      return slots.length > 1 ? "hit one" : "hit";
    case LogOutcome.HitSeveral:
      return slots.length === 2 ? "hit both" : "hit several";
    case LogOutcome.HitGold:
      return slots.length > 1 ? `hit ${slots.length} gold cards` : "hit a gold card";
    case LogOutcome.Shielded:
      return "hit a gold card, but the shield protected it";
    default:
      return "missed";
  }
}

// A card value as a chip colored like the board tokens of the same kind.
function Value({ value }: { value: CardValue }) {
  const kind =
    value.Kind === SpecialKind.Silver
      ? "silver"
      : value.Kind === SpecialKind.Gold
        ? "gold"
        : "suited";
  return (
    <span className="card-small" data-kind={kind}>
      {valueLabel(value)}
    </span>
  );
}
// "called 7 on Top at cards 3 and 4 and missed".
function callText(entry: LogEntry, view: SeatNames): ReactNode {
  const slots = entry.Slots ?? [];
  const outcome = outcomePhrase(entry, slots);
  const target = entry.Target == null ? null : <Seat view={view} seat={entry.Target} />;
  if (entry.Joker === JokerReverseCall) {
    return (
      <>
        called {cardsText(entry.Sources ?? [])} against {cardsText(slots)} on {target} and {outcome}
      </>
    );
  }
  const value = entry.Value ? <Value value={entry.Value} /> : null;
  const where = entry.Joker === JokerSweep ? "every card" : cardsText(slots);
  return (
    <>
      called {value} on {target} at {where} and {outcome}
    </>
  );
}

function jokerText(entry: LogEntry, view: SeatNames): ReactNode {
  const slots = entry.Slots ?? [];
  const sources = entry.Sources ?? [];
  const target = entry.Target == null ? null : <Seat view={view} seat={entry.Target} />;
  switch (entry.Joker) {
    case JokerDiffer:
      return `put a different token on ${cardsText(slots)}`;
    case JokerMatch:
      return `put a same token on ${cardsText(slots)}`;
    case JokerSelfMark:
      return `put a token on ${cardsText(slots)}`;
    case JokerExtraLife:
      return "gained a life";
    case JokerSecondWind:
      return target ? (
        <>restored the double call of {target}</>
      ) : (
        "found nobody who had spent a double call"
      );
    case JokerCensus: {
      const answer = entry.Outcome === LogOutcome.Yes ? "yes" : "no";
      return (
        <>
          answered {answer} to a face-down {entry.Value ? <Value value={entry.Value} /> : "?"}
        </>
      );
    }
    case JokerShield:
      return "armed the shield";
    case JokerPass:
      return <>passed the turn to {target}</>;
    case JokerSwap:
      return sources.length === 2 ? (
        <>
          swapped {cardsText(slots.slice(0, 1))} for {cardsText(slots.slice(1))} of {target}
        </>
      ) : (
        <>
          offered {cardsText(slots)} to {target} for a swap
        </>
      );
    default:
      return "";
  }
}

export function logText(entry: LogEntry, view: SeatNames): ReactNode {
  switch (entry.Event) {
    case LogEvent.Call:
      return entry.Joker === 0 ? (
        callText(entry, view)
      ) : (
        <>
          used {jokerLabel(entry.Joker)}: {callText(entry, view)}
        </>
      );
    case LogEvent.Choice: {
      const card = cardsText(entry.Slots ?? []);
      return entry.Outcome === LogOutcome.Missed
        ? `chose ${card} to receive the token`
        : `chose ${card} to reveal`;
    }
    case LogEvent.Joker:
      return (
        <>
          used {jokerLabel(entry.Joker)}: {jokerText(entry, view)}
        </>
      );
    case LogEvent.SilverDone:
      return "All silver cards have been revealed";
    default:
      return "";
  }
}
