// Derives the prompt variant and the enabled/disabled actions for a view.
// Legality here is only plausibility from the shared selection helpers; the
// engine remains the authority and rejects anything illegal.

import { selfRevealOptions, sourceCards, targetCards, targetsForSeat } from "./actions.ts";
import { Outcome, Phase, type PlayerView, type SeatID } from "../types.ts";

export type ActionId = "openingReveal" | "guess" | "safeCall" | "selfReveal" | "safeChooseReveal";

export interface AvailableAction {
  id: ActionId;
  name: string;
  description: string;
  enabled: boolean;
  /** Present only when the action is disabled. */
  disabledReason?: string;
}

export type PromptVariant = "opening" | "standardTurn" | "safeChooseReveal" | "notActing";

export interface AvailableActions {
  variant: PromptVariant;
  actions: AvailableAction[];
  /** The seat the viewer is waiting on; set only for `notActing`. */
  actingSeat?: SeatID;
  /** Whether the awaited seat is making the double-hit choice; set only for `notActing`. */
  awaitingChoice?: boolean;
}

function enabledAction(id: ActionId, name: string, description: string): AvailableAction {
  return { id, name, description, enabled: true };
}

function disabledAction(
  id: ActionId,
  name: string,
  description: string,
  disabledReason: string,
): AvailableAction {
  return { id, name, description, enabled: false, disabledReason };
}

function standardTurnActions(view: PlayerView): AvailableAction[] {
  const sources = sourceCards(view);
  const targets = targetCards(view);
  const canGuess = sources.length > 0 && targets.length > 0;

  const guessName = "Guess";
  const guessDescription = "Click one of your cards, then an opponent's card.";
  let guess: AvailableAction;
  if (canGuess) {
    guess = enabledAction("guess", guessName, guessDescription);
  } else {
    const reason =
      sources.length === 0
        ? "You have no card that can make a guess."
        : "No opponent card can be targeted.";
    guess = disabledAction("guess", guessName, guessDescription, reason);
  }

  const actions = [guess];

  // Once spent the safe call is not offered at all rather than shown disabled.
  if (!view.PowerUsed[view.Seat]) {
    const safeName = "Safe call";
    const safeDescription = "Add a second card of the same opponent to a guess.";
    const seatsWithPair = new Set<SeatID>();
    for (const card of targets) {
      if (targetsForSeat(view, card.Owner).length >= 2) seatsWithPair.add(card.Owner);
    }
    if (sources.length > 0 && seatsWithPair.size > 0) {
      actions.push(enabledAction("safeCall", safeName, safeDescription));
    } else {
      const reason =
        sources.length === 0
          ? "You have no card that can make a call."
          : "No opponent has two cards you can target.";
      actions.push(disabledAction("safeCall", safeName, safeDescription, reason));
    }
  }

  const revealName = "Self-reveal";
  const revealDescription = "Reveal every card you hold of one value.";
  if (selfRevealOptions(view).length > 0) {
    actions.push(enabledAction("selfReveal", revealName, revealDescription));
  } else {
    actions.push(
      disabledAction(
        "selfReveal",
        revealName,
        revealDescription,
        "You don't hold every remaining card of any value.",
      ),
    );
  }

  return actions;
}

export function availableActions(view: PlayerView): AvailableActions {
  if (view.Phase === Phase.AwaitingResponse) {
    if (view.HasResponsePlayer && view.ResponsePlayer === view.Seat) {
      const miss = view.LastAction?.Outcome === Outcome.AwaitingToken;
      return {
        variant: "safeChooseReveal",
        actions: [
          miss
            ? enabledAction(
                "safeChooseReveal",
                "Choose a card to token",
                "Both called cards missed. Pick one of the two highlighted cards to receive the token.",
              )
            : enabledAction(
                "safeChooseReveal",
                "Choose a card to reveal",
                "Both called cards were hit. Pick one of the two highlighted cards to flip.",
              ),
        ],
      };
    }
    return {
      variant: "notActing",
      actions: [],
      actingSeat: view.HasResponsePlayer ? view.ResponsePlayer : view.CurrentSeat,
      awaitingChoice: true,
    };
  }

  if (view.Phase === Phase.Opening && view.YourTurn) {
    return {
      variant: "opening",
      actions: [
        enabledAction("openingReveal", "Reveal a card", "Select one of your cards, then confirm."),
      ],
    };
  }

  if (view.Phase === Phase.Playing && view.YourTurn) {
    return { variant: "standardTurn", actions: standardTurnActions(view) };
  }

  return { variant: "notActing", actions: [], actingSeat: view.CurrentSeat, awaitingChoice: false };
}
