// Seat-layout mapping for the table view.
//
// The viewing player is always rendered at the BOTTOM edge. The three other
// players are placed so that clockwise turn order, starting from the bottom
// seat, visits bottom -> left -> top -> right around the table. This keeps the
// visual rotation consistent with the engine's clockwise seat advancement
// (SeatID is 0..3 in fixed clockwise order; see backend engine card.go).
//
//             [ top seat ]
// [ left seat ]            [ right seat ]
//             [ you (bottom) ]

import type { SeatID } from "./types";

/** The four screen positions around the table, keyed by seat. */
export interface SeatLayout {
  bottom: SeatID;
  left: SeatID;
  top: SeatID;
  right: SeatID;
}

/** True when `seat` is a valid SeatID (an integer in 0..3). */
function isValidSeat(seat: SeatID): boolean {
  return Number.isInteger(seat) && seat >= 0 && seat <= 3;
}

/**
 * seatLayout maps the viewing player's seat to the four screen positions.
 *
 * Pure and I/O-free. The viewer sits at the bottom; the next three seats in
 * clockwise turn order are placed at left, top, and right:
 *
 *   bottom = viewerSeat
 *   left   = (viewerSeat + 1) % 4
 *   top    = (viewerSeat + 2) % 4
 *   right  = (viewerSeat + 3) % 4
 *
 * @throws RangeError if `viewerSeat` is not an integer in 0..3.
 */
export function seatLayout(viewerSeat: SeatID): SeatLayout {
  if (!isValidSeat(viewerSeat)) {
    throw new RangeError(
      `seatLayout: viewerSeat must be an integer in 0..3, got ${viewerSeat}`,
    );
  }
  return {
    bottom: viewerSeat,
    left: (viewerSeat + 1) % 4,
    top: (viewerSeat + 2) % 4,
    right: (viewerSeat + 3) % 4,
  };
}
