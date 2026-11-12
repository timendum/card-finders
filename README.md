# Card Finders

Card Finders is a multiplayer deduction card game.

A single Go process hosts the game engine, a REST + WebSocket API, and (optionally) the
built frontend assets. A React + TypeScript client talks to that process.

## Repository layout

- `backend/` — Go module (`cardfinders`) housing the game.
  - `cmd/server` — executable entrypoint.
  - `internal/engine` — pure, deterministic game engine.
  - `internal/config` — TOML level-configuration loader.
  - `internal/server` — HTTP/WebSocket transport layer.
- `frontend/` — React + TypeScript client (Bun + Vite).
- `levels.toml` — sample external level configuration.

## Prerequisites

- **Go** — to build and run the backend server (`backend/`).
- **Bun** — to install frontend dependencies and build the client
  (`frontend/`). Bun runs the Vite build and the test suite.

## Building the frontend

From the `frontend/` directory:

```
bun install
bun run build
```

`bun install` installs the dependencies pinned in `frontend/bun.lock`.
`bun run build` type-checks the project (`tsc --noEmit`) and runs the Vite
build, emitting the static bundle to `frontend/dist`. The Go server serves that
directory from disk (see below), so a rebuild is picked up without rebuilding
the Go binary.

Other frontend scripts:

- `bun run dev` — Vite dev server with `/api` and `/ws` proxied to the Go
  backend on `http://localhost:8080`, so the browser talks to one origin.
- `bun run typecheck` — type-check only (`tsc --noEmit`).
- `bun run test` — run the Vitest suite once.

## Building and running the backend

The Go toolchain is required. From the `backend/` directory:

```
go build ./...
```

Run the server directly with `go run`, or build and run the produced binary:

```
go run ./cmd/server
```

The server listens on `:8080` by default and logs the resolved level config
and listen address on startup.

### Command-line flags

All flags are optional except `--auth-hash` (or its env var); defaults shown.

| Flag            | Default         | Purpose                                                                 |
| --------------- | --------------- | ----------------------------------------------------------------------- |
| `--auth-hash`   | (required)      | Argon2id PHC hash of the host password. Overrides `CARD_FINDERS_AUTH_HASH`. |
| `--config`      | `""`            | Path to the level-config TOML file. Overrides `CARD_FINDERS_LEVELS`.    |
| `--static-dir`  | `""`            | On-disk directory holding the built frontend, served from root.         |
| `--listen`      | `:8080`         | HTTP listen address.                                                     |
| `--seed`        | `0`             | RNG seed for deterministic dealing/pool selection (`0` = time-based).   |

## Config path resolution

The server resolves the level-config path with the following precedence:

1. **`--config` flag** — if set, this path is used.
2. **`CARD_FINDERS_LEVELS` environment variable** — used when `--config` is not
   set.
3. **Default `levels.toml`** — used when neither of the above is supplied.

A path supplied by the flag or the env var is *explicit*: if the file is
missing or invalid, startup aborts with an error. The default `levels.toml` is
*soft*: if it is missing, the server logs a notice and continues with a
built-in dummy level. A malformed or invalid config at any path is always a
hard error.

Note that the default path and `frontend/dist` are both relative to the
server's working directory. Run the server from the repository root (or pass
absolute paths via `--config` / `--static-dir`) so these resolve as expected.

## Host authentication

Creating a game is protected. `GET /api/levels` and `POST /api/game` require
`Authorization: Bearer <password>`; joining does not. A missing or wrong
password returns `403 forbidden`, and the lobby UI stays on the password form.

The server is configured with an Argon2id hash in PHC form (flag
`--auth-hash`, else env `CARD_FINDERS_AUTH_HASH`), for example
`$argon2id$v=19$m=8192,t=2,p=1$<salt>$<hash>`. Startup aborts if it is missing
or unparsable, and logs the parsed parameters (never salt or hash bytes). The
hash contains `$`, so quote it with single quotes in a shell. The repo has no
hash generator; use any Argon2id tool. Keep memory low (the `m` value is KiB);
verification is limited to 2 concurrent computations, and verdicts are cached
in RAM by SHA-256 of the password.

The password travels in clear text unless TLS is terminated in front of the
server; the server itself does not do TLS.

## Serving the frontend

The Go server mounts three independent surfaces on one listener:

- `/api/...` — REST endpoints (levels, create game, join).
- `/ws` — the WebSocket endpoint.
- `/...` — an on-disk file server rooted at `--static-dir`, serving the built
  frontend from root. The `/api` and `/ws` routes take priority over this
  catch-all mount. Left unmounted when `--static-dir` is empty.

The static assets are read from disk at request time, not embedded in the
binary, so rebuilding the frontend (`bun run build`) is enough to serve new
assets. The static file server is a convenience: you can instead host the
built frontend behind a reverse proxy and point it at `/api` and `/ws`.

## Level configuration

Levels are defined in a TOML file using a named-table-per-level schema. Each
level is a named table whose key is the level name; the inner keys are:

| Key      | Type    | Description                              |
| -------- | ------- | ---------------------------------------- |
| `silver` | integer | Count of silver special cards (even, 0–10). |
| `gold`   | integer | Count of gold special cards (0–11).      |
| `lives`  | integer | Starting lives (1–100).                  |

### Sample `levels.toml`

```toml
# Card Finders sample level configuration.
#
# Each level is a named TOML table. The table key is the level name and the
# inner keys are:
#   (level number = position in this file, starting at 1)
#   silver - count of silver special cards (even, 0..10)
#   gold   - count of gold special cards (0..11)
#   lives  - starting lives (1..100)

["Tutorial One"]
silver = 0
gold = 0
lives = 4

["Level two"]
silver = 4
gold = 2
lives = 3
```

## Putting it together

A typical local run:

```
# 1. Build the frontend bundle into frontend/dist.
cd frontend
bun install
bun run build
cd ..

# 2. Build and run the server from the repo root so levels.toml and
#    frontend/dist resolve relative to the working directory.
go -C backend build ./...
go -C backend run ./cmd/server --config ../levels.toml --static-dir ../frontend/dist --auth-hash '<argon2id PHC hash>'
```

Then open the server at its listen address (default `http://localhost:8080`),
where the frontend is served from root.
