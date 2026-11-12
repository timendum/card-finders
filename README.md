# Card Finders

# Playing

Card Finders is a cooperative card game for four players. Cards are dealt face
down and you see only your own. Together you must turn every card face up
before the team runs out of lives.

## Your turn

Pick one of your face-down cards and one face-down card in another player's
hand. The rank of your card is your call: you are saying the other card has the
same rank. If you are right, both cards flip face up. If you are wrong, the team
loses a life and the card you pointed at gets a token showing its real value,
so everyone learns something from the miss.

Hands are sorted from low to high and never reordered, so a card's position
tells you roughly how high it is. At the start, each player puts a token on one
of their own cards to give everyone a first clue.

## Double guess

Once per game, each player can aim a call at two face-down cards of the same
player instead of one. If one matches, it is a hit. If both match, the
other player chooses which one flips. If neither matches, the team loses a life
and one of the two cards gets a token (the other player chooses which when both
are possible).

## Self reveal

If you hold every remaining face-down card of a rank, nobody else can be
holding one, so you can flip them all at once on your turn. It costs nothing.

## Silver and gold

Some levels add silver and gold cards. They have no suit. A silver card can be
called like any rank, and calling with it claims the target is silver. A gold
card can never be used for a call, and pointing at someone's gold card ends the
game in a loss. Gold is only revealed by its owner, once every card they have
left is gold.

The list of silver and gold ranks in the level is shown to everyone, but some
entries may be decoys that nobody holds.

## Jokers

Each level draws some jokers, tied to ranks. A joker is ready once all four
cards of its rank are face up, and then any player can use it, once. Some can
be used at any time and some only on your turn. Jokers add a life, shield the
next guess from a miss, swap cards between hands, skip a turn, and so on. The
game shows what each one does.

# Developing

To install and run the server, see [INSTALL.md](INSTALL.md). This part is
about working on the code.

## Where the logic is

The repository layout and architecture are in [AGENTS.md](AGENTS.md). Game
rules are described in [GAME.md](GAME.md). In code they live in
`backend/internal/engine`, one file per mechanic:

- `engine.go`, `state.go`, `lifecycle.go`: engine API, state and phases.
- `deck.go`, `deal.go`, `hand.go`, `order.go`, `turn.go`: cards, dealing, hand
  order and turn order.
- `opening.go`, `guess.go`, `safecall.go`, `selfreveal.go`, `multicall.go`:
  the actions.
- `joker.go`, `jokeractions.go`, `jokercalls.go`: jokers.
- `token.go`: value and link tokens.
- `view.go`: what each player is allowed to see.
- `errors.go`: error values returned to the transport.

Tests sit next to the file they cover (`guess_test.go` for `guess.go`).
In `backend/internal/server`, `rest.go` and `ws.go` are the endpoints,
`wsmsg.go` decodes client messages into engine actions, and `hub.go` handles
broadcasting. In `frontend/src`, `screens/` has the join and game screens,
`board/` and `status/` the in-game UI, and `api/` the REST calls, session
storage and the WebSocket hook.

AGENTS.md also describes the wire protocol, error mapping and the invariants
the tests check.

## Building and testing

Prerequisites are Go and Bun. From `backend/`:

```
go build ./...
go test ./...
```

From `frontend/`:

```
bun install
bun run typecheck
bun run test
```

`bun run build` also type-checks. The production build is in
[INSTALL.md](INSTALL.md).

## Working on it

Run the backend from the repository root without a static dir, then start the
Vite dev server, which proxies `/api` and `/ws` to `:8080`:

```
go -C backend run ./cmd/server --config ../levels.toml --auth-hash '<argon2id PHC hash>'
```

```
cd frontend
bun run dev
```

Pass `--seed 1` to get the same deal every time.

`frontend/src/generated/engine.ts` is generated from the Go structs. After
changing an engine struct that goes over the wire, regenerate it as described
under Types and codegen in [AGENTS.md](AGENTS.md).
