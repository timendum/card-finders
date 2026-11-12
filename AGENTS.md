# AGENTS.md — Card Finders

Technical orientation for agents. For game rules, terms, phases, and action
semantics, read **[GAME.md](GAME.md)** — this file does not restate business
logic. Human-facing build/run detail lives in [README.md](README.md).

## Shape

Single Go deployable + a React/TS client. Two cleanly separated concerns:

- **Pure engine** (`backend/internal/engine`) — deterministic state machine,
  all game state in memory, seeded RNG. Zero knowledge of HTTP/WS/JSON/config.
  All rules and the per-player view projection live here. Fully unit/property
  testable in isolation.
- **Transport** (`backend/internal/server`) — one mutex-guarded engine
  instance behind REST + WebSocket; maps connections to seats, projects views,
  serves the built frontend from disk.

Dependency arrow points one way: transport → engine. The engine never imports
transport.

## Layout

```
backend/            Go module `cardfinders` (go 1.25)
  cmd/server/       entrypoint: resolve config path, load levels, wire + listen
  internal/engine/  pure engine (types, rules, view projection)
  internal/config/  TOML level-config loader (BurntSushi/toml)
  internal/server/  HTTP mux, REST, WS hub, seat/session mgmt, JSON DTOs
  gen.go, tygo.yaml go:generate directive + tygo config
frontend/           React + TS + Vite, built with Bun
  src/generated/    engine.ts — GENERATED from Go, do not hand-edit
  dist/             build output served by Go from disk
levels.toml         sample level config (repo root)
```

Key deps: `github.com/coder/websocket`, `github.com/BurntSushi/toml`,
`pgregory.net/rapid` (property tests).

## Concurrency

Exactly one game, one engine, one `sync.Mutex`. Every create/join/action takes
the lock; the engine body is never concurrency-aware. View projection runs
under the lock for a consistent snapshot. "Apply action" is one critical
section: validate → mutate → compute views to broadcast.

## Engine API

`internal/engine` exposes a small imperative API, all called by transport under
the lock:

- `NewEngine(levels []Level, rng *rand.Rand) *Engine` — seeded for deterministic
  dealing/pool selection; holds the level slice it is handed.
- `CreateGame(levelNumber int) (joinCode string, err error)`
- `Join(joinCode string) (seat SeatID, token SessionToken, err error)` — the 4th
  join internally runs special-pool + deal + opening setup. Joining after the
  start takes over an away seat and rotates its token (rules: GAME.md);
  `ErrGameFull` only when all four seats are connected.
- `Connect(token SessionToken) (SeatID, error)` / `Disconnect(seat SeatID)` —
  transport marks a seat's `Player.Connected` as sockets open and close
  (`ErrSeatOccupied` if already connected). The flags reach clients as
  `PlayerView.Connected`.
- `ResolveSeat(token SessionToken) (SeatID, error)` — reconnection.
- `Apply(seat SeatID, action Action) (Result, error)` — single entry point for
  all gameplay actions (`Action` is a tagged union).
- `View(seat SeatID) PlayerView` — per-player projection (hides others' hidden
  values; see GAME.md visibility rules). `ViewSpecialPool` lists real and
  decoy entries merged and sorted, plus `SilverInPlay`/`GoldInPlay` (real
  counts); decoys live only in the internal `SpecialPool.DecoySilvers/Golds`.
- `Levels() []Level` — defensive copy of the configured levels (served by
  `GET /api/levels`).

Phases are the first gate in every action handler. See GAME.md for phase and
action meaning.

## Transport surfaces

Three independent paths on one `http.ServeMux`:

