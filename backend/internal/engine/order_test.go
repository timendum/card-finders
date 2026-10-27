package engine

import (
	"math/rand"
	"testing"
)

// TestNextSeatWrapsClockwise checks that nextSeat advances clockwise and wraps
// from seat 3 back to seat 0.
func TestNextSeatWrapsClockwise(t *testing.T) {
	cases := []struct {
		from SeatID
		want SeatID
	}{
		{from: 0, want: 1},
		{from: 1, want: 2},
		{from: 2, want: 3},
		{from: 3, want: 0},
	}
	for _, tc := range cases {
		if got := nextSeat(tc.from); got != tc.want {
			t.Errorf("nextSeat(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}

	// Four advances from any seat return to the start: a single 4-cycle.
	for start := SeatID(0); start < seatCount; start++ {
		s := start
		for i := 0; i < seatCount; i++ {
			s = nextSeat(s)
		}
		if s != start {
			t.Errorf("four advances from seat %d landed on %d, want %d", start, s, start)
		}
	}
}

// TestStartGameSelectsFirstPlayerAndOpens verifies that the fourth join picks a
// first player in 0..3 and transitions to PhaseOpening. Several seeds guard
// against depending on one particular draw.
func TestStartGameSelectsFirstPlayerAndOpens(t *testing.T) {
	level := Level{Number: 1, Name: "Basic", SilverCount: 0, GoldCount: 0, StartLives: 4}
	for seed := int64(0); seed < 8; seed++ {
		e := NewEngine([]Level{level}, rand.New(rand.NewSource(seed)))
		code, err := e.CreateGame(level.Number)
		if err != nil {
			t.Fatalf("seed %d: CreateGame: %v", seed, err)
		}
		for i := 0; i < 4; i++ {
			if _, _, err := e.Join(code); err != nil {
				t.Fatalf("seed %d: Join #%d: %v", seed, i+1, err)
			}
		}

		if e.state.Phase != PhaseOpening {
			t.Errorf("seed %d: phase = %d, want PhaseOpening", seed, e.state.Phase)
		}
		if e.state.CurrentSeat >= seatCount {
			t.Errorf("seed %d: CurrentSeat = %d, want a seat in 0..3", seed, e.state.CurrentSeat)
		}
	}
}
