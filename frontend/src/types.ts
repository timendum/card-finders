// Shared wire types for the Card Finders frontend.
//
// Structure (see backend/tygo.yaml):
//   - Everything the engine owns — interfaces (Card, CardValue, Level,
//     SpecialPool, TokenInfo, ViewCard, PlayerView, Action, SelfRevealTarget),
//     the enums (SpecialKind, ActionKind, Phase, Result, Face), the scalar
//     aliases (Rank, SeatID, CardID, SessionToken, Suit, TokenKind) and the
//     TokenKindNaturalIdentity const — is GENERATED from the Go source into
//     ./generated/engine.ts by tygo and re-exported here, so it never drifts
//     from the structs the backend marshals. Do not hand-edit that file.
//   - Suit has no Go constants and TokenKind is a single-member group, so tygo
//     emits them as plain number aliases. This module adds ergonomic wrappers
//     (SuitValue, TokenKindValue) for the named values; their integers are
//     pinned to the Go source by a backend guard test (TestWireEnumValues).
//   - The TRANSPORT envelopes (REST wrappers and WebSocket frames) are
//     hand-written here; they live in the Go `server` package, outside tygo's
//     engine scope.

// Re-export the full generated engine surface.
export * from "./generated/engine";

// -----------------------------------------------------------------------------
// Named values for the plain-number engine scalars (Suit, TokenKind).
//
// tygo emits `Suit` and `TokenKind` as `number` aliases (Suit has no Go
// constants; TokenKind is a single-member const group), so there are no
// generated member names. These const objects give the four suits and the token
// kind ergonomic names without shadowing the generated types. Values are pinned
// to the Go source by the backend guard test TestWireEnumValues.
// -----------------------------------------------------------------------------

import type { Suit, TokenKind } from "./generated/engine";

/** The four suits, numbered 0..3 (engine card.go). Meaningful only for SpecialKind.None cards. */
export const SuitValue = {
  Spades: 0,
  Hearts: 1,
  Diamonds: 2,
  Clubs: 3,
} as const satisfies Record<string, Suit>;

/** Token display kinds (engine token.go). NaturalIdentity (1) is the only kind today. */
export const TokenKindValue = {
  NaturalIdentity: 1,
} as const satisfies Record<string, TokenKind>;

// -----------------------------------------------------------------------------
// REST wrapper types — camelCase keys (json tags on the Go side, rest.go)
// -----------------------------------------------------------------------------

import type { Level } from "./generated/engine";

/** GET /api/levels -> 200 */
export interface LevelsResponse {
  levels: Level[];
}

/** POST /api/game request body. */
export interface CreateGameRequest {
  levelNumber: number;
}

/** POST /api/game -> 201 */
export interface CreateGameResponse {
  joinCode: string;
}

/** POST /api/game/join request body. */
export interface JoinGameRequest {
  joinCode: string;
}

/** POST /api/game/join -> 200 */
export interface JoinGameResponse {
  sessionToken: string;
  seat: number;
}

/** The common error envelope returned by the REST handlers (errors.go). */
export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

// -----------------------------------------------------------------------------
// WebSocket envelopes — camelCase keys, discriminated on `type` (wsmsg.go)
// -----------------------------------------------------------------------------

import type { CardID, PlayerView } from "./generated/engine";

// --- Server -> client (outbound) ---

/** `state` frame: the receiving seat's full PlayerView snapshot. */
export interface StateMessage {
  type: "state";
  view: PlayerView;
}

/** `error` frame: a stable machine code and a human-readable message. */
export interface ErrorMessage {
  type: "error";
  code: string;
  message: string;
}

/** Any message the server may push to the client. */
export type ServerMessage = StateMessage | ErrorMessage;

// --- Client -> server (inbound) ---

/** Opening-phase reveal of one card (Phase.Opening). */
export interface OpeningRevealMessage {
  type: "openingReveal";
  cardId: CardID;
}

/** A guess against a single target card. */
export interface GuessMessage {
  type: "guess";
  sourceCardId: CardID;
  targetCardId: CardID;
}

/** A safe call against two target cards. */
export interface SafeCallMessage {
  type: "safeCall";
  sourceCardId: CardID;
  targetCardId: CardID;
  targetCardId2: CardID;
}

/** The targeted player''s choice of which card to reveal after a double hit. */
export interface SafeChooseRevealMessage {
  type: "safeChooseReveal";
  chosenTargetCardId: CardID;
}

/** A suited rank (2..13) or one of the two specials. */
export type SelfRevealValue = number | "silver" | "gold";

/** A self-reveal naming a suited rank or a special kind. */
export interface SelfRevealMessage {
  type: "selfReveal";
  value: SelfRevealValue;
}

/** Any message the client may send to the server. */
export type ClientMessage =
  | OpeningRevealMessage
  | GuessMessage
  | SafeCallMessage
  | SafeChooseRevealMessage
  | SelfRevealMessage;