- `/api/...` — REST (JSON). Errors use `{ "error": { "code", "message" } }`.
  - `GET  /api/levels` → `{ levels: [Level...] }`. Each `Level` carries
    `Number`, `Name`, `SilverCount`, `GoldCount`, `ExtraSilver`, `ExtraGold`,
    `StartLives`, `Jokers` (extras = decoy specials, see GAME.md).
  - `POST /api/game` `{ levelNumber }` → `201 { joinCode }`. Errors:
    `level_required` (400), `level_invalid` (400), `level_config_invalid` (400),
    `game_exists` (409).
  - `GET /api/levels` and `POST /api/game` require `Authorization: Bearer
    <password>` (hash setup: INSTALL.md). Any failure → 403 `forbidden`.
  - `POST /api/game/join` `{ joinCode }` → `200 { sessionToken, seat }`. Errors:
    `code_required` (400), `code_invalid` (404), `game_full` (409).
- `/ws` — WebSocket; `GET /ws?token=<SessionToken>`. On connect the token is
  resolved to a seat (`engine.Connect`); the current `PlayerView` is pushed and
  the other seats get a fresh snapshot. A token already held by a live
  connection is rejected `seat_occupied`. Disconnect releases the seat↔conn
  association (`engine.Disconnect`) and notifies the other seats, but preserves
  game state.
- `/...` — `http.FileServer` over `--static-dir` (see INSTALL.md); `/api` and
  `/ws` are more specific patterns and keep priority. Unmounted when
  `--static-dir` is empty.

### WebSocket messages

Inbound `{ "type", ...payload }`, decoded (`internal/server/wsmsg.go`) into an
`engine.Action` — decoding is pure, engine does all validation:

| type | payload | action |
|------|---------|--------|
| `openingReveal` | `{ targetCardIds: [id] }` | opening reveal |
| `guess` | `{ sourceCardIds: [id], targetCardIds: [id] }` | guess / silver feat |
| `safeCall` | `{ sourceCardIds: [id], targetCardIds: [id, id] }` | safe call |
| `safeChooseReveal` | `{ targetCardIds: [id] }` | multi-hit reveal choice, miss token choice, or the Swap (joker 3) target's pick of its own card |
| `selfReveal` | `{ value }` (rank number, `"silver"`, or `"gold"`) | self-reveal |
| `joker` | `{ joker, sourceCardIds?, targetCardIds?, targetSeats?, value? }` | joker use (array lengths per rank, see `jokerArities` in the engine; Differ/Match send both own cards in `sourceCardIds`; Census sends its rank in `value`) |

Array lengths are validated by the engine (`invalid_input`), not by decoding.

The announced value is derived from the source card (see GAME.md), never sent
directly. Outbound: `state` (full per-seat `PlayerView` snapshot, on connect and
after every change) and `error` `{ code, message }` (invalid action, state
unchanged). Always full snapshots, never deltas — makes reconnect trivial.

## Types and codegen

