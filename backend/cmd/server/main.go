// Command server is the Card Finders entrypoint. It resolves the level-config
// path (flag > env > default levels.toml), loads the levels (or the dummy
// fallback), constructs the engine and HTTP server, and listens.
package main

import (
	"cardfinders/internal/config"
	"cardfinders/internal/engine"
	"cardfinders/internal/server"
	crand "crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// version is stamped at build time via -ldflags "-X main.version=<v>".
var version = "dev"

// envLevelsConfig is the environment variable consulted for the level-config
// path when the --config flag is not set.
const envLevelsConfig = "CARD_FINDERS_LEVELS"

// envAuthHash is the environment variable consulted for the Argon2id hash when
// the --auth-hash flag is not set.
const envAuthHash = "CARD_FINDERS_AUTH_HASH"

// defaultLevelsConfig is the path used when neither the --config flag nor the
// CARD_FINDERS_LEVELS env var supplies one. A missing file at this default path
// is a soft outcome: the engine falls back to the built-in dummy level.
const defaultLevelsConfig = "levels.toml"

// resolveConfigPath applies the precedence flag > env > default and reports
// whether the chosen path is "explicit" (came from the flag or the env var)
// rather than the default levels.toml. The explicit flag is passed to
// config.LoadLevels so a missing file is a hard error for an explicit path but
// a soft dummy fallback for the default.
func resolveConfigPath(flagValue string) (path string, explicit bool) {
	if flagValue != "" {
		return flagValue, true
	}
	if env := os.Getenv(envLevelsConfig); env != "" {
		return env, true
	}
	return defaultLevelsConfig, false
}

func randomSeed() (int64, error) {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return 0, err
	}
	seed := int64(binary.BigEndian.Uint64(b[:]) >> 1)
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return seed, nil
}

func main() {
	var (
		configPath string
		staticDir  string
		listenAddr string
		seed       int64
		debug      int
		authHash   string
		showVer    bool
	)
	flag.BoolVar(&showVer, "version", false, "print the version and exit")
	flag.StringVar(&authHash, "auth-hash", "",
		"Argon2id PHC hash protecting game creation (overrides "+envAuthHash+"); required")
	flag.StringVar(&configPath, "config", "",
		"path to the level-config TOML file (overrides "+envLevelsConfig+"; "+
			"default "+defaultLevelsConfig+")")
	flag.StringVar(&staticDir, "static-dir", "",
		"on-disk directory holding the built frontend, served from root "+
			"(empty serves API and WS only)")
	flag.StringVar(&listenAddr, "listen", ":8080", "HTTP listen address")
	flag.Int64Var(&seed, "seed", 0,
		"RNG seed for deterministic dealing/pool selection (0 = random)")
	flag.IntVar(&debug, "v", 0,
		"set logging level (0 default, 1 for info, 2 for debug)")
	flag.Parse()

	if showVer {
		fmt.Println(version)
		return
	}

	level := slog.LevelWarn
	if debug == 1 {
		level = slog.LevelInfo
	} else if debug > 1 {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))

	// Resolve the level-config path and load the levels. A missing file at an
	// explicit path (flag/env) and any malformed/invalid config are hard errors
	// that abort startup; a missing file at the default path is a soft fallback
	// to the built-in dummy level.
	path, explicit := resolveConfigPath(configPath)
	levels, usedDummy, err := config.LoadLevels(path, explicit)
	if err != nil {
		slog.Error("loading level config", "path", path, "err", err)
		os.Exit(1)
	}
	if usedDummy {
		slog.Warn("level config not found; continuing with the built-in dummy level",
			"path", path)
	} else {
		slog.Info("loaded levels", "count", len(levels), "path", path)
	}

	// Seed the RNG deterministically when --seed is given; otherwise draw an
	// unpredictable seed from the OS. Logged so a game can be replayed.
	if seed == 0 {
		seed, err = randomSeed()
		if err != nil {
			slog.Error("generating random seed", "err", err)
			os.Exit(1)
		}
	}
	slog.Info("rng seed", "seed", seed)
	rng := rand.New(rand.NewSource(seed))

	// The auth hash is mandatory; a missing or unparsable value aborts startup.
	if authHash == "" {
		authHash = os.Getenv(envAuthHash)
	}
	if authHash == "" {
		slog.Error("auth hash required", "flag", "--auth-hash", "env", envAuthHash)
		os.Exit(1)
	}
	ph, err := server.ParsePasswordHash(authHash)
	if err != nil {
		slog.Error("invalid auth hash", "err", err)
		os.Exit(1)
	}
	slog.Info("auth configured", "hash", ph.Describe())

	eng := engine.NewEngine(levels, rng)
	srv := server.New(eng, server.Config{
		StaticDir: staticDir,
		AuthHash:  &ph,
	})

	slog.Info("Card Finders listening", "addr", listenAddr, "version", version)
	if err := http.ListenAndServe(listenAddr, srv.Handler()); err != nil {
		slog.Error("http server", "err", err)
		os.Exit(1)
	}
}
