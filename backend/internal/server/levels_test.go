package server

import (
	"cardfinders/internal/engine"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLevelsListIncludesExtras(t *testing.T) {
	ph, err := ParsePasswordHash(testHashString("s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	levels := []engine.Level{
		{Number: 1, Name: "Plain", SilverCount: 2, GoldCount: 1, StartLives: 5},
		{Number: 2, Name: "Decoys", SilverCount: 2, GoldCount: 1, ExtraSilver: 3, ExtraGold: 2, StartLives: 5},
	}
	eng := engine.NewEngine(levels, rand.New(rand.NewSource(1)))
	h := New(eng, Config{AuthHash: &ph}).Handler()

	req := httptest.NewRequest("GET", "/api/levels", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}

	var body struct {
		Levels []map[string]any `json:"levels"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Levels) != len(levels) {
		t.Fatalf("got %d levels, want %d", len(body.Levels), len(levels))
	}
	for i, lv := range body.Levels {
		for _, key := range []string{"ExtraSilver", "ExtraGold"} {
			if _, ok := lv[key]; !ok {
				t.Errorf("level %d missing %s", i, key)
			}
		}
	}
	if body.Levels[0]["ExtraSilver"] != float64(0) || body.Levels[0]["ExtraGold"] != float64(0) {
		t.Errorf("level 1 extras = %v/%v, want 0/0", body.Levels[0]["ExtraSilver"], body.Levels[0]["ExtraGold"])
	}
	if body.Levels[1]["ExtraSilver"] != float64(3) || body.Levels[1]["ExtraGold"] != float64(2) {
		t.Errorf("level 2 extras = %v/%v, want 3/2", body.Levels[1]["ExtraSilver"], body.Levels[1]["ExtraGold"])
	}
}