`frontend/src/generated/engine.ts` is **generated** from the Go engine structs
by [tygo](https://github.com/gzuidhof/tygo) — never hand-edit it. Engine value
types carry **no JSON tags**, so Go marshals them with **PascalCase** field
names; tygo mirrors that, and enums are emitted as real TS enums with iota
values read from the Go source.

- Regenerate after changing any engine value struct: from `backend/`, run
  `go generate ./...` (needs `go install github.com/gzuidhof/tygo@latest`).
- Transport envelopes (WS frames, REST wrappers) are hand-written in
  `frontend/src/types.ts`, which re-exports the generated engine types.

So: change an engine wire struct → regenerate → update hand-written envelopes if
the frame shape changed.

## Level config loading

`cmd/server` resolves the config path and `internal/config` parses the TOML and
validates each level via `engine.ValidateLevel` (level number = file order).
Invalid input is a hard error. The engine's only hardcoded level is
`dummyLevel`, used when the default config file is absent. Path resolution and
the file format are in INSTALL.md.

## Commands

Build, test and dev commands are in [README.md](README.md). Server flags and
deployment are in [INSTALL.md](INSTALL.md).

## Conventions

- Keep the engine **pure**: no HTTP/WS/JSON/config imports in
  `internal/engine`. Rules and view projection stay there; transport only
  serializes and routes.
- All engine access goes through the single server mutex; don't add parallel
  state paths.
- Rule changes belong in `internal/engine` with tests; update
  [GAME.md](GAME.md) when the rules themselves change, and the properties
  below when invariants change.

## Testing strategy

Pure engine → **property-based tests** (Go `pgregory.net/rapid`, ≥100 runs, one
test per Property below, tagged `// Feature: card-finders, Property N: ...`). A
generator seeds the engine, does 4 joins, then streams random **legal** actions;
whole-game invariant Properties (10, 13, 14, 16, 18, 19, 20, 21) are asserted at
**every** reachable state. Distributional claims (uniform shuffle, 0.25 first
player) are covered by range/permutation checks plus a separate statistical test,
not per-run.

Everything else → example/edge/integration tests: REST/WS transport (fake
conns), the `internal/config` loader (the four startup outcomes + named-table
round-trip), and routing/serving. Frontend pure logic (hand sort, Value_Summary
order + counts, seat layout) uses `fast-check` (≥100 runs); components use
Vitest + Testing Library; `useGameSocket` uses a mock WebSocket.

## Engine errors

Validate-first, mutate-second: every rejected action is a no-op leaving state
unchanged. The engine returns typed sentinel errors; transport maps them to a
REST status + envelope or a WS `error` code.

| Engine error | REST status / WS code |
|--------------|-----------------------|
| `ErrLevelRequired` | 400 `level_required` |
| `ErrLevelInvalid` | 400 `level_invalid` |
| any other `CreateGame` error (`ValidateLevel` failure) | 400 `level_config_invalid` |
| `ErrGameExists` | 409 `game_exists` |
| `ErrGameFull` | 409 `game_full` |
| `ErrCodeInvalid` / `ErrNoGame` (join) | 404 `code_invalid` |
| `ErrCodeRequired` | 400 `code_required` |
| any other `Join` error | 400 `bad_request` |
| `ErrNoGame` (WS action) | WS `no_game` |
| `ErrGameEnded` | WS `game_ended` |
| `ErrUnknownAction` | WS `unknown_action` |
| `ErrInvalidInput` | WS `invalid_input` |
| `ErrWrongPhase` | WS `wrong_phase` |
| `ErrNotYourTurn` | WS `not_your_turn` |
| `ErrInvalidOpeningSelection` | WS `invalid_opening_selection` |
| `ErrSpecialOpening` | WS `special_opening` |
| `ErrInvalidSource` | WS `invalid_source` |
| `ErrGoldSource` | WS `gold_source` |
| `ErrInvalidTarget` | WS `invalid_target` |
| `ErrTargetRevealed` | WS `target_revealed` |
| `ErrTargetOwn` | WS `target_own` |
| `ErrSafeCallSpent` | WS `safe_call_spent` |
| `ErrSafeCallTargets` | WS `safe_call_targets` |
| `ErrNoPendingChoice` | WS `no_pending_choice` |
| `ErrInvalidChoice` | WS `invalid_choice` |
| `ErrSelfRevealIncomplete` | WS `self_reveal_incomplete` |
| `ErrGoldRevealIneligible` | WS `gold_reveal_ineligible` |
| `ErrJokerUnavailable` | WS `joker_unavailable` |
| `ErrJokerInvalid` | WS `joker_invalid` |
| `ErrShieldActive` | WS `shield_active` |
| any other `Apply` error | WS `invalid_action` |

## Correctness properties

The engine's universal invariants, one PBT each (tags above). Whole-game ones
(*) are checked at every reachable state.

1. `ValidateLevel` accepts a level iff it meets the limits listed under Level
   in GAME.md, and number 1..50, name 1..60.
2. Creating a game sets Life_Pool and MaxLife to the level's starting lives.
3. Sort_Value = rank (suited), rank+0.3 (silver), rank+0.6 (gold).
4. Hand display order is ascending Sort_Value, ties by suit, stable. It is 
   fixed for the whole game except for a Swap joker.
5. Special_Pool has exactly silver+extraSilver / gold+extraGold entries (real
   plus decoy), distinct ranks per kind, each in 2..Q, formed before any deal;
   a shared rank is two entries. Only the real specials are dealt; decoys
   never collide with real ranks of their kind and are never dealt.
6. Dealing: hands union = combined deck exactly, is a permutation, sizes differ
   by ≤1.
7. First player is a valid seat; order is the clockwise rotation from it.
8. Every seat starts with its Safe_Call unused.
9. An opening reveal tokens the chosen own suited card (silver/gold rejected),
   flips nothing face-up; after all four, phase is Playing.
10. *Card conservation: the card set is constant; each card is face-down xor
    face-up.
11. Every hit is a Pair_Reveal flipping exactly two cards (source + resolved
    target), no life change.
12. Every miss costs exactly one life (floored at 0), tokens exactly one
    un-revealed non-gold target (a safe-call double miss lets the target player
    choose which), leaves the source untokened/face-down. Exception: a miss
    covered by the Shield (joker 10).
13. *Pairing invariant: every non-gold value's face-down count stays even.
14. *Life_Pool is non-negative, never increases except by joker 7, otherwise
    decreases only on an uncovered miss.
15. Safe_Call is consumed exactly once; a second use is rejected.
16. *Gold becomes face-up only via its owner's gold self-reveal (all remaining
    face-down cards gold) or when targeted by a guess / all call targets /
    a Reverse Call (the game-ending gold hit); neither changes life. A gold
    target of a guess under an armed Shield is tokened, not flipped.
