package app

import (
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestConnectionProbeReachabilityAndLatency(t *testing.T) {
	a := testApp(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	c := object(t, call(a, "alice", "POST", "/connections", map[string]any{"name": "probe", "protocol": "rdp", "targetOs": "windows", "host": host, "port": port}))
	path := "/connections/" + c["id"].(string) + "/probe"
	if w := call(a, "bob", "POST", path, map[string]any{}); w.Code != 404 {
		t.Fatalf("ownership: %d", w.Code)
	}
	started := time.Now()
	success := object(t, call(a, "alice", "POST", path, map[string]any{}))
	if success["reachable"] != true {
		t.Fatalf("reachable: %#v", success)
	}
	latency, ok := success["latencyMs"].(float64)
	if !ok || latency < 0 || latency > float64(time.Since(started).Milliseconds()+1) {
		t.Fatalf("latency: %#v", success)
	}
	checked, err := time.Parse(time.RFC3339Nano, success["checkedAt"].(string))
	if err != nil || checked.Before(started) || checked.After(time.Now()) {
		t.Fatalf("timestamp: %#v", success)
	}
	if w := call(a, "alice", "POST", path, map[string]any{}); w.Code != 429 {
		t.Fatalf("rate limit: %d", w.Code)
	}
	other := object(t, call(a, "alice", "POST", "/connections", map[string]any{"name": "other probe", "protocol": "rdp", "targetOs": "windows", "host": host, "port": port}))
	if w := call(a, "alice", "POST", "/connections/"+other["id"].(string)+"/probe", map[string]any{}); w.Code != 200 {
		t.Fatalf("different targets must not share cooldown: %d %s", w.Code, w.Body.String())
	}
	listener.Close()
	a.mu.Lock()
	a.probes["alice\x00"+c["id"].(string)] = time.Time{}
	a.mu.Unlock()
	w := call(a, "alice", "POST", path, map[string]any{})
	if w.Code != 422 {
		t.Fatalf("unreachable: %d %s", w.Code, w.Body.String())
	}
	var failure struct {
		Probe connectionProbe `json:"probe"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Probe.Reachable || failure.Probe.LatencyMS != nil || failure.Probe.CheckedAt.IsZero() || failure.Error.Code != "TARGET_UNREACHABLE" {
		t.Fatalf("failure: %#v", failure)
	}
}
