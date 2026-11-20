package engine

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxNameRunes = 20

// defaultEmojis are the names handed out on join. Each is a distinct subject
// (no near-duplicate variants, no faces), mostly animals.
var defaultEmojis = []string{
	"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐨", "🐯",
	"🦁", "🐮", "🐷", "🐸", "🐵", "🐔", "🐧", "🐦", "🦆", "🦉",
	"🦇", "🐺", "🐴", "🦄", "🐝", "🐛", "🦋", "🐌", "🐞", "🐜",
	"🐢", "🐍", "🦎", "🐙", "🦑", "🦀", "🐠", "🐬", "🐳", "🦈",
	"🐊", "🐘", "🦒", "🦓", "🐪", "🦘", "🦔", "🦦", "🦥", "🦩",
	"🌵", "🍄", "🌻", "🍀", "🍎", "🍕", "🍩", "⚽", "🎲", "🚀",
}

// normalizeName trims name and checks it is 1..maxNameRunes runes with no
// control characters.
func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n == 0 || n > maxNameRunes || !utf8.ValidString(name) {
		return "", ErrNameInvalid
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", ErrNameInvalid
		}
	}
	return name, nil
}

// nameTaken reports whether another seat already uses name (case-insensitive).
func (s *GameState) nameTaken(seat SeatID, name string) bool {
	for _, p := range s.Players {
		if p != nil && p.Seat != seat && strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

// defaultName draws a random emoji not used as a name by any seat.
func (e *Engine) defaultName(seat SeatID) string {
	free := make([]string, 0, len(defaultEmojis))
	for _, em := range defaultEmojis {
		if !e.state.nameTaken(seat, em) {
			free = append(free, em)
		}
	}
	return free[e.rng.Intn(len(free))]
}

// applySetName renames the acting seat. It is allowed in any phase and leaves
// the log, LastAction and turn untouched.
func (e *Engine) applySetName(seat SeatID, action Action) (Result, error) {
	s := e.state
	if seat > 3 || s.Players[seat] == nil {
		return ResultNone, ErrInvalidInput
	}
	p := s.Players[seat]
	name, err := normalizeName(action.Name)
	if err != nil {
		return ResultNone, err
	}
	if s.nameTaken(seat, name) {
		return ResultNone, ErrNameTaken
	}
	p.Name = name
	return ResultNone, nil
}
