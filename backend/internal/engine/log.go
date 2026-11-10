package engine

import "strconv"

// LogEntry is one line of the public game log, for events with no visual clue.
// Seq is the Seq of the action that produced it. HasActor is false for
// system-level notices (e.g. all silvers revealed). Text is display-ready.
type LogEntry struct {
	Seq      uint64
	Actor    SeatID
	HasActor bool
	Text     string
}

// logAction appends an entry attributed to actor, stamped with the current Seq.
func (s *GameState) logAction(actor SeatID, text string) {
	s.Log = append(s.Log, LogEntry{Seq: s.Seq, Actor: actor, HasActor: true, Text: text})
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
