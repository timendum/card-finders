import { useMemo, type CSSProperties, type ReactNode } from "react";
import { PlayerSpace, type Edge } from "./PlayerSpace.tsx";
import { seatLayout } from "./seatLayout.ts";
import type { MatchHint } from "./matchHints.ts";
import { lastRevealedIds } from "./lastReveal.ts";
import { cardsBySeat } from "./slots.ts";
import { Phase, type CardID, type PlayerView, type SeatID, type ViewCard } from "../types.ts";

export interface TableProps {
  view: PlayerView;
  // Center column slots, top and bottom.
  status?: ReactNode;
  prompt?: ReactNode;
  log?: ReactNode;
  isSelectable?: (card: ViewCard) => boolean;
  isSelected?: (card: ViewCard) => boolean;
  matchHint?: (card: ViewCard) => MatchHint | undefined;
  onActivate?: (id: CardID) => void;
}

export function Table({
  view,
  status,
  prompt,
  log,
  isSelectable,
  isSelected,
  matchHint,
  onActivate,
}: TableProps) {
  const layout = useMemo(() => seatLayout(view.Seat), [view.Seat]);
  const bySeat = useMemo(() => cardsBySeat(view.Cards), [view.Cards]);

  const maxHand = useMemo(
    () => Math.max(0, ...Array.from(bySeat.values(), (h) => h.length)),
    [bySeat],
  );

  const revealed = lastRevealedIds(view);
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
      connected={view.Connected[seat] ?? true}
      cards={bySeat.get(seat) ?? []}
      isSelectable={isSelectable}
      isSelected={isSelected}
      matchHint={matchHint}
      onActivate={onActivate}
      isHighlighted={(card) => revealed.has(card.ID)}
    />
  );

  return (
    <div
      className="table-region"
      aria-label="Table"
      data-opening={view.Phase === Phase.Opening}
      style={{ "--slots": Math.max(11, maxHand) } as CSSProperties}
    >
      {renderSpace(layout.top, "top")}
      {renderSpace(layout.left, "left")}
      <div className="table-center">
        <div className="table-center-status">{status}</div>
        <div className="table-center-bottom">
          <div className="table-center-prompt">{prompt}</div>
          {log}
        </div>
      </div>
      {renderSpace(layout.right, "right")}
      {renderSpace(layout.bottom, "bottom")}
    </div>
  );
}
