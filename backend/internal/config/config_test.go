package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp writes contents to a levels.toml file in a fresh temp dir and
// returns its path.
func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "levels.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

const validTwoLevels = `
["Level One"]
number = 1
silver = 2
gold = 1
lives = 4

["Level Two"]
number = 2
silver = 4
gold = 2
lives = 3
`

// TestLoadLevels_LoadedValid verifies a valid file yields exactly those levels,
// sorted by Number, with the dummy absent.
func TestLoadLevels_LoadedValid(t *testing.T) {
	path := writeTemp(t, validTwoLevels)

	levels, usedDummy, err := LoadLevels(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usedDummy {
		t.Fatalf("usedDummy = true, want false for a loaded-valid file")
	}
	if len(levels) != 2 {
		t.Fatalf("got %d levels, want 2", len(levels))
	}
	if levels[0].Number != 1 || levels[1].Number != 2 {
		t.Fatalf("levels not sorted by Number: %+v", levels)
	}
	if levels[0].Name != "Level One" || levels[1].Name != "Level Two" {
		t.Fatalf("names not mapped from table keys: %+v", levels)
	}
	if levels[0].SilverCount != 2 || levels[0].GoldCount != 1 || levels[0].StartLives != 4 {
		t.Fatalf("inner keys mismapped: %+v", levels[0])
	}
	for _, l := range levels {
		if l.Name == "Default" {
			t.Fatalf("dummy level leaked into loaded result: %+v", l)
		}
	}
}

// TestLoadLevels_DefaultMissing verifies that an absent, non-explicit path falls
// back to the single dummy level with no error.
func TestLoadLevels_DefaultMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	levels, usedDummy, err := LoadLevels(path, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !usedDummy {
		t.Fatalf("usedDummy = false, want true for default-missing")
	}
	if len(levels) != 1 || levels[0].Name != "Default" {
		t.Fatalf("expected single dummy level, got %+v", levels)
	}
}

// TestLoadLevels_ExplicitMissing verifies an explicit path to an absent file is
// a hard error with no dummy fallback.
func TestLoadLevels_ExplicitMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	levels, usedDummy, err := LoadLevels(path, true)
	if err == nil {
		t.Fatalf("expected error for explicit-missing, got nil")
	}
	if usedDummy {
		t.Fatalf("usedDummy = true, want false on error")
	}
	if levels != nil {
		t.Fatalf("expected nil levels on error, got %+v", levels)
	}
}

// TestLoadLevels_Malformed verifies invalid TOML is a hard error.
func TestLoadLevels_Malformed(t *testing.T) {
	path := writeTemp(t, "this is not = valid = toml [[[")

	if _, usedDummy, err := LoadLevels(path, false); err == nil {
		t.Fatalf("expected parse error, got nil")
	} else if usedDummy {
		t.Fatalf("usedDummy = true, want false on parse error")
	}
}

// TestLoadLevels_InvalidLevel verifies a level failing ValidateLevel (odd
// silver) is a hard error.
func TestLoadLevels_InvalidLevel(t *testing.T) {
	const oddSilver = `
["Bad Level"]
number = 1
silver = 3
gold = 1
lives = 4
`
	path := writeTemp(t, oddSilver)

	if _, _, err := LoadLevels(path, true); err == nil {
		t.Fatalf("expected validation error for odd silver, got nil")
	}
}

// TestLoadLevels_DuplicateNumber verifies two levels sharing a Number is a hard
// error.
func TestLoadLevels_DuplicateNumber(t *testing.T) {
	const dup = `
["Level A"]
number = 1
silver = 2
gold = 1
lives = 4

["Level B"]
number = 1
silver = 4
gold = 2
lives = 3
`
	path := writeTemp(t, dup)

	if _, _, err := LoadLevels(path, true); err == nil {
		t.Fatalf("expected duplicate-number error, got nil")
	}
}

// TestLoadLevels_EmptyConfig verifies a file with no level tables is rejected.
func TestLoadLevels_EmptyConfig(t *testing.T) {
	path := writeTemp(t, "# no levels here\n")

	if _, _, err := LoadLevels(path, true); err == nil {
		t.Fatalf("expected error for empty config, got nil")
	}
}
