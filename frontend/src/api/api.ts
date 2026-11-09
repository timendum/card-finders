// Endpoints are reached relative to the page origin so they work both behind
// the Vite dev proxy and when served by the Go binary.

import type {
  ApiErrorResponse,
  CreateGameResponse,
  JoinGameResponse,
  Level,
  LevelsResponse,
} from "../types.ts";

// `code` is the backend's machine code, or a synthetic code ("network_error",
// "bad_response") when no well-formed envelope was received. `message` is always
// safe to show the user.
export class ApiError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

// Falls back to a generic code when the body is not the { error: { code,
// message } } envelope (e.g. an unexpected 5xx).
async function toApiError(res: Response): Promise<ApiError> {
  try {
    const body = (await res.json()) as Partial<ApiErrorResponse>;
    if (body.error && typeof body.error.code === "string") {
      return new ApiError(body.error.code, body.error.message || body.error.code);
    }
  } catch {
    // body was not JSON
  }
  return new ApiError("bad_response", `Request failed with status ${res.status}`);
}

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

function jsonPost(body: unknown): RequestInit {
  return {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export async function fetchLevels(): Promise<Level[]> {
  const res = await request("/api/levels", { method: "GET" });
  const body = (await res.json()) as LevelsResponse;
  return body.levels;
}

export async function createGame(levelNumber: number): Promise<string> {
  const res = await request("/api/game", jsonPost({ levelNumber }));
  const body = (await res.json()) as CreateGameResponse;
  return body.joinCode;
}

export async function joinGame(joinCode: string): Promise<JoinGameResponse> {
  const res = await request("/api/game/join", jsonPost({ joinCode }));
  return (await res.json()) as JoinGameResponse;
}
