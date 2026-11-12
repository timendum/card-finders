package server

import (
	"cardfinders/internal/engine"
	"encoding/json"
	"strconv"
	"strings"
)

// inbound is the shared client → server envelope: a discriminant type plus the
// union of all payload fields.
type inbound struct {
	Type          string          `json:"type"`
	SourceCardIDs []engine.CardID `json:"sourceCardIds"`
	TargetCardIDs []engine.CardID `json:"targetCardIds"`
	TargetSeats   []engine.SeatID `json:"targetSeats"`
	Value         json.RawMessage `json:"value"`
	Joker         engine.Rank     `json:"joker"`
}

// outboundState is the `state` frame: the receiving seat's full PlayerView.
type outboundState struct {
	Type string            `json:"type"`
	View engine.PlayerView `json:"view"`
}

// outboundError is the `error` frame: a stable machine code plus a readable
// message.
type outboundError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// decodeAction shapes an inbound envelope into an engine.Action. On an unknown
// type or malformed payload it returns ok=false with a stable error code; it
// does no gameplay validation (the engine does).
func decodeAction(in inbound) (engine.Action, string, bool) {
	a := engine.Action{
		SourceCards: in.SourceCardIDs,
		TargetCards: in.TargetCardIDs,
		TargetSeats: in.TargetSeats,
	}
	switch in.Type {
	case "openingReveal":
		a.Kind = engine.ActionKindOpeningReveal
		return a, "", true
	case "guess":
		a.Kind = engine.ActionKindGuess
		return a, "", true
	case "safeCall":
		a.Kind = engine.ActionKindSafeCall
		return a, "", true
	case "safeChooseReveal":
		a.Kind = engine.ActionKindSafeChooseReveal
		return a, "", true
	case "selfReveal":
		target, ok := decodeSelfRevealValue(in.Value)
		if !ok {
			return engine.Action{}, "bad_message", false
		}
		return engine.Action{
			Kind:            engine.ActionKindSelfReveal,
			SelfRevealValue: target,
		}, "", true
	case "joker":
		a.Kind = engine.ActionKindJoker
		a.Joker = in.Joker
		// Census asks a suited rank, sent in the same optional `value` field.
		if len(in.Value) > 0 {
			target, ok := decodeSelfRevealValue(in.Value)
			if !ok {
				return engine.Action{}, "bad_message", false
			}
			a.SelfRevealValue = target
		}
		return a, "", true
	default:
		return engine.Action{}, "unknown_type", false
	}
}

// decodeSelfRevealValue parses a selfReveal `value`: "silver"/"gold"
// (case-insensitive), or a suited rank as a JSON number or numeric string.
func decodeSelfRevealValue(raw json.RawMessage) (engine.SelfRevealTarget, bool) {
	if len(raw) == 0 {
		return engine.SelfRevealTarget{}, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return decodeSelfRevealString(s)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return rankTarget(n)
	}
	return engine.SelfRevealTarget{}, false
}

func decodeSelfRevealString(s string) (engine.SelfRevealTarget, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "silver":
		return engine.SelfRevealTarget{Kind: engine.SpecialKindSilver}, true
	case "gold":
		return engine.SelfRevealTarget{Kind: engine.SpecialKindGold}, true
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return engine.SelfRevealTarget{}, false
	}
	return rankTarget(n)
}

// rankTarget builds a suited-rank target, rejecting ranks outside 2..13.
func rankTarget(n int) (engine.SelfRevealTarget, bool) {
	if n < 2 || n > 13 {
		return engine.SelfRevealTarget{}, false
	}
	return engine.SelfRevealTarget{Kind: engine.SpecialKindNone, Rank: engine.Rank(n)}, true
}
