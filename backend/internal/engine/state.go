package engine

// SessionToken binds a browser to a seat, issued on join; a reconnect presents
// the same token to resume the seat. Unique across active seats.
type SessionToken string

// Phase gates which actions are legal and advances in a fixed order:
// Waiting -> SpecialPool -> Dealing -> Opening -> Playing -> Ended.
type Phase uint8

const (
	PhaseWaiting Phase = iota
	PhaseSpecialPool
	PhaseDealing
	PhaseOpening
	PhasePlaying
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

	LifePool    int    // clamped to [0, Level.StartLives]
	CurrentSeat SeatID // meaningful in Opening/Playing
	OpeningDone [4]bool

	GameResult Result

	// Set while a safe call that hit both targets awaits the targeted player's
	// choice; nil otherwise. The turn has not passed yet — the follow-up
	// ActionKindSafeChooseReveal resolves the Pair_Reveal and passes it.
	PendingSafeChoice *PendingSafeChoice
}

// PendingSafeChoice holds a double-hit Safe_Call awaiting the target's choice.
// The caller's power is already marked used; only the Pair_Reveal and turn pass
// remain, performed by ActionKindSafeChooseReveal once the target chooses.
type PendingSafeChoice struct {
	Caller       SeatID
	TargetPlayer SeatID
	SourceCard   CardID // flipped on resolution
	Option1      CardID // the two matching targets TargetPlayer chooses between
	Option2      CardID
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
