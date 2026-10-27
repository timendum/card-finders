package engine

import "fmt"

// Level is a game configuration loaded from TOML config at startup; dummyLevel
// is the sole built-in fallback.
type Level struct {
	Number      int    // unique 1..50
	Name        string // 1..60 chars
	SilverCount int    // even, 0..10
	GoldCount   int    // 0..11
	StartLives  int    // 1..100
}

// eligibleSpecialRanks counts the ranks a Special_Card may take: 2..queen, king excluded.
const eligibleSpecialRanks = 11

// ValidateLevel reports whether l is a well-formed Level, returning a descriptive
// error otherwise.
func ValidateLevel(l Level) error {
	if l.Number < 1 || l.Number > 50 {
		return fmt.Errorf("level number must be between 1 and 50, got %d", l.Number)
	}
	if n := len(l.Name); n < 1 || n > 60 {
		return fmt.Errorf("level name must be 1 to 60 characters, got %d", n)
	}
	if l.SilverCount < 0 || l.SilverCount > 10 {
		return fmt.Errorf("silver count must be between 0 and 10, got %d", l.SilverCount)
	}
	if l.SilverCount%2 != 0 {
		return fmt.Errorf("silver count must be even, got %d", l.SilverCount)
	}
	if l.GoldCount < 0 || l.GoldCount > 11 {
		return fmt.Errorf("gold count must be between 0 and 11, got %d", l.GoldCount)
	}
	if l.StartLives < 1 || l.StartLives > 100 {
		return fmt.Errorf("starting lives must be between 1 and 100, got %d", l.StartLives)
	}
	if l.SilverCount > eligibleSpecialRanks {
		return fmt.Errorf("silver count %d exceeds the %d eligible ranks (2..queen)", l.SilverCount, eligibleSpecialRanks)
	}
	if l.GoldCount > eligibleSpecialRanks {
		return fmt.Errorf("gold count %d exceeds the %d eligible ranks (2..queen)", l.GoldCount, eligibleSpecialRanks)
	}
	return nil
}

// dummyLevel is the only Level defined in Go source: the fallback used when no
// config is loaded.
var dummyLevel = Level{
	Number:      1,
	Name:        "Default",
	SilverCount: 0,
	GoldCount:   0,
	StartLives:  4,
}

// DummyLevels returns the built-in fallback level list.
func DummyLevels() []Level { return []Level{dummyLevel} }
