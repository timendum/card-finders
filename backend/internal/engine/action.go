package engine

// ActionKind is the discriminant of the Action tagged union; it selects which
// of Action's fields are meaningful.
type ActionKind uint8

const (
	ActionKindOpeningReveal ActionKind = iota
	ActionKindGuess
	ActionKindSafeCall
	ActionKindSafeChooseReveal
	ActionKindSelfReveal
)

// SelfRevealTarget names the value chosen to self-reveal: Kind SpecialKindNone
// uses Rank (suited rank); SpecialKindSilver / SpecialKindGold ignore Rank.
type SelfRevealTarget struct {
	Kind SpecialKind
	Rank Rank
}

// Action is the tagged union of every gameplay action. Kind selects the active
// action; only the fields documented for that Kind are meaningful.
type Action struct {
	Kind ActionKind

	// OpeningReveal: the card revealed in the opening phase.
	OpeningCard CardID

	// Guess / SafeCall: for a guess, the targeted face-down card; for a safe
	// call, the first of the two targets.
	TargetCard CardID
	// Guess / SafeCall: the caller's own face-down card. Announces the value
	// (gold rejected) and flips face-up on a hit.
	SourceCard CardID

	// SafeCall: the second target (first is TargetCard).
	TargetCard2 CardID

	// SafeChooseReveal: the card chosen to reveal after the safe call resolves.
	ChosenTarget CardID

	// SelfReveal: the chosen value; all matching cards reveal at once.
	SelfRevealValue SelfRevealTarget
}

// Apply is the single entry point for all gameplay actions. It returns the
// terminal Result (ResultNone while the game continues) or a typed sentinel
// error. Per-kind handlers run their own phase/turn validation.
func (e *Engine) Apply(seat SeatID, action Action) (Result, error) {
	if e.state == nil {
		return ResultNone, ErrNoGame
	}
	if e.state.Phase == PhaseEnded {
		return e.state.GameResult, ErrGameEnded
	}

	switch action.Kind {
	case ActionKindOpeningReveal:
		return e.applyOpeningReveal(seat, action)
	case ActionKindGuess:
		return e.applyGuess(seat, action)
	case ActionKindSafeCall:
		return e.applySafeCall(seat, action)
	case ActionKindSafeChooseReveal:
		return e.applySafeChooseReveal(seat, action)
	case ActionKindSelfReveal:
		return e.applySelfReveal(seat, action)
	default:
		return ResultNone, ErrUnknownAction
	}
}
