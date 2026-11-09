// Board card primitives: CardElement (a card in a CardRow slot), TokenSquare
// (a face-down card's token), RevealedCardMini (a face-up card in the
// TokenRow) and PowerIndicator. All are presentational; state arrives as props.

import type { KeyboardEvent } from "react";
import { valueLabel } from "./cardLabel.ts";
import { Face, SpecialKind, type CardValue } from "../types.ts";

export function kindName(kind: SpecialKind): "suited" | "silver" | "gold" {
  switch (kind) {
    case SpecialKind.Silver:
      return "silver";
    case SpecialKind.Gold:
      return "gold";
    default:
      return "suited";
  }
}

export interface CardElementProps {
  facing: Face;
  // Absent for another player's face-down card: nothing about it is known.
  value?: CardValue;
  own?: boolean;
  selectable?: boolean;
  selected?: boolean;
  onActivate?: () => void;
}

export function cardAccessibleName({
  facing,
  value,
  selectable,
  selected,
}: Pick<CardElementProps, "facing" | "value" | "selectable" | "selected">): string {
  const subject = value
    ? `${valueLabel(value)}${facing === Face.Up ? ", face up" : ""}`
    : "Hidden card";
  const state = selected ? "selected" : selectable ? "selectable" : "not selectable";
  return `${subject}, ${state}`;
}

export function CardElement({
  facing,
  value,
  own = false,
  selectable = false,
  selected = false,
  onActivate,
}: CardElementProps) {
  const activate = () => {
    if (selectable) onActivate?.();
  };
  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" || e.key === " ") {
      // Space would otherwise scroll the page.
      e.preventDefault();
      activate();
    }
  };

  return (
    <div
      className="card-element"
      role="button"
      tabIndex={selectable ? 0 : -1}
      aria-label={cardAccessibleName({ facing, value, selectable, selected })}
      aria-pressed={selectable ? selected : undefined}
      aria-disabled={selectable ? undefined : true}
      data-facing={facing === Face.Up ? "up" : "down"}
      data-own={own}
      data-kind={value ? kindName(value.Kind) : undefined}
      data-selectable={selectable}
      data-selected={selected}
      onClick={activate}
      onKeyDown={onKeyDown}
    >
      {value ? <span className="card-value-label">{valueLabel(value)}</span> : null}
    </div>
  );
}

export function TokenSquare({ value }: { value: CardValue }) {
  return (
    <div
      className="token-square"
      data-kind={kindName(value.Kind)}
      aria-label={`Token ${valueLabel(value)}`}
    >
      {valueLabel(value)}
    </div>
  );
}

export function RevealedCardMini({ value }: { value: CardValue }) {
  return (
    <div
      className="revealed-card-mini"
      data-kind={kindName(value.Kind)}
      aria-label={`Revealed ${valueLabel(value)}`}
    >
      {valueLabel(value)}
    </div>
  );
}

export function PowerIndicator({ used }: { used: boolean }) {
  return (
    <div className="power-indicator" data-power={used ? "consumed" : "available"}>
      {used ? "Safe call used" : "Safe call available"}
    </div>
  );
}
