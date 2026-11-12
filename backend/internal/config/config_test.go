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

// TestLoadLevels_NumberFromOrder verifies a level's Number is its 1-based
// position in the file, regardless of name ordering.
func TestLoadLevels_NumberFromOrder(t *testing.T) {
	const body = `
["Zeta"]
lives = 4

["Alpha"]
lives = 3

["Mid"]
lives = 2
`
	levels, _, err := LoadLevels(writeTemp(t, body), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"Zeta", "Alpha", "Mid"}
	if len(levels) != len(want) {
		t.Fatalf("got %d levels, want %d", len(levels), len(want))
	}
	for i, l := range levels {
		if l.Number != i+1 || l.Name != want[i] {
			t.Fatalf("level %d = %+v, want number %d name %q", i, l, i+1, want[i])
		}
	}
}

// TestLoadLevels_EmptyConfig verifies a file with no level tables is rejected.
func TestLoadLevels_EmptyConfig(t *testing.T) {
	path := writeTemp(t, "# no levels here\n")

	if _, _, err := LoadLevels(path, true); err == nil {
		t.Fatalf("expected error for empty config, got nil")
	}
}

// TestLoadLevels_ExtrasDefaultZero verifies omitted extras default to 0 and
// omitted jokers default to 4.
func TestLoadLevels_ExtrasDefaultZero(t *testing.T) {
	levels, _, err := LoadLevels(writeTemp(t, validTwoLevels), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, l := range levels {
		if l.ExtraSilver != 0 || l.ExtraGold != 0 || l.Jokers != 4 {
			t.Fatalf("extras should default to 0 and jokers to 4: %+v", l)
		}
	}
}

// TestLoadLevels_ExtrasValid verifies valid extras round trip.
func TestLoadLevels_ExtrasValid(t *testing.T) {
	const withExtras = `
["Decoys"]
number = 1
silver = 2
gold = 1
lives = 4
extraSilver = 3
extraGold = 2
jokers = 9

["NoJokers"]
silver = 2
gold = 1
lives = 4
jokers = 0
`
	levels, _, err := LoadLevels(writeTemp(t, withExtras), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 2 || levels[0].ExtraSilver != 3 || levels[0].ExtraGold != 2 || levels[0].Jokers != 9 || levels[1].Jokers != 0 {
		t.Fatalf("extras not round-tripped: %+v", levels)
	}
}

// TestLoadLevels_ExtrasInvalid verifies negative extras, sums above 11 and
// out-of-range jokers are hard errors.
func TestLoadLevels_ExtrasInvalid(t *testing.T) {
	cases := map[string]string{
		"negative extraSilver": "silver = 2\ngold = 1\nextraSilver = -1",
		"negative extraGold":   "silver = 2\ngold = 1\nextraGold = -1",
		"silver sum above 11":  "silver = 4\ngold = 1\nextraSilver = 8",
		"gold sum above 11":    "silver = 2\ngold = 3\nextraGold = 9",
		"negative jokers":      "silver = 2\ngold = 1\njokers = -1",
		"jokers above 12":      "silver = 2\ngold = 1\njokers = 13",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTemp(t, "[\"Bad\"]\nnumber = 1\nlives = 4\n"+body+"\n")
			if _, _, err := LoadLevels(path, true); err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
}

// TestLoadLevels_RepoLevelsToml verifies the repo-root levels.toml loads.
func TestLoadLevels_RepoLevelsToml(t *testing.T) {
	path := filepath.Join("..", "..", "..", "levels.toml")
	levels, usedDummy, err := LoadLevels(path, true)
	if err != nil {
		t.Fatalf("repo levels.toml failed to load: %v", err)
	}
	if usedDummy || len(levels) == 0 {
		t.Fatalf("expected real levels, got usedDummy=%v levels=%+v", usedDummy, levels)
	}
}
