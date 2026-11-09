package server

import (
	"bytes"
	"cardfinders/internal/engine"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Transport tests for the state frame and inbound messages. The hub holds
// real *websocket.Conn values, so these dial an httptest server instead of
// faking the connection type.

type seatConn struct {
	conn *websocket.Conn
	seat int
}

func startSeatedGame(t *testing.T) []seatConn {
	t.Helper()
	levels := []engine.Level{{Number: 1, Name: "T", SilverCount: 2, GoldCount: 1, StartLives: 5}}
	srv := httptest.NewServer(New(engine.NewEngine(levels, rand.New(rand.NewSource(7))), Config{}).Handler())
	t.Cleanup(srv.Close)

	var created createResponse
	post(t, srv.URL+"/api/game", `{"levelNumber":1}`, &created)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	// Join all four before dialing so every first frame is the dealt state.
	var joins []joinResponse
	for range 4 {
		var j joinResponse
		post(t, srv.URL+"/api/game/join", `{"joinCode":"`+created.JoinCode+`"}`, &j)
		joins = append(joins, j)
	}
	var conns []seatConn
	for _, j := range joins {
		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=" + j.SessionToken
		c, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("dial seat %d: %v", j.Seat, err)
		}
		t.Cleanup(func() { _ = c.CloseNow() })
		conns = append(conns, seatConn{conn: c, seat: j.Seat})
	}
	return conns
}

func post(t *testing.T, url, body string, out any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

// readFrame reads one frame as raw JSON fields.
func readFrame(t *testing.T, c *websocket.Conn) map[string]json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var f map[string]json.RawMessage
	if err := wsjson.Read(ctx, c, &f); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	return f
}

// obj decodes raw JSON into an object, failing the test if it is not one.
func obj(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode object %s: %v", raw, err)
	}
	return m
}

// wantKeys fails unless m contains every key.
func wantKeys(t *testing.T, what string, m map[string]json.RawMessage, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("%s missing %q", what, k)
		}
	}
}

func TestStateFrameShapeOnConnect(t *testing.T) {
	conns := startSeatedGame(t)
	f := readFrame(t, conns[0].conn)

	// Frame shape is exactly {type, view}.
	var keys []string
	for k := range f {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"type", "view"}) || string(f["type"]) != `"state"` {
		t.Fatalf("frame keys = %v, type = %s", keys, f["type"])
	}
	v := obj(t, f["view"])
	wantKeys(t, "view", v, "LastAction", "ResponsePlayer", "HasResponsePlayer", "Cards", "Phase", "YourTurn")
	if string(v["LastAction"]) != "null" || string(v["HasResponsePlayer"]) != "false" {
		t.Errorf("fresh game: LastAction=%s HasResponsePlayer=%s", v["LastAction"], v["HasResponsePlayer"])
	}
	var cards []struct{ Slot int }
	if err := json.Unmarshal(v["Cards"], &cards); err != nil || len(cards) == 0 {
		t.Fatalf("decode Cards: %v (%d)", err, len(cards))
	}
	for _, c := range cards {
		if c.Slot < 1 {
			t.Fatalf("card Slot = %d, want >=1", c.Slot)
		}
	}
}

func TestOpeningRevealBroadcastsLastActionToAllSeats(t *testing.T) {
	conns := startSeatedGame(t)
	var first *seatConn
	var cardID engine.CardID
	for i := range conns {
		var env struct{ View engine.PlayerView }
		raw, _ := json.Marshal(readFrame(t, conns[i].conn))
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		if !env.View.YourTurn {
			continue
		}
		first = &conns[i]
		for _, c := range env.View.Cards {
			if int(c.Owner) == conns[i].seat {
				cardID = c.ID
				break
			}
		}
	}
	if first == nil {
		t.Fatal("no seat has the first turn")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, first.conn, map[string]any{"type": "openingReveal", "cardId": cardID}); err != nil {
		t.Fatal(err)
	}
	for _, c := range conns {
		v := obj(t, readFrame(t, c.conn)["view"])
		la := obj(t, v["LastAction"])
		wantKeys(t, "LastAction", la, "Seq", "Actor", "Kind", "Outcome", "Targets", "SourceCard")
		if string(la["Seq"]) != "1" {
			t.Errorf("seat %d Seq = %s, want 1", c.seat, la["Seq"])
		}
	}
}

