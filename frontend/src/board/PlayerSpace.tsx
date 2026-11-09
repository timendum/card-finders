import { PowerIndicator } from "./CardElement.tsx";
import { CardRow, TokenRow, type CardRowProps } from "./CardRows.tsx";
import type { SeatID } from "../types.ts";

export type Edge = "top" | "left" | "right" | "bottom";

export interface PlayerSpaceProps extends Omit<CardRowProps, "own"> {
  seat: SeatID;
  edge: Edge;
  isViewer: boolean;
  highlighted: boolean;
  // Marks the highlight as the pending responder rather than the acting seat.
  responder: boolean;
  powerUsed: boolean;
}

export function PlayerSpace({
  seat,
  edge,
  isViewer,
  highlighted,
  responder,
  powerUsed,
  cards,
  isSelectable,
  isSelected,
  onActivate,
}: PlayerSpaceProps) {
  const reversed = edge === "top" || edge === "right";
  return (
    <section
      className="player-space"
      aria-label={isViewer ? "Your cards" : `Seat ${seat}`}
      data-edge={edge}
      data-seat={seat}
      data-current={highlighted && !responder}
      data-responder={highlighted && responder}
    >
      <TokenRow cards={cards} reversed={reversed} />
      <CardRow
        cards={cards}
        own={isViewer}
        reversed={reversed}
        isSelectable={isSelectable}
        isSelected={isSelected}
        onActivate={onActivate}
      />
      <PowerIndicator used={powerUsed} />
    </section>
  );
}
