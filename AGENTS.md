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
  join internally runs special-pool + deal + opening setup.
- `ResolveSeat(token SessionToken) (SeatID, error)` — reconnection.
- `Apply(seat SeatID, action Action) (Result, error)` — single entry point for
  all gameplay actions (`Action` is a tagged union).
- `View(seat SeatID) PlayerView` — per-player projection (hides others' hidden
  values; see GAME.md visibility rules).
- `Phase() Phase`.

Phases are the first gate in every action handler. See GAME.md for phase and
action meaning.

## Transport surfaces

Three independent paths on one `http.ServeMux`:

- `/api/...` — REST (JSON). Errors use `{ "error": { "code", "message" } }`.
  - `GET  /api/levels` → `{ levels: [Level...] }`.
  - `POST /api/game` `{ levelNumber }` → `201 { joinCode }`. Errors:
    `level_required` (400), `level_invalid` (400), `level_config_invalid` (400),
    `game_exists` (409).
  - `POST /api/game/join` `{ joinCode }` → `200 { sessionToken, seat }`. Errors:
    `code_required` (400), `code_invalid` (404), `game_full` (409).
- `/ws` — WebSocket; `GET /ws?token=<SessionToken>`. On connect the token is
  resolved to a seat; the current `PlayerView` is pushed. A token already held
  by a live connection is rejected `seat_occupied`. Disconnect releases the
  seat↔conn association but preserves game state; reconnect resumes.
- `/...` — `http.FileServer` over `--static-dir`, serving the built frontend
  from root (read from disk at request time, not embedded). The Vite bundle
  references its assets with absolute `/assets/...` URLs, so it can only be
  served from root; `/api` and `/ws` are more specific patterns and keep
  priority. Unmounted when `--static-dir` is empty.

### WebSocket messages

Inbound `{ "type", ...payload }`, decoded (`internal/server/wsmsg.go`) into an
`engine.Action` — decoding is pure, engine does all validation:

| type | payload | action |
|------|---------|--------|
| `openingReveal` | `{ cardId }` | opening reveal |
| `guess` | `{ sourceCardId, targetCardId }` | guess / silver feat |
| `safeCall` | `{ sourceCardId, targetCardId, targetCardId2 }` | safe call |
| `safeChooseReveal` | `{ chosenTargetCardId }` | both-hit choice |
| `selfReveal` | `{ value }` (rank number, `"silver"`, or `"gold"`) | self-reveal |

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

`cmd/server` resolves the config path `--config` flag > `CARD_FINDERS_LEVELS`
env > default `levels.toml`; (flag/env = *explicit*). `internal/config` parses +
validates (via `engine.ValidateLevel`, unique `number`). Outcomes:

- Loaded-valid → those levels replace the dummy.
- Default-missing (not explicit) → warn, use built-in `dummyLevel`, continue.
- Explicit-missing → hard error, exit non-zero.
- Malformed / invalid / duplicate number → hard error, exit non-zero.

TOML schema: one named table per level (key = name); inner keys `number`,
`silver`, `gold`, `lives`. The engine's only hardcoded level is `dummyLevel`.

## Commands

Backend (from `backend/`):

- `go build ./...` — build.
- `go test ./...` — unit + property tests (`pgregory.net/rapid`).
- `go run ./cmd/server` — run (flags: `--config`, `--static-dir`, `--listen`,
  `--seed`; `--seed N` = deterministic RNG).
- `go generate ./...` — regenerate frontend engine types (see codegen above).

Frontend (from `frontend/`, Bun):

- `bun install`
- `bun run build` — `tsc --noEmit` then Vite build → `frontend/dist`.
- `bun run typecheck` — type-check only.
- `bun run test` — Vitest once (`vitest run`).
- `bun run dev` — Vite dev server, proxies `/api` and `/ws` to `:8080`.

Run the server from the repo root so `levels.toml` and `frontend/dist` resolve,
or pass absolute paths. See [README.md](README.md) for a full local run.

## Conventions

- Keep the engine **pure**: no HTTP/WS/JSON/config imports in
  `internal/engine`. Rules and view projection stay there; transport only
  serializes and routes.
- All engine access goes through the single server mutex; don't add parallel
  state paths.