func TestStateFrameSerializesAwaitingResponseAndLastAction(t *testing.T) {
	src := engine.CardID(9)
	v := engine.PlayerView{
		Phase:             engine.PhaseAwaitingResponse,
		ResponsePlayer:    2,
		HasResponsePlayer: true,
		LastAction: &engine.LastAction{
			Seq: 3, Actor: 1, Kind: engine.ActionKindSafeCall,
			Outcome: engine.OutcomeAwaitingChoice, Targets: []engine.CardID{4, 5}, SourceCard: &src,
		},
	}
	b, err := json.Marshal(outboundState{Type: "state", View: v})
	if err != nil {
		t.Fatal(err)
	}
	view := obj(t, obj(t, b)["view"])
	if string(view["ResponsePlayer"]) != "2" || string(view["HasResponsePlayer"]) != "true" {
		t.Errorf("ResponsePlayer=%s HasResponsePlayer=%s", view["ResponsePlayer"], view["HasResponsePlayer"])
	}
	la := obj(t, view["LastAction"])
	if string(la["Seq"]) != "3" || string(la["Targets"]) != "[4,5]" || string(la["SourceCard"]) != "9" {
		t.Errorf("LastAction = %v", la)
	}
}

// The five client messages decode to the expected engine actions.
func TestClientMessagesDecode(t *testing.T) {
	silver := engine.SelfRevealTarget{Kind: engine.SpecialKindSilver}
	gold := engine.SelfRevealTarget{Kind: engine.SpecialKindGold}
	cases := []struct {
		name, json string
		want       engine.Action
	}{
		{"openingReveal", `{"type":"openingReveal","cardId":4}`,
			engine.Action{Kind: engine.ActionKindOpeningReveal, OpeningCard: 4}},
		{"guess", `{"type":"guess","sourceCardId":1,"targetCardId":2}`,
			engine.Action{Kind: engine.ActionKindGuess, SourceCard: 1, TargetCard: 2}},
		{"safeCall", `{"type":"safeCall","sourceCardId":1,"targetCardId":2,"targetCardId2":3}`,
			engine.Action{Kind: engine.ActionKindSafeCall, SourceCard: 1, TargetCard: 2, TargetCard2: 3}},
		{"safeChooseReveal", `{"type":"safeChooseReveal","chosenTargetCardId":6}`,
			engine.Action{Kind: engine.ActionKindSafeChooseReveal, ChosenTarget: 6}},
		{"selfReveal rank", `{"type":"selfReveal","value":7}`,
			engine.Action{Kind: engine.ActionKindSelfReveal, SelfRevealValue: engine.SelfRevealTarget{Rank: 7}}},
		{"selfReveal silver", `{"type":"selfReveal","value":"silver"}`,
			engine.Action{Kind: engine.ActionKindSelfReveal, SelfRevealValue: silver}},
		{"selfReveal gold", `{"type":"selfReveal","value":"gold"}`,
			engine.Action{Kind: engine.ActionKindSelfReveal, SelfRevealValue: gold}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in inbound
			if err := json.Unmarshal([]byte(c.json), &in); err != nil {
				t.Fatal(err)
			}
			got, code, ok := decodeAction(in)
			if !ok || code != "" || got != c.want {
				t.Fatalf("decodeAction = %+v, %q, %v; want %+v", got, code, ok, c.want)
			}
		})
	}
	if _, code, ok := decodeAction(inbound{Type: "nope"}); ok || code != "unknown_type" {
		t.Errorf("unknown type: ok=%v code=%q", ok, code)
	}
}
