package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"remote-desktop/internal/wol"
	"testing"
	"time"
)

func TestWakeAPI(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	if call(a, "alice", "POST", "/connections/"+id+"/wake", nil).Code != 400 {
		t.Fatal("unconfigured WOL accepted")
	}
	var c Connection
	a.get("connections", id, "alice", &c)
	c.WOL = wol.Config{Enabled: true, MAC: "02-11-22-33-44-55", Broadcast: "192.168.1.255"}
	object(t, call(a, "alice", "PATCH", "/connections/"+id, c))
	calls := 0
	a.wakeSender = func(ctx context.Context, host string, w wol.Config) ([]string, error) {
		calls++
		if host != c.Host || w.MAC != "02:11:22:33:44:55" || w.Broadcast != c.WOL.Broadcast {
			t.Fatal("wrong saved target")
		}
		return []string{w.Broadcast}, nil
	}
	if call(a, "bob", "POST", "/connections/"+id+"/wake", nil).Code != 404 {
		t.Fatal("cross-user access")
	}
	if call(a, "alice", "GET", "/connections/"+id+"/wake", nil).Code != 405 {
		t.Fatal("GET mutation")
	}
	r := httptest.NewRequest("POST", Prefix+"/api/v1/connections/"+id+"/wake", nil)
	r.Header.Set("X-Trim-Userid", "alice")
	r.Header.Set("Origin", a.cfg.Origin)
	res := httptest.NewRecorder()
	a.ServeHTTP(res, r)
	if res.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	result := object(t, call(a, "alice", "POST", "/connections/"+id+"/wake", map[string]string{"broadcast": "10.0.0.255"}))
	if _, err := time.Parse(time.RFC3339Nano, result["sentAt"].(string)); err != nil {
		t.Fatal("missing authoritative send time", err)
	}
	history := object(t, call(a, "alice", "GET", "/wake-history", nil))["items"].(map[string]any)
	if history[id] == nil {
		t.Fatal("successful wake not recoverable after refresh")
	}
	lastSent := history[id]
	if calls != 1 {
		t.Fatal("unauthorized send")
	}
	if call(a, "alice", "POST", "/connections/"+id+"/wake", nil).Code != 429 {
		t.Fatal("no rate limit")
	}
	a.wakes["alice"] = time.Time{}
	a.wakeSender = func(context.Context, string, wol.Config) ([]string, error) { return nil, errors.New("send failed") }
	if call(a, "alice", "POST", "/connections/"+id+"/wake", nil).Code != 422 {
		t.Fatal("send failure hidden")
	}
	history = object(t, call(a, "alice", "GET", "/wake-history", nil))["items"].(map[string]any)
	if history[id] != lastSent {
		t.Fatal("failed send replaced successful wake time")
	}
	var count int
	a.db.QueryRow("SELECT count(*) FROM audit_events WHERE action IN ('WAKE_SENT','WAKE_FAILED')").Scan(&count)
	if count != 2 {
		t.Fatal("missing audit", count)
	}
	c.Version++
	c.WOL.MAC = "bad"
	if call(a, "alice", "PATCH", "/connections/"+id, c).Code != 400 {
		t.Fatal("invalid MAC saved")
	}
}

func TestWakeHistoryOwnershipAndOldEvents(t *testing.T) {
	a := testApp(t)
	old := "2026-01-02T03:04:05Z"
	newer := "2026-01-02T03:04:05.001Z"
	for _, event := range []struct{ owner, action, id, at string }{
		{"alice", "WAKE_SENT", "pc", old},
		{"bob", "WAKE_SENT", "private-pc", newer},
		{"alice", "WAKE_SENT", "pc", newer},
		{"alice", "WAKE_FAILED", "pc", "2026-01-03T00:00:00Z"},
	} {
		if _, err := a.db.Exec("INSERT INTO audit_events VALUES(?,?,?,?,?)", randomID(), event.owner, event.action, event.id, event.at); err != nil {
			t.Fatal(err)
		}
	}
	// History must not depend on the first page of recent audit events.
	for i := 0; i < 60; i++ {
		a.audit("alice", "SAVE_connections", "pc")
	}
	history := object(t, call(a, "alice", "GET", "/wake-history", nil))["items"].(map[string]any)
	if len(history) != 1 || history["pc"] != newer {
		t.Fatal("incorrect durable wake history", history)
	}
	bob := object(t, call(a, "bob", "GET", "/wake-history", nil))["items"].(map[string]any)
	if len(bob) != 1 || bob["private-pc"] != newer {
		t.Fatal("wake history leaked across users", bob)
	}
}
