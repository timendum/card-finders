// Persistence of the Session_Token (and the seat it was issued for) in
// localStorage. `App` reads this on mount to decide whether to show the
// JoinScreen or the GameScreen, and the join flow writes it on a successful
// create+join or join so a page reload resumes the same seat. All access is
// wrapped so a browser with storage disabled degrades to an in-memory-only
// session rather than throwing.

import type { SeatID, SessionToken } from "./types.ts";

const TOKEN_KEY = "cardfinders.sessionToken";
const SEAT_KEY = "cardfinders.seat";

/** A stored session: the token bound to a seat and that seat's id. */
export interface StoredSession {
  token: SessionToken;
  seat: SeatID;
}

/** Read the persisted session, or null when none is stored or storage is unavailable. */
export function loadSession(): StoredSession | null {
  try {
    const token = window.localStorage.getItem(TOKEN_KEY);
    const seatRaw = window.localStorage.getItem(SEAT_KEY);
    if (!token) {
      return null;
    }
    const seat = seatRaw === null ? 0 : Number.parseInt(seatRaw, 10);
    return { token, seat: Number.isNaN(seat) ? 0 : seat };
  } catch {
    return null;
  }
}

/** Persist the session so a reload resumes the same seat. */
export function saveSession(session: StoredSession): void {
  try {
    window.localStorage.setItem(TOKEN_KEY, session.token);
    window.localStorage.setItem(SEAT_KEY, String(session.seat));
  } catch {
    // Storage unavailable (private mode, quota). The caller still holds the
    // token in React state for this tab, so the game is playable without
    // reload-resume.
  }
}

/** Remove the persisted session (e.g. on an explicit leave). */
export function clearSession(): void {
  try {
    window.localStorage.removeItem(TOKEN_KEY);
    window.localStorage.removeItem(SEAT_KEY);
  } catch {
    // ignore
  }
}
