// Session_Token lives in the URL hash (#token=...&seat=2), so
// each tab carries its own session. The hash never reaches the server.

import type { SeatID, SessionToken } from "../types.ts";

const TOKEN_PARAM = "token";
const SEAT_PARAM = "seat";

export interface StoredSession {
  token: SessionToken;
  seat: SeatID;
}

function hashParams(): URLSearchParams {
  return new URLSearchParams(window.location.hash.replace(/^#/u, ""));
}

export function loadSession(): StoredSession | null {
  try {
    const params = hashParams();
    const token = params.get(TOKEN_PARAM);
    if (!token) {
      return null;
    }
    const seatRaw = params.get(SEAT_PARAM);
    const seat = seatRaw === null ? 0 : Math.trunc(Number(seatRaw));
    return { token, seat: Number.isNaN(seat) ? 0 : seat };
  } catch {
    return null;
  }
}

export function saveSession(session: StoredSession): void {
  try {
    const params = new URLSearchParams();
    params.set(TOKEN_PARAM, session.token);
    params.set(SEAT_PARAM, String(session.seat));
    // replaceState, not push: reload-resume without piling up history entries.
    // Path only: a token URL never carries a ?join= code.
    window.history.replaceState(null, "", `${window.location.pathname}#${params.toString()}`);
  } catch {
    // URL update unavailable: caller still holds the token in React state, so
    // the game is playable this tab without reload-resume.
  }
}

export function clearSession(): void {
  try {
    // Drop the hash; the URL never carries a query alongside a token.
    window.history.replaceState(null, "", window.location.pathname);
  } catch {
    // ignore
  }
}
