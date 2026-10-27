// REST client helpers for the Card Finders join flow. The three endpoints —
// GET /api/levels, POST /api/game, and
// POST /api/game/join — are reached relative to the page origin so they work
// both behind the Vite dev proxy and when served by the Go binary. Every
// failure is normalized into an ApiError carrying the backend's stable machine
// code (errors.go: level_required, level_invalid, game_exists, game_full,
// code_invalid, code_required, level_config_invalid, bad_request) so the UI can
// present a human-readable message without parsing HTTP status codes itself.

import type {
  ApiErrorResponse,
  CreateGameResponse,
  JoinGameResponse,
  Level,
  LevelsResponse,
} from "./types.ts";

/**
 * ApiError is thrown by every helper below when a request fails. `code` is the
 * backend's stable machine code when the server returned the JSON error
 * envelope, or a synthetic code ("network_error", "bad_response") when the
 * request never reached a well-formed envelope. `message` is always safe to
 * show the user.
 */
export class ApiError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

/**
 * Parse a non-OK Response into an ApiError. The REST handlers always answer a
 * failure with the { error: { code, message } } envelope; if the body is not
 * that shape (an unexpected 5xx, say) we fall back to a generic code so the
 * caller still gets a usable ApiError.
 */
async function toApiError(res: Response): Promise<ApiError> {
  try {
    const body = (await res.json()) as Partial<ApiErrorResponse>;
    if (body.error && typeof body.error.code === "string") {
      return new ApiError(body.error.code, body.error.message || body.error.code);
    }
  } catch {
    // body was not JSON; fall through to the generic error below
  }
  return new ApiError("bad_response", `Request failed with status ${res.status}`);
}

/** Issue a request, normalizing transport failures into an ApiError. */
async function request(input: string, init?: RequestInit): Promise<Response> {
  let res: Response;
  try {
    res = await fetch(input, init);
  } catch {
    throw new ApiError("network_error", "Could not reach the server. Check your connection.");
  }
  if (!res.ok) {
    throw await toApiError(res);
  }
  return res;
}

/** JSON body + headers for a POST request. */
function jsonPost(body: unknown): RequestInit {
  return {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

/** GET /api/levels — the level list the host picks from. */
export async function fetchLevels(): Promise<Level[]> {
  const res = await request("/api/levels", { method: "GET" });
  const body = (await res.json()) as LevelsResponse;
  return body.levels;
}

/**
 * POST /api/game — create the single game at the chosen level, returning the
 * join code other players use to take a seat.
 */
export async function createGame(levelNumber: number): Promise<string> {
  const res = await request("/api/game", jsonPost({ levelNumber }));
  const body = (await res.json()) as CreateGameResponse;
  return body.joinCode;
}

/**
 * POST /api/game/join — take a seat in the game identified by the join code,
 * returning the new Session_Token and the assigned seat.
 */
export async function joinGame(joinCode: string): Promise<JoinGameResponse> {
  const res = await request("/api/game/join", jsonPost({ joinCode }));
  return (await res.json()) as JoinGameResponse;
}
