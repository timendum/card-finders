// Table — four-seat layout (edge placement via seatLayout) with the center
// Value_Summary. Pure/presentational: derived entirely from the PlayerView.

import { useMemo } from "react";
import { HandArea } from "./HandArea.tsx";
import { ValueSummary } from "./ValueSummary.tsx";
import { seatLayout } from "./seatLayout.ts";
import type { PlayerView, SeatID, ViewCard } from "./types.ts";

export interface TableProps {
  view: PlayerView;
}

type CardsBySeat = Map<SeatID, ViewCard[]>;

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
