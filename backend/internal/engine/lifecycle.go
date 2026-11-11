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
	seat, ok := e.joinableSeat()
	if !ok {
		return 0, "", ErrGameFull
	}

	tok := e.newSessionToken()
	if p := e.state.Players[seat]; p != nil {
		p.Token = tok
		return seat, tok, nil
	}
	e.state.Players[seat] = &Player{
		Seat:  seat,
		Token: tok,
	}

	if e.state.Phase == PhaseWaiting && e.state.seatsFilled() == 4 {
		if err := e.startGame(); err != nil {
			return seat, tok, err
		}
	}

	return seat, tok, nil
}

// joinableSeat returns the lowest empty seat, else the lowest seat without a
// live connection. Empty seats are preferred so a player who just joined and
// has not opened their socket yet is not displaced.
func (e *Engine) joinableSeat() (SeatID, bool) {
	for i := range 4 {
		if e.state.Players[i] == nil {
			return SeatID(i), true
		}
	}
	for i := range 4 {
		if !e.state.Players[i].Connected {
			return SeatID(i), true
		}
	}
	return 0, false
}

// Connect marks the seat bound to token as having a live connection, for the
// transport to call when a socket opens. It rejects a seat that is already
// connected, leaving the existing connection in place.
func (e *Engine) Connect(token SessionToken) (SeatID, error) {
	seat, err := e.ResolveSeat(token)
	if err != nil {
		return 0, err
	}
	p := e.state.Players[seat]
	if p.Connected {
		return 0, ErrSeatOccupied
	}
	p.Connected = true
	return seat, nil
}

// Disconnect marks the seat as no longer connected, for the transport to call
// when a socket closes. Before the game starts the seat is released outright so
// the lobby count stays accurate; afterwards the player and token are kept.
func (e *Engine) Disconnect(seat SeatID) {
	if e.state == nil || seat > 3 || e.state.Players[seat] == nil {
		return
	}
	e.state.Players[seat].Connected = false
	if e.state.Phase == PhaseWaiting {
		e.state.Players[seat] = nil
	}
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
