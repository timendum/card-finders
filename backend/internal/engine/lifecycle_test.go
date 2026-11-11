package engine

import (
	"errors"
	"math/rand"
	"testing"
)

// newTestEngine builds an engine with a well-formed level list and a
// deterministic RNG so join codes and tokens are reproducible.
func newTestEngine() *Engine {
	levels := []Level{
		{Number: 1, Name: "One", SilverCount: 0, GoldCount: 0, StartLives: 4},
		{Number: 2, Name: "Two", SilverCount: 2, GoldCount: 1, StartLives: 7},
	}
	return NewEngine(levels, rand.New(rand.NewSource(1)))
}

func TestCreateGameSuccess(t *testing.T) {
	e := newTestEngine()
	code, err := e.CreateGame(2)
	if err != nil {
		t.Fatalf("CreateGame: unexpected error %v", err)
	}
	if code == "" {
		t.Fatal("CreateGame: expected a non-empty join code")
	}
	if e.state == nil {
		t.Fatal("CreateGame: expected game state to be set")
	}
	if e.state.Phase != PhaseWaiting {
		t.Errorf("CreateGame: phase = %d, want PhaseWaiting", e.state.Phase)
	}
	if e.state.LifePool != 7 {
		t.Errorf("CreateGame: LifePool = %d, want 7 (level start lives)", e.state.LifePool)
	}
	if e.state.Level.Number != 2 {
		t.Errorf("CreateGame: level = %d, want 2", e.state.Level.Number)
	}
	if e.state.JoinCode != code {
		t.Errorf("CreateGame: state.JoinCode = %q, want returned code %q", e.state.JoinCode, code)
	}
	if n := e.state.seatsFilled(); n != 0 {
		t.Errorf("CreateGame: seatsFilled() = %d, want 0", n)
	}
}

func TestCreateGameRejections(t *testing.T) {
	t.Run("missing level", func(t *testing.T) {
		e := newTestEngine()
		if _, err := e.CreateGame(0); !errors.Is(err, ErrLevelRequired) {
			t.Errorf("err = %v, want ErrLevelRequired", err)
		}
		if e.state != nil {
			t.Error("expected no game state retained on rejection")
		}
	})
	t.Run("unknown level", func(t *testing.T) {
		e := newTestEngine()
		if _, err := e.CreateGame(99); !errors.Is(err, ErrLevelInvalid) {
			t.Errorf("err = %v, want ErrLevelInvalid", err)
		}
		if e.state != nil {
			t.Error("expected no game state retained on rejection")
		}
	})
	t.Run("game already exists", func(t *testing.T) {
		e := newTestEngine()
		if _, err := e.CreateGame(1); err != nil {
			t.Fatalf("first CreateGame: %v", err)
		}
		if _, err := e.CreateGame(1); !errors.Is(err, ErrGameExists) {
			t.Errorf("err = %v, want ErrGameExists", err)
		}
	})
}

func TestJoinSeatsAndTokens(t *testing.T) {
	e := newTestEngine()
	code, err := e.CreateGame(1)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	seenTokens := map[SessionToken]bool{}
	for i := range 4 {
		seat, token, err := e.Join(code)
		if err != nil {
			t.Fatalf("Join #%d: unexpected error %v", i+1, err)
		}
		if int(seat) != i {
			t.Errorf("Join #%d: seat = %d, want %d (lowest free seat)", i+1, seat, i)
		}
		if token == "" {
			t.Errorf("Join #%d: empty session token", i+1)
		}
		if seenTokens[token] {
			t.Errorf("Join #%d: duplicate token %q", i+1, token)
		}
		seenTokens[token] = true

		// Phase must stay Waiting until the fourth seat fills.
		if i < 3 && e.state.Phase != PhaseWaiting {
			t.Errorf("after %d joins: phase = %d, want PhaseWaiting", i+1, e.state.Phase)
		}
		if n := e.state.seatsFilled(); n != i+1 {
			t.Errorf("after %d joins: seatsFilled() = %d, want %d", i+1, n, i+1)
		}
	}
}

func TestJoinRejections(t *testing.T) {
	t.Run("no game", func(t *testing.T) {
		e := newTestEngine()
		if _, _, err := e.Join("ABCDEF"); !errors.Is(err, ErrNoGame) {
			t.Errorf("err = %v, want ErrNoGame", err)
		}
	})
	t.Run("empty code", func(t *testing.T) {
		e := newTestEngine()
		if _, err := e.CreateGame(1); err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		if _, _, err := e.Join(""); !errors.Is(err, ErrCodeRequired) {
			t.Errorf("err = %v, want ErrCodeRequired", err)
		}
	})
	t.Run("wrong code", func(t *testing.T) {
		e := newTestEngine()
		code, err := e.CreateGame(1)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		if _, _, err := e.Join(code + "X"); !errors.Is(err, ErrCodeInvalid) {
			t.Errorf("err = %v, want ErrCodeInvalid", err)
		}
	})
	t.Run("game full", func(t *testing.T) {
		e := newTestEngine()
		code, err := e.CreateGame(1)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		for i := range 4 {
			if _, _, err := e.Join(code); err != nil {
				t.Fatalf("Join #%d: %v", i+1, err)
			}
		}
		if _, _, err := e.Join(code); !errors.Is(err, ErrGameFull) {
			t.Errorf("err = %v, want ErrGameFull", err)
		}
	})
}

func TestResolveSeat(t *testing.T) {
	e := newTestEngine()
	code, err := e.CreateGame(1)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	tokens := map[SeatID]SessionToken{}
	for i := range 3 {
		seat, token, err := e.Join(code)
		if err != nil {
			t.Fatalf("Join #%d: %v", i+1, err)
		}
		tokens[seat] = token
	}

	for wantSeat, token := range tokens {
		gotSeat, err := e.ResolveSeat(token)
		if err != nil {
			t.Errorf("ResolveSeat(%q): unexpected error %v", token, err)
		}
		if gotSeat != wantSeat {
			t.Errorf("ResolveSeat(%q): seat = %d, want %d", token, gotSeat, wantSeat)
		}
	}

	if _, err := e.ResolveSeat("not-a-real-token"); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("ResolveSeat(unknown): err = %v, want ErrTokenInvalid", err)
	}
}

func TestResolveSeatNoGame(t *testing.T) {
	e := newTestEngine()
	if _, err := e.ResolveSeat("anything"); !errors.Is(err, ErrNoGame) {
		t.Errorf("err = %v, want ErrNoGame", err)
	}
}
