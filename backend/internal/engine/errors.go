package engine

// Typed sentinel errors returned by the engine, one canonical value per
// failure mode.

import "errors"

var (
	ErrGameExists    = errors.New("a game already exists")
	ErrNoGame        = errors.New("no game exists")
	ErrLevelRequired = errors.New("a level selection is required")
	// ErrLevelInvalid: the level is not in the active level list.
	ErrLevelInvalid = errors.New("the selected level is invalid")
)

var (
	ErrCodeRequired = errors.New("a join code is required and must be well-formed")
	ErrCodeInvalid  = errors.New("the join code is invalid")
	ErrGameFull     = errors.New("the game is full")
	ErrTokenInvalid = errors.New("the session token is invalid")
)

var (
	// ErrInsufficientCards: fewer than four cards to deal; the deal is aborted
	// and no seat gets a hand.
	ErrInsufficientCards = errors.New("insufficient cards to deal to four players")
)

var (
	ErrGameEnded   = errors.New("the game has already ended")
	ErrWrongPhase  = errors.New("the action is not allowed in the current phase")
	ErrNotYourTurn = errors.New("it is not that seat's turn")
	// ErrSeatIneligible: the seat has no face-down cards, so the turn advances
	// clockwise instead.
	ErrSeatIneligible = errors.New("the seat is ineligible: it has no face-down cards remaining")
	// ErrInvalidOpeningSelection: the chosen card is missing, not owned, or not
	// face-down.
	ErrInvalidOpeningSelection = errors.New("the opening selection is invalid")
	// ErrSpecialOpening: silver and gold cards cannot be revealed in Opening.
	ErrSpecialOpening = errors.New("a special card cannot be revealed during the opening")
	ErrUnknownAction           = errors.New("unknown action kind")
)

var (
	ErrInvalidSource     = errors.New("a valid owned face-down source card is required")
	ErrGoldSource        = errors.New("a gold card cannot be used as a source card")
	ErrInvalidTarget     = errors.New("a valid target must be selected")
	ErrTargetRevealed    = errors.New("the target is already revealed")
	ErrTargetOwn         = errors.New("only other players' cards may be targeted")
	ErrGoldNotTargetable = errors.New("a gold card can only be revealed by its owner's gold self-reveal")
)

var (
	ErrSafeCallSpent = errors.New("the safe call power is already spent")
	// ErrSafeCallTargets: the two targets must be distinct, face-down, and both
	// held by one other player who has at least two face-down cards.
	ErrSafeCallTargets = errors.New("the safe call target selection is invalid")
	ErrNoPendingChoice = errors.New("no safe call choice is pending")
	ErrInvalidChoice   = errors.New("the chosen target is not one of the pending options")
)

var (
	ErrSelfRevealIncomplete = errors.New("the self-reveal is incomplete: another player still holds that value, or you hold none")
	ErrGoldRevealIneligible = errors.New("the gold self-reveal is ineligible: you still hold a non-gold face-down card")
)
