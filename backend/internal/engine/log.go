package engine

import "strconv"

// LogEntry is one line of the public game log, for events with no visual clue.
// Seq is the Seq of the action that produced it. Actor is nil for system-level
// notices (e.g. all silvers revealed); Target is nil unless the event was aimed
// at another seat. The client renders both relative to the viewer. Text is
// display-ready and excludes the actor and target.
type LogEntry struct {
	Seq    uint64
	Actor  *SeatID
	Target *SeatID
	Text   string
}

// logAction appends an entry attributed to actor, stamped with the current Seq.
func (s *GameState) logAction(actor SeatID, text string) {
	s.Log = append(s.Log, LogEntry{Seq: s.Seq, Actor: &actor, Text: text})
}

// logActionOn appends an actor entry that also names the targeted seat.
func (s *GameState) logActionOn(actor, target SeatID, text string) {
	s.Log = append(s.Log, LogEntry{Seq: s.Seq, Actor: &actor, Target: &target, Text: text})
}

// logSystem appends an entry with no actor.
func (s *GameState) logSystem(text string) {
	s.Log = append(s.Log, LogEntry{Seq: s.Seq, Text: text})
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
	s.logSystem("All silver cards have been revealed")
}

// announcedText renders an announced value for the log.
func announcedText(v *CardValue) string {
	if v == nil {
		return "?"
	}
	if v.Kind == SpecialKindSilver {
		return "silver"
	}
	switch v.Rank {
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	}
	return strconv.Itoa(int(v.Rank))
}
