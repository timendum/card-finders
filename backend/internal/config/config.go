// Package config loads and validates the TOML level configuration.
package config

import (
	"cardfinders/internal/engine"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

// rawLevel decodes one TOML level table; the table key supplies the level name.
type rawLevel struct {
	Silver      *int `toml:"silver"`
	Gold        *int `toml:"gold"`
	Lives       *int `toml:"lives"`
	ExtraSilver *int `toml:"extraSilver"`
	ExtraGold   *int `toml:"extraGold"`
	Jokers      *int `toml:"jokers"`
}

// Defaults applied when silver/gold/lives are omitted from a level table.
const (
	defaultSilver = 0
	defaultGold   = 0
	defaultLives  = 4
	defaultJokers = 4
)

func orDefault(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
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
	md, decErr := toml.DecodeFile(path, &raw)
	if decErr != nil {
		return nil, false, fmt.Errorf("parsing level config %q: %w", path, decErr)
	}

	// Document order of the top-level level tables (map iteration is random).
	var order []string
	for _, k := range md.Keys() {
		if len(k) == 1 {
			order = append(order, k[0])
		}
	}

	return convertAndValidate(path, raw, order)
}

// convertAndValidate builds the Level slice in file order, validating each
// level. A level's Number is its 1-based position in the file.
func convertAndValidate(path string, raw map[string]rawLevel, order []string) (levels []engine.Level, usedDummy bool, err error) {
	if len(raw) == 0 {
		return nil, false, fmt.Errorf("level config %q defines no levels", path)
	}

	out := make([]engine.Level, 0, len(raw))
	for i, name := range order {
		r := raw[name]
		lvl := engine.Level{
			Number:      i + 1,
			Name:        name,
			SilverCount: orDefault(r.Silver, defaultSilver),
			GoldCount:   orDefault(r.Gold, defaultGold),
			StartLives:  orDefault(r.Lives, defaultLives),
			ExtraSilver: orDefault(r.ExtraSilver, 0),
			ExtraGold:   orDefault(r.ExtraGold, 0),
			Jokers:      orDefault(r.Jokers, defaultJokers),
		}
		if vErr := engine.ValidateLevel(lvl); vErr != nil {
			return nil, false, fmt.Errorf("level %q in %q: %w", name, path, vErr)
		}
		out = append(out, lvl)
	}

	return out, false, nil
}