17. Suited-rank/silver self-reveal flips exactly the owner's cards of that
    value, nothing else, no life change.
18. *Turn advance lands on the nearest clockwise seat with a face-down card, or
    reports none (terminal) rather than selecting a seat.
19. *Ended=win exactly when all cards face-up; Ended=loss exactly when life
    hits 0 or a gold hit occurs; these are the only transitions into Ended.
20. *A seat's view attaches a value only to face-up or own cards; others'
    face-down cards carry no value.
21. *Every view includes all value-token display values (link tokens carry
    none), every joker with its derived state, the Shield flag, and the full Special_Pool
    (real and decoy entries merged, ascending rank per kind, with
    `SilverInPlay`/`GoldInPlay` = the level's silver/gold counts), with no
    special's owner and no real-vs-decoy marker revealed, in any phase.
22. Session_Tokens are pairwise distinct across occupied seats.
23. Phase stays Waiting through joins 0..3; the 4th join advances past it.
24. Value_Summary orders suited ranks 2..K with every Special_Pool entry (real
    and decoy) by Sort_Value (each special right after its base rank).
25. Value_Summary found = face-up count of that value; total = 4 per suited
    rank. A special shows `1/1` when found; else `0/0` (complete) when the
    kind's found sum equals its in-play count, else `0/1` when the kind's
    entry count equals its in-play count, else `0/1?` (possibly not in
    play). Recomputes on reveal.
26. Seat layout puts the viewer bottom and the next three clockwise seats at
    left, top, right.
27. Jokers are drawn after the first-player pick (the deal is independent of
    the joker count): `Level.Jokers` distinct ranks in 2..13, ascending. A
    joker is ready only once all four suited cards of its rank are face-up, is
    usable only in Playing with no choice pending, and only once.
28. *Link tokens come in adjacent same-kind pairs on face-down cards of one
    hand, never share a card with another token, and are truthful (Same iff
    the two cards have the same identity: suited by rank, specials by kind).
    They are cleared when either card flips, moves in a swap, or is tokened.
29. Anytime jokers (2, 3, 5, 7, 8, 9, 13) and the Shield (10) never change
    CurrentSeat; Pass (12) sets it to the chosen seat; the call jokers (4, 6,
    11) advance it like a guess.