- Don't hand-edit generated `engine.ts`; change the Go struct and regenerate.
- Engine wire structs use PascalCase field names (no JSON tags) by design.
- The server always sends full per-seat snapshots, never deltas.
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
| `ErrLevelConfig` | 400 `level_config_invalid` |
| `ErrGameExists` | 409 `game_exists` |
| `ErrGameFull` | 409 `game_full` |
| `ErrCodeInvalid` | 404 `code_invalid` |
| `ErrCodeRequired` | 400 `code_required` |
| `ErrInsufficientCards` | 500 `deal_failed` |
| `ErrNotYourTurn` / `ErrIneligible` | WS `not_your_turn` / `ineligible` |
| `ErrInvalidTarget` | WS `invalid_target` |
| `ErrTargetRevealed` | WS `target_revealed` |
| `ErrInvalidSource` | WS `invalid_source` |
| `ErrGoldNotTargetable` | WS `gold_not_targetable` |
| `ErrSelfRevealInvalid` | WS `self_reveal_invalid` |
| `ErrSafeCallSpent` | WS `safe_call_spent` |
| `ErrSafeCallTargets` | WS `safe_call_targets` |
| `ErrGameEnded` | WS `game_ended` |
Notes: `ErrInvalidSource` covers every bad source on a guess/safe-call (not
owned, face-up, missing, or gold — gold can never be a source). One
`ErrSelfRevealInvalid` covers all ineligible self-reveal branches (gold while a
non-gold card is still held; suited-rank/silver another player still holds).
`ErrGoldNotTargetable` is a separate concern: another player *targeting* gold.
WS connection/reconnection also yields `session_invalid`, `session_required`,
and `seat_occupied` (existing connection preserved). Config failures are
**startup-time** errors in `cmd/server` (see Level config loading), distinct
from these runtime errors.
## Correctness properties

The engine's universal invariants, one PBT each (tags above). Whole-game ones
(*) are checked at every reachable state.

1. `ValidateLevel` accepts a level iff number 1..50, name 1..60, silver even
   0..10, gold 0..11, lives 1..100, and both counts fit the 11 ranks (2..Q).
2. Creating a game sets Life_Pool and MaxLife to the level's starting lives.
3. Sort_Value = rank (suited), rank+0.3 (silver), rank+0.6 (gold).
4. Hand display order is ascending Sort_Value, ties by suit, stable.
5. Special_Pool has exactly the level's silver/gold counts, distinct ranks per
   kind, each in 2..Q, formed before any deal; a shared rank is two entries.
6. Dealing: hands union = combined deck exactly, is a permutation, sizes differ
   by ≤1.
7. First player is a valid seat; order is the clockwise rotation from it.
8. Every seat starts with its Safe_Call unused.
9. An opening reveal tokens the chosen own card, flips nothing face-up; after
   all four, phase is Playing.
10. *Card conservation: the card set is constant; each card is face-down xor
    face-up.
11. Every hit is a Pair_Reveal flipping exactly two cards (source + resolved
    target), no life change.
12. Every miss costs exactly one life (floored at 0), tokens each un-revealed
    target, leaves the source untokened/face-down.
13. *Pairing invariant: every non-gold value's face-down count stays even.
14. *Life_Pool is monotonic non-increasing, bounded [0, start], changes only on
    a miss.
15. Safe_Call is consumed exactly once; a second use is rejected.
16. *Gold becomes face-up only via its owner's gold self-reveal (all remaining
    face-down cards gold), never targeted; no life change.
17. Suited-rank/silver self-reveal flips exactly the owner's cards of that
    value, nothing else, no life change.
18. *Turn advance lands on the nearest clockwise seat with a face-down card, or
    reports none (terminal) rather than selecting a seat.
19. *Ended=win exactly when all cards face-up; Ended=loss exactly when life
    hits 0; these are the only transitions into Ended.
20. *A seat's view attaches a value only to face-up or own cards; others'
    face-down cards carry no value.
21. *Every view includes all token display values and the full Special_Pool,
    with no special's owner revealed.
22. Session_Tokens are pairwise distinct across occupied seats.
23. Phase stays Waiting through joins 0..3; the 4th join advances past it.
24. Value_Summary orders suited ranks 2..K with specials by Sort_Value (each
    special right after its base rank).
25. Value_Summary found = face-up count of that value, total = count in play (4
    per suited rank, 1 per special); recomputes on reveal.
26. Seat layout puts the viewer bottom and the next three clockwise seats at
    left, top, right.
