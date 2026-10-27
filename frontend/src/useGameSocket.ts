// useGameSocket: the React hook that owns a single WebSocket to the Go backend.
// It opens `/ws?token=...` when a token is present, keeps the latest PlayerView
// from each `state` frame, surfaces the latest `error` frame, reconnects with
// capped exponential backoff + jitter on unexpected close (re-using the same
// token so the server restores the seat and re-pushes the current view), and
// exposes a `sendAction` to serialize ClientMessage values to the server.
//
// Backend contract (backend/internal/server/ws.go): the client connects to
// `/ws?token=<sessionToken>`. On connect the server immediately pushes a full
// `state` frame; after every successful action it broadcasts a fresh `state`
// frame; an invalid action yields an `error` frame with state unchanged. A
// missing token is HTTP 400 and an invalid token HTTP 401 — the upgrade fails,
// which the browser surfaces as a close, so those are handled on the close path.

import { useCallback, useEffect, useRef, useState } from "react";
import type {
  ClientMessage,
  ErrorMessage,
  PlayerView,
  ServerMessage,
} from "./types.ts";

/** Connection status exposed to the UI. */
export type SocketStatus = "idle" | "connecting" | "open" | "closed";

/** The latest `error` frame, narrowed to its payload (no `type` field). */
export interface SocketError {
  code: string;
  message: string;
}

/** The hook's return shape. */
export interface GameSocket {
  /** Latest PlayerView from the most recent `state` frame, or null until one arrives. */
  view: PlayerView | null;
  /** Latest `error` frame payload, or null when none has arrived since the last clear. */
  error: SocketError | null;
  /** Current connection status for the UI. */
  status: SocketStatus;
  /**
   * Serialize and send a ClientMessage over the socket. When the socket is not
   * OPEN the call is a no-op and returns false (the caller can gate on `status`
   * === "open"); on a successful send it returns true. Actions are intentionally
   * not queued: the server pushes a full snapshot on (re)connect, so a UI built
   * on the latest `view` can simply re-issue once reconnected.
   */
  sendAction: (msg: ClientMessage) => boolean;
  /** Clear the latest error (e.g. after the UI has shown it). */
  clearError: () => void;
}

// Backoff schedule: exponential from BASE, doubling per attempt, capped at MAX,
// with up to JITTER added so reconnecting clients don't align into a thundering
// herd. Values are milliseconds.
const BACKOFF_BASE_MS = 500;
const BACKOFF_MAX_MS = 15_000;
const BACKOFF_JITTER_MS = 250;

/** Compute the delay for a given 0-based reconnect attempt. */
function backoffDelay(attempt: number): number {
  const exp = Math.min(BACKOFF_BASE_MS * 2 ** attempt, BACKOFF_MAX_MS);
  return exp + Math.floor(Math.random() * BACKOFF_JITTER_MS);
}

/**
 * Build the absolute ws:// or wss:// URL for the given token from the current
 * page origin, so it works both behind the Vite dev proxy and when served by
 * the Go binary. Uses wss when the page is https.
 */
function buildSocketUrl(token: string): string {
  const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${scheme}//${window.location.host}/ws?token=${encodeURIComponent(token)}`;
}

/**
 * useGameSocket manages one live WebSocket keyed on `token`. Pass null (e.g.
 * before the player has a Session_Token) to stay disconnected; passing a token
 * opens the socket, and changing or clearing the token tears the old one down
 * before opening the new one.
 *
 * React-18 StrictMode double-invokes effects in dev. The effect cleanup here
 * fully closes the socket and cancels any pending reconnect timer, and a
 * generation counter makes the just-torn-down socket's late events no-ops, so a
 * double-invoke settles on exactly one live connection.
 */
export function useGameSocket(token: string | null): GameSocket {
  const [view, setView] = useState<PlayerView | null>(null);
  const [error, setError] = useState<SocketError | null>(null);
  const [status, setStatus] = useState<SocketStatus>("idle");

  // The current socket, so sendAction can reach it without re-rendering.
  const socketRef = useRef<WebSocket | null>(null);
  // Pending reconnect timer id, so cleanup can cancel it.
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Consecutive reconnect attempts; reset to 0 on a successful open.
  const attemptRef = useRef(0);
  // Generation counter: bumped on every teardown so a stale socket's queued
  // onclose/onmessage handlers can detect they belong to a dead connection and
  // bail, which is what keeps StrictMode's double-invoke from reconnecting a
  // socket the cleanup already closed.
  const genRef = useRef(0);

  const clearError = useCallback(() => setError(null), []);

  useEffect(() => {
    // No token means stay disconnected and reset the UI state.
    if (!token) {
      setStatus("idle");
      setView(null);
      return;
    }

    // Each effect run owns a generation. If cleanup runs (unmount, token
    // change, StrictMode re-invoke) it bumps genRef; any handler that captured
    // an older generation then knows it is stale and does nothing.
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
        attemptRef.current = 0; // reset backoff after a clean connect
        setStatus("open");
      };

      ws.onmessage = (ev) => {
        if (isStale()) return;
        let msg: ServerMessage;
        try {
          msg = JSON.parse(ev.data as string) as ServerMessage;
        } catch {
          // A frame we can't parse is ignored; the next valid snapshot still
          // renders. The server only ever sends well-formed JSON envelopes.
          return;
        }
        switch (msg.type) {
          case "state":
            // Latest full snapshot wins.
            setView(msg.view);
            break;
          case "error": {
            // Surface the latest error payload without the discriminator. State is unchanged on the server, so `view` is left as is.
            const { code, message } = msg as ErrorMessage;
            setError({ code, message });
            break;
          }
          default:
            // Unknown frame type: ignore forward-compatibly.
            break;
        }
      };

      ws.onclose = () => {
        if (isStale()) return;
        socketRef.current = null;
        setStatus("closed");
        // Unexpected close (dropped connection, or a failed upgrade from a
        // missing/invalid token surfacing as a close): reconnect with backoff,
        // re-using the same token so the server restores the seat and re-pushes
        // the current view.
        scheduleReconnect();
      };

      ws.onerror = () => {
        // Errors are followed by a close event; closing here guarantees the
        // onclose path (and thus the reconnect) runs even if the browser would
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

    // Cleanup on unmount or when `token` changes. Bumping the generation makes
    // any in-flight handler stale; then we cancel a pending reconnect and close
    // the live socket after detaching its handlers so the close doesn't trigger
    // another reconnect.
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
  }, [token]);

  const sendAction = useCallback((msg: ClientMessage): boolean => {
    const ws = socketRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      // Not connected: no-op. The server re-pushes a full snapshot on
      // reconnect, so the UI can re-issue the action once `status` is "open".
      return false;
    }
    ws.send(JSON.stringify(msg));
    return true;
  }, []);

  return { view, error, status, sendAction, clearError };
}
