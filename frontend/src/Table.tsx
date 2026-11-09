import { useMemo, type ReactNode } from "react";
import { PlayerSpace, type Edge } from "./PlayerSpace.tsx";
import { seatLayout } from "./seatLayout.ts";
import { cardsBySeat } from "./slots.ts";
import { Phase, type CardID, type PlayerView, type SeatID, type ViewCard } from "./types.ts";

export interface TableProps {
  view: PlayerView;
  // Center column slots, top and bottom.
  status?: ReactNode;
  prompt?: ReactNode;
  isSelectable?: (card: ViewCard) => boolean;
  isSelected?: (card: ViewCard) => boolean;
  onActivate?: (id: CardID) => void;
}

export function Table({ view, status, prompt, isSelectable, isSelected, onActivate }: TableProps) {
  const layout = useMemo(() => seatLayout(view.Seat), [view.Seat]);
  const bySeat = useMemo(() => cardsBySeat(view.Cards), [view.Cards]);

  const awaiting = view.Phase === Phase.AwaitingResponse && view.HasResponsePlayer;
  const highlightedSeat = awaiting ? view.ResponsePlayer : view.CurrentSeat;

  const renderSpace = (seat: SeatID, edge: Edge) => (
    <PlayerSpace
      key={edge}
      seat={seat}
      edge={edge}
      isViewer={seat === view.Seat}
      highlighted={seat === highlightedSeat}
      responder={awaiting}
      powerUsed={view.PowerUsed[seat] ?? false}
      cards={bySeat.get(seat) ?? []}
      isSelectable={isSelectable}
      isSelected={isSelected}
      onActivate={onActivate}
    />
  );

  return (
    <div className="table-region" aria-label="Table">
      {renderSpace(layout.top, "top")}
      {renderSpace(layout.left, "left")}
      <div className="table-center">
        <div className="table-center-status">{status}</div>
        <div className="table-center-prompt">{prompt}</div>
      </div>
      {renderSpace(layout.right, "right")}
      {renderSpace(layout.bottom, "bottom")}
    </div>
  );
}
