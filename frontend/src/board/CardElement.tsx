// Board card primitives: CardElement (a card in a CardRow slot), TokenSquare
// (a face-down card's token), RevealedCardMini (a face-up card in the
// TokenRow) and Indicators. All are presentational; state arrives as props.

import type { KeyboardEvent } from "react";
import { valueLabel } from "./cardLabel.ts";
import type { MatchHint } from "./matchHints.ts";
import type { LinkInfo } from "./links.ts";
import { Face, SpecialKind, TokenKind, type CardValue } from "../types.ts";

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
  // Exclusion hint for the selected source card; absent when not applicable.
  match?: MatchHint;
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
  match,
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
      data-match={match}
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

/** A joker link token: "=" (same) or "≠" (different), shared by two adjacent cards. */
export function LinkMarker({ link }: { link: LinkInfo }) {
  const same = link.kind === TokenKind.Same;
  return (
    <div
      className="link-marker"
      data-link={same ? "same" : "different"}
      data-part={link.part}
      aria-label={same ? "Linked as same rank" : "Linked as different rank"}
    >
      {same ? "=" : "≠"}
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

export function Indicators({ used, away = false }: { used: boolean; away?: boolean }) {
  return (
    <div className="indicators">
      <span className="power-indicator" data-power={used ? "consumed" : "available"}>
        {used ? "Double call used" : "Double call available"}
      </span>
      {away && (
        <span className="away-indicator" role="status">
          Away
        </span>
      )}
    </div>
  );
}
