// App — the top-level router. It holds the active session (Session_Token +
// seat) in state, seeded from localStorage on first render so a reload resumes
// the same seat.
// With no session it shows the JoinScreen; once a seat is taken it persists the
// session and swaps in the GameScreen. Leaving clears the stored session and
// returns to the lobby.

import { useCallback, useState } from "react";
import { GameScreen } from "./GameScreen.tsx";
import { JoinScreen } from "./JoinScreen.tsx";
import { clearSession, loadSession, saveSession, type StoredSession } from "./session.ts";
import type { JoinGameResponse } from "./types.ts";

export function App() {
  // Lazy initializer so localStorage is read exactly once, on mount.
  const [session, setSession] = useState<StoredSession | null>(() => loadSession());

  const onJoined = useCallback((joined: JoinGameResponse) => {
    const next: StoredSession = { token: joined.sessionToken, seat: joined.seat };
    saveSession(next);
    setSession(next);
  }, []);

  const onLeave = useCallback(() => {
    clearSession();
    setSession(null);
  }, []);

  if (session) {
    return (
      <GameScreen token={session.token} seat={session.seat} onLeave={onLeave} />
    );
  }
  return <JoinScreen onJoined={onJoined} />;
}
