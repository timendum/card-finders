// JoinScreen — pre-game lobby with Create and Join panels. Both end by handing
// the issued session up to the parent via onJoined.

import { useCallback, useEffect, useState } from "react";
import { ApiError, createGame, fetchLevels, joinGame } from "./api.ts";
import type { JoinGameResponse, Level } from "./types.ts";

export interface JoinScreenProps {
  onJoined: (session: JoinGameResponse) => void;
}

type Mode = "create" | "join";

function messageOf(err: unknown): string {
  if (err instanceof ApiError) {
    return err.message;
  }
  return "Something went wrong. Please try again.";
}

export function JoinScreen({ onJoined }: JoinScreenProps) {
  const [mode, setMode] = useState<Mode>("create");

  return (
    <main>
      <h1>Card Finders</h1>
      <nav aria-label="Lobby mode">
        <button type="button" aria-pressed={mode === "create"} onClick={() => setMode("create")}>
          Create game
        </button>
        <button type="button" aria-pressed={mode === "join"} onClick={() => setMode("join")}>
          Join game
        </button>
      </nav>
      {mode === "create" ? <CreatePanel onJoined={onJoined} /> : <JoinPanel onJoined={onJoined} />}
    </main>
  );
}

// CreatePanel picks a level and creates the game. Creating does not seat the
// host, so it finishes by joining the host into the freshly created game.
function CreatePanel({ onJoined }: JoinScreenProps) {
  const [levels, setLevels] = useState<Level[] | null>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const [joinCode, setJoinCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchLevels()
      .then((loaded) => {
        if (cancelled) return;
        setLevels(loaded);
        setSelected(loaded.length > 0 ? loaded[0].Number : null);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(messageOf(err));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const onCreate = useCallback(async () => {
    if (selected === null || busy) return;
    setBusy(true);
    setError(null);
    try {
      const code = await createGame(selected);
      setJoinCode(code);
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBusy(false);
    }
  }, [selected, busy]);

  const onJoinCreated = useCallback(async () => {
    if (!joinCode || busy) return;
    setBusy(true);
    setError(null);
    try {
      const session = await joinGame(joinCode);
      onJoined(session);
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBusy(false);
    }
  }, [joinCode, busy, onJoined]);

  if (joinCode) {
    return (
      <section aria-label="Game created">
        <h2>Game created</h2>
        <p>
          Share this join code: <strong>{joinCode}</strong>
        </p>
        {error && <p role="alert">{error}</p>}
        <button type="button" onClick={onJoinCreated} disabled={busy}>
          {busy ? "Joining…" : "Join this game"}
        </button>
      </section>
    );
  }

  if (error && !levels) {
    return (
      <section aria-label="Create game">
        <p role="alert">{error}</p>
      </section>
    );
  }

  if (!levels) {
    return (
      <section aria-label="Create game">
        <p>Loading levels…</p>
      </section>
    );
  }

  return (
    <section aria-label="Create game">
      <h2>Pick a level</h2>
      <ul>
        {levels.map((level) => (
          <li key={level.Number}>
            <label>
              <input
                type="radio"
                name="level"
                value={level.Number}
                checked={selected === level.Number}
                onChange={() => setSelected(level.Number)}
              />
              {level.Name} — silver {level.SilverCount}, gold {level.GoldCount}, lives{" "}
              {level.StartLives}
            </label>
          </li>
        ))}
      </ul>
      {error && <p role="alert">{error}</p>}
      <button type="button" onClick={onCreate} disabled={busy || selected === null}>
        {busy ? "Creating…" : "Create game"}
      </button>
    </section>
  );
}

// JoinPanel seats the player from a join code.
function JoinPanel({ onJoined }: JoinScreenProps) {
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const onSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (busy) return;
      setBusy(true);
      setError(null);
      try {
        const session = await joinGame(code.trim());
        onJoined(session);
      } catch (err) {
        setError(messageOf(err));
      } finally {
        setBusy(false);
      }
    },
    [code, busy, onJoined],
  );

  return (
    <section aria-label="Join game">
      <h2>Join a game</h2>
      <form onSubmit={onSubmit}>
        <label>
          Join code
          <input
            type="text"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            autoComplete="off"
          />
        </label>
        {error && <p role="alert">{error}</p>}
        <button type="submit" disabled={busy || code.trim() === ""}>
          {busy ? "Joining…" : "Join game"}
        </button>
      </form>
    </section>
  );
}
