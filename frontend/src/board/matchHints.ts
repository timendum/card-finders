// Pure exclusion hints for a guess / safe call: given the viewer's chosen source
// card, which of the other players' face-down cards can still match its value.
// Only public facts are used (value-sorted fixed hand order, tokens, face-up
// cards, the viewer's own hand, the Special_Pool). Hints never block a click;
// the engine stays the only authority.

import {
  Face,
  Phase,
  SpecialKind,
  type CardID,
  type CardValue,
  type PlayerView,
  type ViewCard,
} from "../types.ts";

/** "sure": a token already shows the called value. "excluded": cannot match. */
export type MatchHint = "possible" | "sure" | "excluded";

const SUFFIX = { [SpecialKind.None]: 0, [SpecialKind.Silver]: 3, [SpecialKind.Gold]: 6 };

/** Sort_Value times ten, so comparisons stay integer. */
function sortValue(v: CardValue): number {
  return v.Rank * 10 + SUFFIX[v.Kind];
}

/** A card's publicly known value: face-up value or token value. */
function knownValue(card: ViewCard): CardValue | undefined {
  return card.Facing === Face.Up ? card.Value : card.TokenDisplay;
}

function handOf(view: PlayerView, seat: number): ViewCard[] {
  return view.Cards.filter((c) => c.Owner === seat).sort((a, b) => a.Slot - b.Slot);
}

/** Known sort values to the left of and right of index i within a sorted hand. */
function neighbourBounds(hand: ViewCard[], i: number): { lo: number; hi: number } {
  let lo = -Infinity;
  let hi = Infinity;
  hand.forEach((c, j) => {
    const v = j === i ? undefined : knownValue(c);
    if (!v) return;
    if (j < i) lo = Math.max(lo, sortValue(v));
    else hi = Math.min(hi, sortValue(v));
  });
  return { lo, hi };
}

/** Whole-board count of cards matching `test`, as an upper bound of what other hands can hold. */
function othersCapacity(view: PlayerView, test: (sv: number) => boolean, skipSeat: number): number {
  let total = 0;
  for (let rank = 2; rank <= 13; rank++) if (test(rank * 10)) total += 4;
  for (const e of view.SpecialPool.Silvers) if (test(sortValue(e))) total++;
  for (const e of view.SpecialPool.Golds) if (test(sortValue(e))) total++;
  for (const c of view.Cards) {
    if (c.Owner === skipSeat) continue;
    const v = c.Owner === view.Seat ? c.Value : knownValue(c);
    if (v && test(sortValue(v))) total--;
  }
  return total;
}

/**
 * Every value an unidentified face-down card in another hand could still have:
 * suited ranks with a copy not yet accounted for, and pool specials (real or
 * decoy) whose rank is not already held, face-up or tokened.
 */
function feasibleValues(view: PlayerView): CardValue[] {
  const suitedKnown = new Map<number, number>();
  const silverTaken = new Set<number>();
  const goldTaken = new Set<number>();
  for (const c of view.Cards) {
    const v = c.Owner === view.Seat ? c.Value : knownValue(c);
    if (!v) continue;
    if (v.Kind === SpecialKind.Silver) silverTaken.add(v.Rank);
    else if (v.Kind === SpecialKind.Gold) goldTaken.add(v.Rank);
    else suitedKnown.set(v.Rank, (suitedKnown.get(v.Rank) ?? 0) + 1);
  }
  const out: CardValue[] = [];
  for (let rank = 2; rank <= 13; rank++) {
    if ((suitedKnown.get(rank) ?? 0) < 4) out.push({ Rank: rank, Kind: SpecialKind.None, Suit: 0 });
  }
  for (const e of view.SpecialPool.Silvers) if (!silverTaken.has(e.Rank)) out.push(e);
  for (const e of view.SpecialPool.Golds) if (!goldTaken.has(e.Rank)) out.push(e);
  return out;
}

/**
 * matchHints returns a hint for each other-player face-down card, or null when
 * hints do not apply (not the viewer's turn, no usable source card selected).
 */
