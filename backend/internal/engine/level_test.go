package engine

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: card-finders, Property 1: ValidateLevel accepts a level iff number 1..50, name 1..60, silver even 0..10, gold 0..11, lives 1..100, extras >= 0, and silver+extraSilver and gold+extraGold each fit the 11 ranks.
// **Validates: Requirements 1.3, 1.4, 1.5, 1.6, 7.1**
func TestPropValidateLevelAcceptsExactlyWellFormed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		l := Level{
			Number:      rapid.IntRange(-2, 55).Draw(t, "number"),
			Name:        strings.Repeat("x", rapid.IntRange(0, 65).Draw(t, "nameLen")),
			SilverCount: rapid.IntRange(-2, 13).Draw(t, "silver"),
			GoldCount:   rapid.IntRange(-2, 14).Draw(t, "gold"),
			ExtraSilver: rapid.IntRange(-3, 14).Draw(t, "extraSilver"),
			ExtraGold:   rapid.IntRange(-3, 14).Draw(t, "extraGold"),
			StartLives:  rapid.IntRange(-1, 105).Draw(t, "lives"),
		}

		want := l.Number >= 1 && l.Number <= 50 &&
			len(l.Name) >= 1 && len(l.Name) <= 60 &&
			l.SilverCount >= 0 && l.SilverCount <= 10 && l.SilverCount%2 == 0 &&
			l.GoldCount >= 0 && l.GoldCount <= 11 &&
			l.StartLives >= 1 && l.StartLives <= 100 &&
			l.ExtraSilver >= 0 && l.ExtraGold >= 0 &&
			l.SilverCount+l.ExtraSilver <= 11 &&
			l.GoldCount+l.ExtraGold <= 11

		err := ValidateLevel(l)
		if (err == nil) != want {
			t.Fatalf("ValidateLevel(%+v) err=%v, want accepted=%v", l, err, want)
		}
	})
}

// Boundary examples for the extras bounds, ensuring both sides are exercised.
func TestValidateLevelExtrasBounds(t *testing.T) {
	base := Level{Number: 1, Name: "L", SilverCount: 4, GoldCount: 3, StartLives: 5}
	cases := []struct {
		name         string
		es, eg       int
		wantAccepted bool
	}{
		{"zero extras", 0, 0, true},
		{"silver sum exactly 11", 7, 0, true},
		{"silver sum 12", 8, 0, false},
		{"gold sum exactly 11", 0, 8, true},
		{"gold sum 12", 0, 9, false},
		{"negative extra silver", -1, 0, false},
		{"negative extra gold", 0, -1, false},
	}
	for _, c := range cases {
		l := base
		l.ExtraSilver, l.ExtraGold = c.es, c.eg
		if err := ValidateLevel(l); (err == nil) != c.wantAccepted {
			t.Errorf("%s: err=%v, want accepted=%v", c.name, err, c.wantAccepted)
		}
	}
}
