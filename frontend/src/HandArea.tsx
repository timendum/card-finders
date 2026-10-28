// HandArea — pure presentational component rendering one seat's hand.
// Derives each card's display state from (Facing, Token) on the already
// viewer-projected ViewCards, in shared stable display order (handSort).

import { useMemo } from "react";
import { cardLabel } from "./cardLabel.ts";
import { compareViewCards } from "./handSort.ts";
import { Face, type SeatID, type ViewCard } from "./types.ts";

type CardDisplay =
  | { state: "face-up"; label: string }
  | { state: "face-down" }
  | { state: "token"; label: string };

export function displayFor(card: ViewCard): CardDisplay {
  if (card.Facing === Face.Up) {
    // Face-up cards always carry a Value; guard defensively anyway.
    return { state: "face-up", label: card.Value ? cardLabel(card.Value) : "" };
  }
  if (card.Token && card.TokenDisplay) {
    return { state: "token", label: cardLabel(card.TokenDisplay) };
  }
  return { state: "face-down" };
}

export interface HandAreaProps {
  seat: SeatID;
  cards: ViewCard[];
  /** True when this seat is the viewer's own hand (bottom edge). */
  isViewer: boolean;
  isCurrent: boolean;
}

function CardView({ card }: { card: ViewCard }) {
  const display = displayFor(card);
  switch (display.state) {
    case "face-up":
      return (
        <li className="card card-face-up" data-facing="up">
          <span className="card-value">{display.label}</span>
        </li>
      );
    case "token":
      return (
        <li className="card card-face-down card-token" data-facing="down" data-token="true">
          <span className="card-back" aria-hidden="true" />
          <span className="card-token-value">{display.label}</span>
        </li>
      );
    default:
      return (
        <li className="card card-face-down" data-facing="down">
          <span className="card-back" aria-hidden="true" />
        </li>
      );
  }
}

export function HandArea({ seat, cards, isViewer, isCurrent }: HandAreaProps) {
  const ordered = useMemo(() => [...cards].sort(compareViewCards), [cards]);

  return (
    <section
      className="hand-area"
      data-viewer={isViewer}
      data-current={isCurrent}
      aria-label={isViewer ? `Your hand (seat ${seat})` : `Seat ${seat} hand`}
    >
      <header className="hand-header">
        {isViewer ? "You" : `Seat ${seat}`}
        {isCurrent && <span className="turn-marker"> • turn</span>}
      </header>
      <ul className="hand-cards">
        {ordered.map((card) => (
          <CardView key={card.ID} card={card} />
        ))}
      </ul>
    </section>
  );
}
