package engine

import "sort"

// LogEvent is the kind of a LogEntry. The client turns an entry into a
// sentence; the engine records facts only.
type LogEvent uint8

const (
	// LogEventCall is a call on face-down cards. Value is the announced value,
	// Slots the called cards and Outcome how it resolved. Sources lists the
	// own cards played when there are two (Reverse Call).
	LogEventCall LogEvent = iota
	// LogEventChoice is the target player's pick after a call: Slots is the
	// chosen card, Outcome LogOutcomeHit (revealed) or LogOutcomeMissed
	// (receives the token).
	LogEventChoice
	// LogEventJoker is any other joker use; Joker says which. Slots, Value and
	// Outcome carry its details. A resolved Swap has the actor's and target's
	// card positions before it in Slots and after it in Sources, actor first.
	LogEventJoker
	// LogEventSilverDone is the system notice that every silver is face-up.
	LogEventSilverDone
)

// LogOutcome is how a logged call or answer resolved.
type LogOutcome uint8

const (
	LogOutcomeNone LogOutcome = iota
	// LogOutcomeHit is a single hit.
	LogOutcomeHit
	// LogOutcomeHitSeveral is a multi-call with several matches, left to the
	// target player's choice.
	LogOutcomeHitSeveral
	// LogOutcomeHitGold is a hit on gold cards: the game is lost.
	LogOutcomeHitGold
	LogOutcomeMissed
	// LogOutcomeShielded is a guess at a gold card covered by the Shield.
	LogOutcomeShielded
	// LogOutcomeYes and LogOutcomeNo answer a Census.
	LogOutcomeYes
	LogOutcomeNo
)

// LogEntry is one line of the public game log, for events with no visual clue.
// Seq is the Seq of the action that produced it. Actor is nil for system-level
// notices; Target is nil unless the event was aimed at another seat. Sources
// and Slots are card positions in ascending order. The client renders the
// sentence, with seats relative to the viewer.
type LogEntry struct {
	Seq     uint64
	Actor   *SeatID
	Target  *SeatID
	Event   LogEvent
	Joker   Rank
	Value   *CardValue
	Sources []int
	Slots   []int
	Outcome LogOutcome
}

// logEntry appends e, stamped with the current Seq.
func (s *GameState) logEntry(e LogEntry) {
	e.Seq = s.Seq
	s.Log = append(s.Log, e)
}

// slotsOf returns the ascending slot numbers of the given cards.
func (s *GameState) slotsOf(ids ...CardID) []int {
	slots := make([]int, len(ids))
	for i, id := range ids {
		slots[i] = s.Slot[id]
	}
	sort.Ints(slots)
	return slots
}

// noteSilverDone logs once when the level had silvers and all are now face-up.
func (s *GameState) noteSilverDone() {
	if s.silverLogged {
		return
	}
	total := 0
	for _, pc := range s.Cards {
		if pc.Card.Kind != SpecialKindSilver {
			continue
		}
		if pc.Facing == FaceDown {
			return
		}
		total++
	}
	if total == 0 {
		return
	}
	s.silverLogged = true
	s.logEntry(LogEntry{Event: LogEventSilverDone})
}
