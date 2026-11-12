// One seat's CardRow and TokenRow. Both lay out the same fixed slots 1..N so a
// card and its token or revealed mini line up column for column.

import { CardElement, LinkMarker, RevealedCardMini, TokenSquare } from "./CardElement.tsx";
import { linkRoles } from "./links.ts";
import type { MatchHint } from "./matchHints.ts";
import { slotted } from "./slots.ts";
import { Face, type CardID, type ViewCard } from "../types.ts";

function ordered<T>(items: T[], reversed: boolean): T[] {
  return reversed ? [...items].reverse() : items;
}

export interface CardRowProps {
  // The seat's whole hand, in any order.
  cards: readonly ViewCard[];
  own?: boolean;
  isSelectable?: (card: ViewCard) => boolean;
  isSelected?: (card: ViewCard) => boolean;
  matchHint?: (card: ViewCard) => MatchHint | undefined;
  onActivate?: (id: CardID) => void;
  // Lay slots out in descending order (top and right seats read ascending counterclockwise).
  reversed?: boolean;
}

export function CardRow({
  cards,
  own = false,
  isSelectable,
  isSelected,
  matchHint,
  onActivate,
  reversed = false,
}: CardRowProps) {
  return (
    <div className="card-row" role="group">
      {ordered(slotted(cards), reversed).map((card, i) =>
        // A face-up card has moved to the TokenRow and leaves its slot empty.
        card && card.Facing === Face.Down ? (
          <CardElement
            key={i}
            facing={card.Facing}
            value={card.Value}
            own={own}
            selectable={isSelectable?.(card) ?? false}
            selected={isSelected?.(card) ?? false}
            match={matchHint?.(card)}
            onActivate={() => onActivate?.(card.ID)}
          />
        ) : (
          <div key={i} className="card-slot-empty" aria-hidden="true" />
        ),
      )}
    </div>
  );
}

export function TokenRow({
  cards,
  reversed = false,
}: {
  cards: readonly ViewCard[];
  reversed?: boolean;
}) {
  const links = linkRoles(cards);
  return (
    <div className="token-row" role="group">
      {ordered(slotted(cards), reversed).map((card, i) => {
        if (card?.Facing === Face.Up && card.Value) {
          return <RevealedCardMini key={i} value={card.Value} />;
        }
        if (card?.Facing === Face.Down && card.TokenDisplay) {
          return <TokenSquare key={i} value={card.TokenDisplay} />;
        }
        const link = card ? links.get(card.ID) : undefined;
        if (link) return <LinkMarker key={i} link={link} />;
        return <div key={i} aria-hidden="true" />;
      })}
    </div>
  );
}
