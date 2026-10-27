// HandArea — renders one player's hand on the table.
//
// A seat's cards arrive in the PlayerView as ViewCards already projected for
// the viewer: a card the viewer may see carries a non-null Value (it is either
// face-up for everyone or owned by the viewer), a token's public value rides
// along in TokenDisplay for ALL viewers, and a hidden face-down card carries
// neither. HandArea derives the display state purely from (Facing, Token) and
// renders one of three card states:
//
//   face-up            -> the card's true value
//   face-down          -> a blank card back, no value
//   face-down + token  -> the card back overlaid with the token's displayed
//                         value from TokenDisplay
//
// Cards are shown in the shared stable display order (handSort) so repeated
// renders of the same hand are identical. HandArea is a pure presentational
// component: it reads only its props and holds no state.

import { useMemo } from "react";
import { cardLabel } from "./cardLabel.ts";
import { compareViewCards } from "./handSort.ts";
import { Face, type SeatID, type ViewCard } from "./types.ts";

/** The three display states a card can be in, derived from (Facing, Token). */
type CardDisplay =
  | { state: "face-up"; label: string }
  | { state: "face-down" }
  | { state: "token"; label: string };

/**
 * displayFor derives a card's display state from its facing and token, matching
 * the view projection contract: a face-up card shows its Value; a face-down
 * card with a token shows the token's resolved public value (TokenDisplay); a
 * plain face-down card shows a blank back with no leaked value.
 */
export function displayFor(card: ViewCard): CardDisplay {
  if (card.Facing === Face.Up) {
    // Face-up cards always carry a Value; guard defensively anyway.
    return { state: "face-up", label: card.Value ? cardLabel(card.Value) : "" };
  }
  if (card.Token && card.TokenDisplay) {
    // Face-down but tokened: show the token's public value.
    return { state: "token", label: cardLabel(card.TokenDisplay) };
  }
  return { state: "face-down" };
}

export interface HandAreaProps {
  /** The seat whose hand this is, for labelling. */
  seat: SeatID;
  /** This seat's cards, as projected for the viewer. */
  cards: ViewCard[];
  /** True when this seat is the viewer's own hand (bottom edge). */
  isViewer: boolean;
  /** True when it is this seat's turn, to highlight the active player. */
  isCurrent: boolean;
}

/** Render a single card in whichever of the three display states it is in. */
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

/**
 * HandArea lays out one seat's cards in stable display order. The seat label
 * and an active-turn marker give the viewer their bearings around the table.
 */
export function HandArea({ seat, cards, isViewer, isCurrent }: HandAreaProps) {
  // Stable display order; memoized so a re-render with the same
  // cards reference does not re-sort.
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
