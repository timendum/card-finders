// Frontend wire types: re-exports the generated engine surface and adds the
// hand-written transport envelopes (REST wrappers, WS frames) that live in the
// Go `server` package, outside tygo's engine scope.

export * from "./generated/engine";

// Named values for Suit and TokenKind, which tygo emits as plain number aliases.
// Integers pinned to the Go source by backend guard test TestWireEnumValues.

import type { Suit, TokenKind } from "./generated/engine";

/** The four suits. Meaningful only for SpecialKind.None cards. */
export const SuitValue = {
  Spades: 0,
  Hearts: 1,
  Diamonds: 2,
  Clubs: 3,
} as const satisfies Record<string, Suit>;

/** Token display kinds. NaturalIdentity is the only kind today. */
export const TokenKindValue = {
  NaturalIdentity: 1,
} as const satisfies Record<string, TokenKind>;

// REST envelopes — camelCase keys.

import type { Level } from "./generated/engine";

/** GET /api/levels */
export interface LevelsResponse {
  levels: Level[];
}

/** POST /api/game request body. */
export interface CreateGameRequest {
  levelNumber: number;
}

/** POST /api/game response. */
export interface CreateGameResponse {
  joinCode: string;
}

/** POST /api/game/join request body. */
export interface JoinGameRequest {
  joinCode: string;
}

/** POST /api/game/join response. */
export interface JoinGameResponse {
  sessionToken: string;
  seat: number;
}

/** Common error envelope for all REST handlers. */
export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

// WebSocket envelopes — camelCase keys, discriminated on `type`.

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

/** Opening-phase reveal of one card. */
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

/** The targeted player's choice of which card to reveal after a double hit. */
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
