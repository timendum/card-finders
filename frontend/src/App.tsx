// the top-level router

import { useCallback, useState } from "react";
import { GameScreen } from "./screens/GameScreen.tsx";
import { JoinScreen } from "./screens/JoinScreen.tsx";
import { clearSession, loadSession, saveSession, type StoredSession } from "./api/session.ts";
import type { JoinGameResponse } from "./types.ts";

export function App() {
  // Lazy initializer so session storage is read exactly once, on mount.
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
    return <GameScreen token={session.token} seat={session.seat} onLeave={onLeave} />;
  }
  return <JoinScreen onJoined={onJoined} />;
}
