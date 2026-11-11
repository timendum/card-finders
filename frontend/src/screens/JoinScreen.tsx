// JoinScreen — pre-game lobby with Create and Join panels. Both end by handing
// the issued session up to the parent via onJoined.

import { useCallback, useState } from "react";
import { ApiError, FORBIDDEN_CODE, createGame, fetchLevels, joinGame } from "../api/api.ts";
import type { JoinGameResponse, Level } from "../types.ts";

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

// A shared link (/?join=CODE) opens straight into the join panel.
const JOIN_PARAM = "join";
const linkedCode = new URLSearchParams(window.location.search).get(JOIN_PARAM) ?? "";

export function JoinScreen({ onJoined }: JoinScreenProps) {
  const [mode, setMode] = useState<Mode | null>(linkedCode ? "join" : null);

  // Back is a full page reload so all in-memory state (including the auth
  // password) is dropped.
  const onBack = () => window.location.reload();

  return (
    <main>
      <h1>Card Finders</h1>
      {mode === null ? (
        <nav aria-label="Lobby mode">
          <button type="button" onClick={() => setMode("create")}>
            Create game
          </button>
          <button type="button" onClick={() => setMode("join")}>
            Join game
          </button>
        </nav>
      ) : (
        <>
          <button type="button" onClick={onBack}>
            Back
          </button>
          {mode === "create" ? (
            <CreatePanel onJoined={onJoined} />
          ) : (
            <JoinPanel onJoined={onJoined} />
          )}
        </>
      )}
    </main>
  );
}

function formatLevelPart(kind: string, count: number, extra: number): string {
  return extra > 0 ? `${kind} ${count} + ${extra} decoy` : `${kind} ${count}`;
}

/** Describes a level's specials, e.g. "silver 2 + 1 decoy, gold 1". */
function formatLevelSpecials(
  level: Level,
): string {
  return `${formatLevelPart("silver", level.SilverCount, level.ExtraSilver)}, ${formatLevelPart("gold", level.GoldCount, level.ExtraGold)}`;
}


// CreatePanel picks a level and creates the game. Creating does not seat the
// host, so it finishes by joining the host into the freshly created game.
function CreatePanel({ onJoined }: JoinScreenProps) {
  const [password, setPassword] = useState("");
  const [levels, setLevels] = useState<Level[] | null>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const [joinCode, setJoinCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Fetching the levels doubles as the credential check: a 403 keeps the user
  // on the auth form, success moves on to level selection.
  const onAuth = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (busy || password === "") return;
      setBusy(true);
      setError(null);
      try {
        const loaded = await fetchLevels(password);
        setLevels(loaded);
        setSelected(loaded.length > 0 ? loaded[0].Number : null);
      } catch (err) {
        if (err instanceof ApiError && err.code === FORBIDDEN_CODE) {
          setError("Wrong password. Please try again.");
        } else {
          setError(messageOf(err));
        }
      } finally {
        setBusy(false);
      }
    },
    [busy, password],
  );

  const onCreate = useCallback(async () => {
    if (selected === null || busy) return;
    setBusy(true);
    setError(null);
    try {
      const code = await createGame(selected, password);
      setJoinCode(code);
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBusy(false);
    }
  }, [selected, busy, password]);

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
    const shareUrl = `${window.location.origin}/?${JOIN_PARAM}=${encodeURIComponent(joinCode)}`;
    return (
      <section aria-label="Game created">
        <h2>Game created</h2>
        <p>
          Share this join code: <strong>{joinCode}</strong>
        </p>
        <p>
          Share this URL: <a href={shareUrl}>{shareUrl}</a>
        </p>
        {error && <p role="alert">{error}</p>}
        <button type="button" onClick={onJoinCreated} disabled={busy}>
          {busy ? "Joining…" : "Join this game"}
        </button>
      </section>
    );
  }

  if (!levels) {
    return (
      <section aria-label="Authentication">
        <h2>Authentication</h2>
        <form onSubmit={onAuth}>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="off"
            />
          </label>
          {error && <p role="alert">{error}</p>}
          <button type="submit" disabled={busy || password === ""}>
            {busy ? "Checking…" : "Continue"}
          </button>
        </form>
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
              {level.Name} — {formatLevelSpecials(level)}, lives{" "}
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
  const [code, setCode] = useState(linkedCode);
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
