package engine

const (
	// joinCodeAlphabet omits visually ambiguous characters (0/O, 1/I/L) so
	// codes are easy to read aloud and type.
	joinCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	joinCodeLength   = 6
	// sessionTokenAlphabet uses the full alphanumeric range; tokens are never
	// read aloud.
	sessionTokenAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	sessionTokenLength   = 24
)

// randString draws n characters uniformly from alphabet using the engine's
// seeded RNG, so generation is reproducible for a given seed.
func (e *Engine) randString(alphabet string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[e.rng.Intn(len(alphabet))]
	}
	return string(b)
}

// newSessionToken returns a token unique across occupied seats, re-rolling on
// the rare collision.
func (e *Engine) newSessionToken() SessionToken {
	for {
		tok := SessionToken(e.randString(sessionTokenAlphabet, sessionTokenLength))
		if !e.tokenInUse(tok) {
			return tok
		}
	}
}

func (e *Engine) tokenInUse(tok SessionToken) bool {
	for _, p := range e.state.Players {
		if p != nil && p.Token == tok {
			return true
		}
	}
	return false
}

func (e *Engine) findLevel(number int) (Level, bool) {
	for _, l := range e.levels {
		if l.Number == number {
			return l, true
		}
	}
	return Level{}, false
}

// CreateGame creates the single game at the chosen level, leaving it Waiting.
// The level is re-validated via ValidateLevel so a bad level table surfaces
// here (ErrLevelConfig). Any rejection retains no game state.
func (e *Engine) CreateGame(levelNumber int) (joinCode string, err error) {
	if e.state != nil {
		return "", ErrGameExists
	}
	if levelNumber == 0 {
		return "", ErrLevelRequired
	}
	level, ok := e.findLevel(levelNumber)
	if !ok {
		return "", ErrLevelInvalid
	}
	if err := ValidateLevel(level); err != nil {
		return "", err
	}

	e.state = &GameState{
		Phase:    PhaseWaiting,
		Level:    level,
		JoinCode: e.randString(joinCodeAlphabet, joinCodeLength),
		Cards:    make(map[CardID]*PlacedCard),
		LifePool: level.StartLives,
	}
	return e.state.JoinCode, nil
}

// Join seats a player into the lowest-numbered free seat with a token unique
// across active seats. The fourth join triggers startGame (Special_Pool +
// deal).
// Seat occupancy is derived from Players, not tracked separately.
func (e *Engine) Join(joinCode string) (seat SeatID, token SessionToken, err error) {
	if e.state == nil {
		return 0, "", ErrNoGame
	}
	if joinCode == "" {
		return 0, "", ErrCodeRequired
	}
	if joinCode != e.state.JoinCode {
		return 0, "", ErrCodeInvalid
	}
	if e.state.seatsFilled() >= 4 {
		return 0, "", ErrGameFull
	}

	free := e.firstFreeSeat()
	tok := e.newSessionToken()
	e.state.Players[free] = &Player{
		Seat:  free,
		Token: tok,
	}

	if e.state.seatsFilled() == 4 {
		if err := e.startGame(); err != nil {
			return free, tok, err
		}
	}

	return free, tok, nil
}

// firstFreeSeat returns the lowest-numbered empty seat; only valid when a seat
// is free.
func (e *Engine) firstFreeSeat() SeatID {
	for i := 0; i < 4; i++ {
		if e.state.Players[i] == nil {
			return SeatID(i)
		}
	}
	return 0 // unreachable: callers guard on a free seat existing
}

// ResolveSeat returns the seat bound to token, for reconnection.
func (e *Engine) ResolveSeat(token SessionToken) (SeatID, error) {
	if e.state == nil {
		return 0, ErrNoGame
	}
	for _, p := range e.state.Players {
		if p != nil && p.Token == token {
			return p.Seat, nil
		}
	}
	return 0, ErrTokenInvalid
}
