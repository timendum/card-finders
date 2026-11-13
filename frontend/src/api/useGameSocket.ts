// React hook owning a single WebSocket to the backend, keyed on `token`. A
// failed upgrade (missing/invalid token) surfaces as a close, so it reconnects
// on the close path like any other drop.

import { useCallback, useEffect, useRef, useState } from "react";
import type { ClientMessage, ErrorMessage, PlayerView, ServerMessage } from "../types.ts";

// "rejected": the server refused the token (unknown or taken over); no retry.
export type SocketStatus = "idle" | "connecting" | "open" | "closed" | "rejected";

const TOKEN_INVALID_REASON = "token_invalid";

/** The latest `error` frame, narrowed to its payload (no `type` field). */
export interface SocketError {
  code: string;
  message: string;
}

export interface GameSocket {
  view: PlayerView | null;
  error: SocketError | null;
  status: SocketStatus;
  /** Sends a ClientMessage; no-op returning false when the socket is not OPEN. */
  sendAction: (msg: ClientMessage) => boolean;
}

// Exponential backoff with jitter so reconnecting clients don't align into a
// thundering herd. Milliseconds.
const BACKOFF_BASE_MS = 500;
const BACKOFF_MAX_MS = 15_000;
const BACKOFF_JITTER_MS = 250;

function backoffDelay(attempt: number): number {
  const exp = Math.min(BACKOFF_BASE_MS * 2 ** attempt, BACKOFF_MAX_MS);
  return exp + Math.floor(Math.random() * BACKOFF_JITTER_MS);
}

// Builds the ws/wss URL from the page origin so it works both behind the Vite
// dev proxy and when served by the Go binary.
function buildSocketUrl(token: string): string {
  const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${scheme}//${window.location.host}/ws?token=${encodeURIComponent(token)}`;
}

// Pass null for `token` to stay disconnected; changing it tears the old socket
// down before opening the new one. A generation counter neutralizes a
// torn-down socket's late events, so StrictMode's double-invoke settles on one
// live connection.
export function useGameSocket(token: string | null): GameSocket {
  const [view, setView] = useState<PlayerView | null>(null);
  const [error, setError] = useState<SocketError | null>(null);
  const [status, setStatus] = useState<SocketStatus>("idle");

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Consecutive reconnect attempts; reset to 0 on a successful open.
  const attemptRef = useRef(0);
  // Bumped on every teardown; handlers that captured an older value bail.
  const genRef = useRef(0);

  // Reset UI state during render (React's reset-on-prop-change pattern) rather
  // than in the effect, avoiding an extra commit.
  const [prevToken, setPrevToken] = useState(token);
  if (prevToken !== token) {
    setPrevToken(token);
    if (!token) {
      setStatus("idle");
      setView(null);
    }
  }

  useEffect(() => {
    if (!token) {
      return;
    }

    const myGen = ++genRef.current;
    let cancelled = false;

    const isStale = () => cancelled || genRef.current !== myGen;

    const scheduleReconnect = () => {
      if (isStale()) return;
      const delay = backoffDelay(attemptRef.current);
      attemptRef.current += 1;
      reconnectRef.current = setTimeout(() => {
        reconnectRef.current = null;
        connect();
      }, delay);
    };

    const connect = () => {
      if (isStale()) return;
      setStatus("connecting");

      const ws = new WebSocket(buildSocketUrl(token));
      socketRef.current = ws;

      ws.onopen = () => {
        if (isStale()) return;
        attemptRef.current = 0;
        setStatus("open");
      };

      ws.onmessage = (ev) => {
        if (isStale()) return;
        let msg: ServerMessage;
        try {
          msg = JSON.parse(ev.data as string) as ServerMessage;
        } catch {
          return; // ignore unparseable frames; the next snapshot still renders
        }
        switch (msg.type) {
          case "state":
            setView(msg.view);
            setError(null); // a new snapshot makes any earlier error stale
            break;
          case "error": {
            const { code, message } = msg as ErrorMessage;
            setError({ code, message });
            break;
          }
          default:
            break; // ignore unknown frame types forward-compatibly
        }
      };

      ws.onclose = (ev) => {
        if (isStale()) return;
        socketRef.current = null;
        if (ev.reason === TOKEN_INVALID_REASON) {
          setStatus("rejected");
          return;
        }
        setStatus("closed");
        scheduleReconnect();
      };

      ws.onerror = () => {
        // Force the close path (and its reconnect) in case the browser would
        // otherwise leave the socket half-open.
        if (isStale()) return;
        try {
          ws.close();
        } catch {
          // already closing/closed
        }
      };
    };

    connect();

    // Detach handlers before closing so the close doesn't trigger a reconnect.
    // The cleanup reads socketRef/reconnectRef at teardown time, not values
    // captured at effect start, because connect() and the backoff reconnect
    // reassign the live socket; capturing at start (what exhaustive-deps wants)
    // would close the wrong socket and leak the live one.
    /* eslint-disable react-hooks/exhaustive-deps */
    return () => {
      cancelled = true;
      genRef.current++;
      if (reconnectRef.current !== null) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
      attemptRef.current = 0;
      const ws = socketRef.current;
      socketRef.current = null;
      if (ws) {
        ws.onopen = null;
        ws.onmessage = null;
        ws.onclose = null;
        ws.onerror = null;
        try {
          ws.close();
        } catch {
          // already closing/closed
        }
      }
    };
    /* eslint-enable react-hooks/exhaustive-deps */
  }, [token]);

  const sendAction = useCallback((msg: ClientMessage): boolean => {
    const ws = socketRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return false;
    }
    ws.send(JSON.stringify(msg));
    return true;
  }, []);

  return { view, error, status, sendAction };
}
