package engine

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultEmojisDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, em := range defaultEmojis {
		if seen[em] {
			t.Errorf("duplicate default emoji %q", em)
		}
		seen[em] = true
	}
	if len(defaultEmojis) < 8 {
		t.Errorf("only %d default emojis", len(defaultEmojis))
	}
}

func TestJoinAssignsDistinctDefaultNames(t *testing.T) {
	e := newTestEngine()
	code, _ := e.CreateGame(1)
	for range 4 {
		if _, _, err := e.Join(code); err != nil {
			t.Fatalf("Join: %v", err)
		}
	}
	seen := map[string]bool{}
	for i, p := range e.View(0).Players {
		if p.Name == "" || seen[p.Name] {
			t.Errorf("seat %d: name %q empty or duplicate", i, p.Name)
		}
		seen[p.Name] = true
	}
}

func TestSetName(t *testing.T) {
	e := newTestEngine()
	code, _ := e.CreateGame(1)
	for range 4 {
		e.Join(code)
	}
	set := func(seat SeatID, name string) error {
		_, err := e.Apply(seat, Action{Kind: ActionKindSetName, Name: name})
		return err
	}
	if err := set(0, "  Alice "); err != nil {
		t.Fatalf("valid name: %v", err)
	}
	if got := e.View(1).Players[0].Name; got != "Alice" {
		t.Errorf("name = %q, want trimmed Alice", got)
	}
	if err := set(0, "alice"); err != nil {
		t.Errorf("re-setting own name (other case): %v", err)
	}
	if err := set(1, "ALICE"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("duplicate name err = %v, want ErrNameTaken", err)
	}
	for _, bad := range []string{"", "   ", "a\nb", strings.Repeat("x", 21)} {
		if err := set(1, bad); !errors.Is(err, ErrNameInvalid) {
			t.Errorf("name %q err = %v, want ErrNameInvalid", bad, err)
		}
	}
}

func TestTakeoverRevertsToDefaultName(t *testing.T) {
	e := newTestEngine()
	code, _ := e.CreateGame(1)
	for range 4 {
		e.Join(code)
	}
	e.Apply(0, Action{Kind: ActionKindSetName, Name: "Bob"})
	seat, _, err := e.Join(code)
	if err != nil || seat != 0 {
		t.Fatalf("takeover seat=%d err=%v", seat, err)
	}
	if got := e.View(0).Players[0].Name; got == "Bob" || got == "" {
		t.Errorf("takeover name = %q", got)
	}
}
