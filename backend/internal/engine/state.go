package engine

// SessionToken binds a browser to a seat, issued on join; a reconnect presents
// the same token to resume the seat. Unique across active seats.
type SessionToken string

// Phase gates which actions are legal and advances in a fixed order:
// Waiting -> Opening -> Playing -> Ended, with
// Playing <-> AwaitingResponse while another seat's response is pending.
// Gating is by equality only; never compare phases by ordinal.
type Phase uint8

const (
	PhaseWaiting Phase = iota
	PhaseOpening
	PhasePlaying
	PhaseAwaitingResponse
	PhaseEnded
)

type Player struct {
	Seat      SeatID
	Token     SessionToken
	PowerUsed bool // one-time Safe_Call spent
}

type Result uint8

const (
	ResultNone Result = iota
	ResultWin
	ResultLoss
)

// GameState is the single authoritative, in-memory state the engine mutates.
// At most one game at a time, always four seats; nothing is persisted.
type GameState struct {
	Phase       Phase
	Level       Level
	SpecialPool SpecialPool
	JoinCode    string

	Players [4]*Player // nil until that seat is filled

	Cards     map[CardID]*PlacedCard
	HandOrder [4][]CardID // each seat's card ids, for stable positions

	// Slot is each card's 1-based value-sorted position within its owning
	// seat's hand, fixed at deal and never mutated for the rest of the game.
	Slot map[CardID]int

	LifePool    int    // clamped to [0, Level.StartLives]
	CurrentSeat SeatID // meaningful in Opening/Playing
	OpeningDone [4]bool

	GameResult Result

	// Seq increments once per applied action, labelling LastAction so clients
	// can distinguish a fresh action from a repeat. LastAction holds the most
	// recent action for display, nil until the first action lands.
	Seq        uint64
	LastAction *LastAction

	// Set while a safe call that hit both targets awaits the targeted player's
	// choice; nil otherwise. The turn has not passed yet — the follow-up
	// ActionKindSafeChooseReveal resolves the Pair_Reveal and passes it.
	PendingSafeChoice *PendingSafeChoice

	// Log is the public, ordered record of events with no visual clue.
	Log          []LogEntry
	silverLogged bool
}

// PendingSafeChoice holds a double-hit Safe_Call awaiting the target's choice.
// The caller's power is already marked used; only the Pair_Reveal and turn pass
// remain, performed by ActionKindSafeChooseReveal once the target chooses.
type PendingSafeChoice struct {
	Caller       SeatID
	TargetPlayer SeatID
	SourceCard   CardID // flipped on resolution (hit only)
	Option1      CardID // the two targets TargetPlayer chooses between
	Option2      CardID
	// Miss marks a double-miss: the life is already lost and the choice picks
	// which single option gets the token; nothing is revealed.
	Miss bool
}

// recordAction bumps Seq once and stores la as the authoritative LastAction,
// stamping it with the new Seq. The full record is kept here including
// SourceCard; privacy gating happens only in the view projection.
func (s *GameState) recordAction(la *LastAction) {
	s.Seq++
	la.Seq = s.Seq
	s.LastAction = la
}

// announcedValueOf returns the public value a source card announces (its natural
// identity), as a fresh pointer for a LastAction record.
func announcedValueOf(source Card) *CardValue {
	v := naturalValue(source)
	return &v
}

// seatsFilled counts non-nil Players, the source of truth for how many joined:
// gates joins, triggers start on the fourth, and feeds PlayerView.SeatsFilled.
func (s *GameState) seatsFilled() int {
	n := 0
	for _, p := range s.Players {
		if p != nil {
			n++
		}
	}
	return n
}
