# Card Finders — Game Logic

The authoritative business logic. `AGENTS.md` (technical) and the spec under
`.kiro/specs/card-finders/` defer to this file for rules. The source of truth
for exact behavior is `backend/internal/engine`.

## Concept

Four-player, hidden-hand, cooperative card deduction game. Each player sees only
their own card values. On your turn you name a value you also hold and point at
another player's face-down card; a correct call reveals cards, a wrong call
costs the shared team a life. The team **wins** by revealing every card and
**loses** when the shared life pool hits zero.

## Terms

- **Base_Deck** — 48 suited cards: 4 suits × ranks 2–king (2–10, J=11, Q=12,
  K=13). No ace / no rank 1.
- **Rank** — 2..13.
- **Special_Card** — a silver or gold card. No suit; only ranks 2..queen (king
  excluded).
- **Silver_Card** / **Gold_Card** — specials with Sort_Value `rank + 0.3` /
  `rank + 0.6`.
- **Special_Pool** — the set of rank identities assigned to the level's silvers
  and golds, chosen randomly at game start and revealed to everyone before
  dealing. A silver and a gold may share a rank; two silvers (or two golds) may
  not.
- **Sort_Value** — display-ordering number only (rank, +0.3 silver, +0.6 gold).
- **Level** — config: number, name, silver count (even, 0–10), gold count
  (0–11), starting lives (1–100).
- **Life_Pool** — shared team life counter. Reaches 0 → loss.
- **Token** — a public marker in front of a **face-down** card showing its true
  value to everyone. Placed by an opening reveal or a miss. A face-up card never
  carries a token.
- **Face_Down / Face_Up** — hidden-to-others vs. revealed-to-all.
- **Value_Summary** — shared panel listing every value in play (suited ranks +
  Special_Pool identities) with found/total counts.

## Setup and lifecycle

Phases (engine state machine; phase gates every action):

1. **Waiting** — game created at a level; <4 seats filled. No dealing.
2. **Opening** — entered atomically when the 4th player joins (see Game start
   below). Each player, in turn order, reveals one non-special card of their
   choice; it stays **face-down** but gets a token showing its true value.
   Each player is granted one single-use Safe_Call here.
3. **Playing** — normal turns until win or loss.
4. **AwaitingResponse** — entered from Playing when a Safe_Call hits both
   targets; the targeted player chooses which card is revealed, then play
   resumes (or the game ends).
5. **Ended** — win (all cards face-up) or loss (Life_Pool hit 0). All further
   actions rejected.

### Game start

The 4th join runs these steps in one atomic transition (they are not separate
phases; nothing can observe the intermediate states):

1. **Special_Pool** — silvers/golds get random ranks, forming the Special_Pool,
   revealed to all before any deal.
2. **Deal** — Base_Deck + the level's specials are combined, uniformly shuffled,
   and dealt across the 4 players (hand sizes differ by ≤1, no cards left
   over). Each hand is then laid out in a **fixed, value-sorted order**
   (ascending Sort_Value) that never changes for the rest of the game.
3. **First player** — one of the four is chosen (each 0.25) and opens.

## Hand order and reveal in place

Every hand is kept in a **fixed order** for the whole game. At deal time each
player's cards are sorted ascending by Sort_Value (suited rank, then silver at
`rank + 0.3`, then gold at `rank + 0.6`); ties keep their shuffled deal
order. A card's **position never moves** once dealt.

Revealing a card does not reorder or remove it: a card flips **face-up in
place**, keeping its original position in the hand. So the layout of a hand is
stable from deal to game end — only each card's facing (and any token) changes.

Because the order is public and value-sorted, a card's **position conveys its
relative value** even while its exact value is hidden: a player's leftmost
(lowest-position) card is their lowest-value card, the rightmost their highest.
This positional information is an intended part of the deduction, not a leak;
the card's concrete value stays hidden to other players until it is revealed.

## Turns and actions

Turn order is clockwise over seats 0–3, starting from the chosen first player.
Players with zero face-down cards are skipped. On your turn you take exactly one
of these actions, then the turn passes to the next eligible player.

### Source card and derived value

Guess and Safe_Call do **not** announce a value directly. You play one of your
own face-down cards as the **source card**; the called value is derived from it:

- Silver source → the **silver feat** (calling a target as silver).
- Suited source → that card's **rank**.
- Gold source → **rejected**; gold is never an announceable value.

On a **Hit**, the source card and one matching target both flip **face-up**
together (a Pair_Reveal), no tokens. On a **Miss**, the source stays face-down
and untokened.

### Guess (normal action)

Play one owned face-down source card; target one face-down card of **another**
player. Targets must be another player's face-down, non-gold card.

- **Hit** (target matches derived value): Pair_Reveal — source + target flip
  face-up, no tokens. Turn passes.
- **Miss**: Life_Pool −1 (never below 0); target stays face-down and gets a
  token showing its true value; source stays face-down, untokened. Turn passes.

### Silver feat

A Guess whose source is a silver card calls the target as silver. Hit if the
target is a silver card (Pair_Reveal); otherwise a normal Miss. Target must
belong to another player.

### Safe_Call (one-time power per player)

In place of a Guess: play one owned face-down source card and target **two
distinct** face-down cards of a **single** other player (that player must have
≥2 face-down cards).

- **Exactly one** target matches: Hit, Pair_Reveal (source + that target).
- **Both** match: Hit; the targeted player chooses which one is revealed
  (Pair_Reveal with source).
- **Neither** matches: Miss; Life_Pool −1; **each** un-revealed target gets a
  token; source untokened.

Resolving either way marks the power used and passes the turn. Reusing a spent
Safe_Call is rejected.

### Self-reveal of a fully-held value

In place of a Guess, reveal all your face-down cards of a value you fully hold.
No life change, no tokens; revealed cards flip face-up; turn passes.

- **Suited rank**: allowed only if you hold **every** remaining face-down card
  of that rank across all players.
- **Silver**: allowed only if you hold **every** remaining face-down silver in
  the game.
- **Gold**: see below. Gold is excluded from rank/silver self-reveal.

### Gold self-reveal

A gold card can only be revealed by its owner, and only when **all** of that
owner's face-down cards are gold. Then the player reveals all their gold at once
(no tokens, no life change) and the turn passes. Gold can never be targeted by a
Guess, silver feat, or Safe_Call.

## Win / loss

- **Loss**: Life_Pool reaches 0 (from a Miss). Game ends immediately.
- **Win**: every card across all four players is face-up.
- In **Ended**, all gameplay actions are rejected and state is frozen.

## Visibility rules

The server computes a per-player view:

- You see the true values of **only your own** cards.
- Other players' unrevealed cards appear as face-down: you see their **fixed
  value-sorted position** and facing, but not their value. Position conveys
  relative value (see Hand order and reveal in place), never the exact value.
- A token's displayed value is visible to **everyone** (it sits on a face-down
  card).
- Every face-up card's value is visible to everyone.
- The full Special_Pool is visible to everyone (which identities exist), without
  revealing who holds any particular special.

## Card-count invariant

Every non-gold value is composed so the count of face-down cards of that value
stays **even** until fully revealed (four per suited rank; silvers come in an
even set). This keeps values revealable in Pair_Reveal pairs. Gold is cleared
only via gold self-reveal.
