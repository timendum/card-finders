// The viewer sits at the bottom; the next three seats in clockwise turn order
// go to left, top, then right.

import type { PlayerView, SeatID } from "../types";

export interface SeatLayout {
  bottom: SeatID;
  left: SeatID;
  top: SeatID;
  right: SeatID;
}

function isValidSeat(seat: SeatID): boolean {
  return Number.isInteger(seat) && seat >= 0 && seat <= 3;
}

/** Position name of `seat` relative to the viewer, matching the table layout. */
export function seatPositionName(viewer: SeatID, seat: SeatID): "you" | "left" | "top" | "right" {
  return (["you", "left", "top", "right"] as const)[(seat - viewer + 4) % 4] ?? "you";
}

/** Display name of `seat` for the viewer: "You" for their own seat, else the player's name. */
export function seatName(view: Pick<PlayerView, "Seat" | "Players">, seat: SeatID): string {
  if (seat === view.Seat) return "You";
  const name = view.Players?.[seat]?.Name;
  if (name) return name;
  const pos = seatPositionName(view.Seat, seat);
  return pos.charAt(0).toUpperCase() + pos.slice(1);
}

export function seatLayout(viewerSeat: SeatID): SeatLayout {
  if (!isValidSeat(viewerSeat)) {
    throw new RangeError(`seatLayout: viewerSeat must be an integer in 0..3, got ${viewerSeat}`);
  }
  return {
    bottom: viewerSeat,
    left: (viewerSeat + 1) % 4,
    top: (viewerSeat + 2) % 4,
    right: (viewerSeat + 3) % 4,
  };
}