export function matchHints(
  view: PlayerView,
  sourceId: CardID | null,
): Map<CardID, MatchHint> | null {
  if (sourceId === null || view.Phase !== Phase.Playing || !view.YourTurn) return null;
  const source = view.Cards.find((c) => c.ID === sourceId);
  const called = source?.Value;
  if (!source || !called || source.Owner !== view.Seat || source.Facing !== Face.Down) return null;
  if (called.Kind === SpecialKind.Gold) return null;

  const isSilver = called.Kind === SpecialKind.Silver;
  const sv = sortValue(called);
  const sameValue = (v: CardValue) =>
    isSilver
      ? v.Kind === SpecialKind.Silver
      : v.Kind === SpecialKind.None && v.Rank === called.Rank;

  // Copies of the called value neither face-up nor in the viewer's hand, and how many carry tokens.
  const total = isSilver ? view.SpecialPool.SilverInPlay : 4;
  let accounted = 0;
  let tokened = 0;
  for (const c of view.Cards) {
    if (c.Facing === Face.Up && c.Value && sameValue(c.Value)) accounted++;
    else if (c.Owner === view.Seat && c.Value && sameValue(c.Value)) accounted++;
    else if (c.Owner !== view.Seat && c.TokenDisplay && sameValue(c.TokenDisplay)) tokened++;
  }
  const remaining = total - accounted;
  const feasible = feasibleValues(view);

  const hintFor = (hand: ViewCard[], i: number): MatchHint => {
    const card = hand[i]!;
    if (card.TokenDisplay) return sameValue(card.TokenDisplay) ? "sure" : "excluded";
    if (remaining <= 0 || tokened === remaining) return "excluded";
    const { lo, hi } = neighbourBounds(hand, i);
    const inRange = feasible.filter((v) => sortValue(v) >= lo && sortValue(v) <= hi);
    const matching = inRange.filter((v) => sameValue(v));
    if (matching.length === 0) return "excluded";
    if (!isSilver) {
      // Everything right of the card is >= sv, everything left <= sv.
      if (othersCapacity(view, (x) => x >= sv, card.Owner) < hand.length - i) return "excluded";
      if (othersCapacity(view, (x) => x <= sv, card.Owner) < i + 1) return "excluded";
    }
    // Nothing else the card could be: the call is certain.
    return matching.length === inRange.length ? "sure" : "possible";
  };

  const hints = new Map<CardID, MatchHint>();
  const hands: ViewCard[][] = [];
  for (let seat = 0; seat < 4; seat++) {
    if (seat === view.Seat) continue;
    const hand = handOf(view, seat);
    hands.push(hand);
    hand.forEach((card, i) => {
      if (card.Facing === Face.Down) hints.set(card.ID, hintFor(hand, i));
    });
  }
  if (!isSilver) refineBySlots(hands, hints, remaining - tokened, sv, feasible);
  return hints;
}

/**
 * Placement pass for a suited call. Within a run of untokened face-down cards
 * (hands are sorted), the copies of the called value form one contiguous block.
 * When no feasible value fits below (above) the call in the run's bounds, the
 * block touches the run's start (end), which excludes the other slots. When
 * every remaining copy must sit in a single run, the slots shared by all block
 * placements are "sure".
 */
function refineBySlots(
  hands: ViewCard[][],
  hints: Map<CardID, MatchHint>,
  copies: number,
  sv: number,
  feasible: CardValue[],
): void {
  if (copies < 1) return;
  type Run = { cards: ViewCard[]; lowNone: boolean; highNone: boolean };
  const runs: Run[] = [];
  for (const hand of hands) {
    for (let start = 0; start < hand.length; start++) {
      const isOpen = (c: ViewCard) => c.Facing === Face.Down && !c.TokenDisplay;
      if (!isOpen(hand[start]!) || (start > 0 && isOpen(hand[start - 1]!))) continue;
      let end = start;
      while (end + 1 < hand.length && isOpen(hand[end + 1]!)) end++;
      const { lo } = neighbourBounds(hand, start);
      const { hi } = neighbourBounds(hand, end);
      const fits = (min: number, max: number) =>
        feasible.some(
          (v) => sortValue(v) >= Math.max(lo, min) && sortValue(v) <= Math.min(hi, max),
        );
      runs.push({
        cards: hand.slice(start, end + 1),
        lowNone: !fits(-Infinity, sv - 1),
        highNone: !fits(sv + 1, Infinity),
      });
    }
  }

  const candidates: { run: Run; index: number }[] = [];
  for (const run of runs) {
    const k = run.cards.length;
    const block = Math.min(copies, k);
    run.cards.forEach((card, j) => {
      let allowed = true;
      if (run.lowNone && run.highNone) allowed = k <= copies;
      else if (run.lowNone) allowed = j < block;
      else if (run.highNone) allowed = j >= k - block;
      if (!allowed) hints.set(card.ID, "excluded");
      else if (hints.get(card.ID) !== "excluded") candidates.push({ run, index: j });
    });
  }

  if (candidates.length === copies) {
    for (const { run, index } of candidates) hints.set(run.cards[index]!.ID, "sure");
    return;
  }
  const only = candidates[0]?.run;
  if (!only || candidates.some((c) => c.run !== only) || copies > only.cards.length) return;
  const k = only.cards.length;
  const first = only.lowNone ? 0 : k - copies;
  const last = (only.highNone ? k - copies : 0) + copies;
  for (const { run, index } of candidates) {
    if (index >= first && index < last) hints.set(run.cards[index]!.ID, "sure");
  }
}
