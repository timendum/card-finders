// Package config loads and validates the TOML level configuration.
package config

import (
	"cardfinders/internal/engine"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
)

// rawLevel decodes one TOML level table; the table key supplies the level name.
type rawLevel struct {
	Number int `toml:"number"`
	Silver int `toml:"silver"`
	Gold   int `toml:"gold"`
	Lives  int `toml:"lives"`
}

// LoadLevels parses and validates the level config at path (see AGENTS.md
// "Level config loading"). explicit marks a flag/env path, whose absence is a
// hard error; a missing default path instead falls back to the dummy level
// (usedDummy true).
func LoadLevels(path string, explicit bool) (levels []engine.Level, usedDummy bool, err error) {
	if _, statErr := os.Stat(path); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			if explicit {
				return nil, false, fmt.Errorf("level config %q not found", path)
			}
			return engine.DummyLevels(), true, nil
		}
		return nil, false, fmt.Errorf("level config %q: %w", path, statErr)
	}

	var raw map[string]rawLevel
	if _, decErr := toml.DecodeFile(path, &raw); decErr != nil {
		return nil, false, fmt.Errorf("parsing level config %q: %w", path, decErr)
	}

	return convertAndValidate(path, raw)
}

// convertAndValidate builds a Number-sorted Level slice, validating each level
// and rejecting duplicate Numbers.
func convertAndValidate(path string, raw map[string]rawLevel) (levels []engine.Level, usedDummy bool, err error) {
	if len(raw) == 0 {
		return nil, false, fmt.Errorf("level config %q defines no levels", path)
	}

	seen := make(map[int]string, len(raw))
	out := make([]engine.Level, 0, len(raw))
	for name, r := range raw {
		lvl := engine.Level{
			Number:      r.Number,
			Name:        name,
			SilverCount: r.Silver,
			GoldCount:   r.Gold,
			StartLives:  r.Lives,
		}
		if vErr := engine.ValidateLevel(lvl); vErr != nil {
			return nil, false, fmt.Errorf("level %q in %q: %w", name, path, vErr)
		}
		if other, dup := seen[lvl.Number]; dup {
			return nil, false, fmt.Errorf(
				"duplicate level number %d in %q: used by both %q and %q",
				lvl.Number, path, other, name)
		}
		seen[lvl.Number] = name
		out = append(out, lvl)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, false, nil
}
