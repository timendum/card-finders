// Table — the four-seat table layout with the center Value_Summary.
//
// The viewing player's hand is pinned to the BOTTOM edge; the three other
// seats are placed at left, top, and right so that clockwise turn order,
// starting from the bottom, visits bottom -> left -> top -> right. That mapping
// is the shared pure seatLayout function. The ValueSummary
// occupies the center slot. Each edge renders a HandArea for the
// seat mapped to it, fed the cards that seat owns from the projected view.
//
// Table is pure/presentational: it derives everything from the PlayerView and
// holds no state. Cards are grouped by owner once per view.

import { useMemo } from "react";
import { HandArea } from "./HandArea.tsx";
import { ValueSummary } from "./ValueSummary.tsx";
import { seatLayout } from "./seatLayout.ts";
import type { PlayerView, SeatID, ViewCard } from "./types.ts";

export interface TableProps {
  /** The viewer's current snapshot. */
  view: PlayerView;
}

/** The four seats and an empty-hand fallback, keyed by seat id. */
type CardsBySeat = Map<SeatID, ViewCard[]>;

/** Group the view's projected cards by owning seat. */
function groupBySeat(cards: ViewCard[]): CardsBySeat {
  const bySeat: CardsBySeat = new Map();
  for (let seat = 0; seat < 4; seat++) {
    bySeat.set(seat, []);
  }
  for (const card of cards) {
    const hand = bySeat.get(card.Owner);
    if (hand) hand.push(card);
  }
  return bySeat;
}

export function Table({ view }: TableProps) {
  const layout = useMemo(() => seatLayout(view.Seat), [view.Seat]);
  const bySeat = useMemo(() => groupBySeat(view.Cards), [view.Cards]);

  const renderHand = (seat: SeatID) => (
    <HandArea
      seat={seat}
      cards={bySeat.get(seat) ?? []}
      isViewer={seat === view.Seat}
      isCurrent={seat === view.CurrentSeat}
    />
  );

  return (
    <div className="table" aria-label="Table">
      <div className="table-top">{renderHand(layout.top)}</div>
      <div className="table-left">{renderHand(layout.left)}</div>
      <div className="table-center">
        <ValueSummary view={view} />
      </div>
      <div className="table-right">{renderHand(layout.right)}</div>
      <div className="table-bottom">{renderHand(layout.bottom)}</div>
    </div>
  );
}
