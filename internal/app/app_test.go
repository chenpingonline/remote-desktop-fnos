package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"remote-desktop/internal/guac"
	"remote-desktop/internal/rdpcert"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testApp(t *testing.T) *App {
	t.Helper()
	d := t.TempDir()
	a, e := New(Config{DataDir: d, KeyPath: filepath.Join(d, "key"), Origin: "https://localhost", Gateway: true, Guacd: "127.0.0.1:1"})
	if e != nil {
		t.Fatal(e)
	}
	a.cfg.CertificateProbe = func(context.Context, Connection) (rdpcert.Info, error) { return testCertificate(true, "a"), nil }
	t.Cleanup(a.Close)
	return a
}
func call(a *App, owner, method, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, Prefix+"/api/v1"+path, bytes.NewReader(b))
	r.Header.Set("X-Trim-Userid", owner)
	r.Header.Set("Origin", a.cfg.Origin)
	r.Header.Set("X-CSRF-Token", a.csrfForOrigin(owner, a.cfg.Origin))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func object(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if w.Code >= 300 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var b map[string]any
	json.Unmarshal(w.Body.Bytes(), &b)
	return b
}
func fixture(t *testing.T, a *App) string {
	cr := object(t, call(a, "alice", "POST", "/credentials", map[string]any{"name": "login", "type": "rdp", "username": "alice", "password": "sensitive-password"}))
	c := object(t, call(a, "alice", "POST", "/connections", map[string]any{"name": "PC", "protocol": "rdp", "targetOs": "windows", "host": "localhost", "port": 3389, "credentialId": cr["id"]}))
	return c["id"].(string)
}
func testCertificate(verified bool, hex string) rdpcert.Info {
	return rdpcert.Info{Fingerprint: strings.Repeat(hex, 64), Subject: "CN=test", Issuer: "CN=test", Verified: verified, CanTrust: true, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
}
func certificateToken(t *testing.T, a *App, id, owner string) string {
	t.Helper()
	a.certificateProbes[owner] = time.Time{}
	return object(t, call(a, owner, "POST", "/connections/"+id+"/certificate", nil))["check"].(string)
}
func createSession(t *testing.T, a *App, id string) string {
	s := object(t, call(a, "alice", "POST", "/sessions", map[string]any{"connectionId": id, "display": Display{800, 600, 96}, "timezone": "UTC", "certificateCheck": certificateToken(t, a, id, "alice")}))
	return s["id"].(string)
}
func TestIsolationValidationEncryption(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	if call(a, "bob", "GET", "/connections/"+id, nil).Code != 404 {
		t.Fatal("cross owner access")
	}
	b := object(t, call(a, "bob", "GET", "/connections", nil))
	if len(b["items"].([]any)) != 0 {
		t.Fatal("leaked list")
	}
	if strings.Contains(call(a, "alice", "GET", "/credentials", nil).Body.String(), "sensitive-password") {
		t.Fatal("password in metadata")
	}
	rows, _ := a.db.Query("SELECT data FROM objects")
	defer rows.Close()
	for rows.Next() {
		var data []byte
		rows.Scan(&data)
		if bytes.Contains(data, []byte("sensitive-password")) {
			t.Fatal("plaintext in database")
		}
	}
	rows.Close()
	if call(a, "alice", "POST", "/connections", map[string]any{"ownerUid": "bob"}).Code != 400 {
		t.Fatal("unknown field accepted")
	}
	r := httptest.NewRequest("POST", Prefix+"/api/v1/groups", strings.NewReader(`{"name":"x"}`))
	r.Header.Set("X-Trim-Userid", "alice")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("missing csrf accepted")
	}
}
func TestVersionsAndRevocation(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	sid := createSession(t, a, id)
	object(t, call(a, "alice", "POST", "/sessions/"+sid+"/ticket", nil))
	var c Connection
	a.get("connections", id, "alice", &c)
	c.Name = "new"
	object(t, call(a, "alice", "PATCH", "/connections/"+id, c))
	if call(a, "alice", "PATCH", "/connections/"+id, c).Code != 409 {
		t.Fatal("stale version accepted")
	}
	if a.sessions[sid].State != "CLOSED" {
		t.Fatal("pending session not invalidated")
	}
	if call(a, "alice", "POST", "/sessions/"+sid+"/ticket", nil).Code != 409 {
		t.Fatal("terminal ticket issued")
	}
}
func TestEphemeralSingleClaimAndQuota(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	ref := object(t, call(a, "alice", "POST", "/ephemeral-credentials", map[string]any{"connectionId": id, "password": "one-time"}))["ref"]
	tokens := make([]string, 10)
	for i := range tokens {
		tokens[i] = certificateToken(t, a, id, "alice")
	}
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			body := map[string]any{"connectionId": id, "display": Display{800, 600, 96}, "ephemeralCredentialRef": ref, "certificateCheck": token}
			if call(a, "alice", "POST", "/sessions", body).Code == 201 {
				success.Add(1)
			}
		}(tokens[i])
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatal("claims", success.Load())
	}
	createSession(t, a, id)
	if call(a, "alice", "POST", "/sessions", map[string]any{"connectionId": id, "display": Display{800, 600, 96}}).Code != 429 {
		t.Fatal("quota not enforced")
	}
	for _, s := range a.sessions {
		object(t, call(a, "alice", "DELETE", "/sessions/"+s.ID, nil))
	}
	createSession(t, a, id)
}
func TestTicketCompetitionAndDisplayReadiness(t *testing.T) {
	a := testApp(t)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	a.cfg.Guacd = listener.Addr().String()
	display := make(chan struct{})
	resized := make(chan []string, 1)
	go func() {
		c, e := listener.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		d := guac.NewDecoder(c)
		d.Read()
		c.Write([]byte(guac.Encode("args", "VERSION_1_5_0", "hostname")))
		for i := 0; i < 6; i++ {
			d.Read()
		}
		c.Write([]byte(guac.Encode("ready", "uuid")))
		<-display
		c.Write([]byte(guac.Encode("size", "0", "800", "600") + guac.Encode("sync", "123")))
		for {
			parts, err := d.Read()
			if err != nil {
				return
			}
			if parts[0] == "size" {
				resized <- parts
			}
		}
	}()
	srv := httptest.NewTLSServer(a)
	defer srv.Close()
	a.cfg.Origin = srv.URL
	sid := createSession(t, a, fixture(t, a))
	ticket := object(t, call(a, "alice", "POST", "/sessions/"+sid+"/ticket", nil))["ticket"].(string)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wrong, _, wrongErr := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+Prefix+"/api/v1/tunnel?ticket="+ticket, &websocket.DialOptions{Subprotocols: []string{"guacamole"}, HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {"https://other-nas.example"}, "X-Trim-Userid": {"alice"}}})
	if wrongErr == nil {
		wrong.CloseNow()
		t.Fatal("cross-origin ticket accepted")
	}
	var successes atomic.Int32
	var connsMu sync.Mutex
	var conns []*websocket.Conn
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+Prefix+"/api/v1/tunnel?ticket="+ticket, &websocket.DialOptions{Subprotocols: []string{"guacamole"}, HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {srv.URL}, "X-Trim-Userid": {"alice"}}})
			if e == nil {
				successes.Add(1)
				connsMu.Lock()
				conns = append(conns, ws)
				connsMu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful tunnels %d", successes.Load())
	}
	a.mu.Lock()
	if a.sessions[sid].State != "CONNECTING" {
		t.Error("ready marked active")
	}
	a.mu.Unlock()
	close(display)
	ws := conns[0]
	for i := 0; i < 3; i++ {
		if _, _, e = ws.Read(ctx); e != nil {
			t.Fatal(e)
		}
	}
	a.mu.Lock()
	if a.sessions[sid].State != "ACTIVE" {
		t.Error("display not marked active")
	}
	a.mu.Unlock()
	if err := ws.Write(ctx, websocket.MessageText, []byte(guac.Encode("size", "1920", "1080"))); err != nil {
		t.Fatal(err)
	}
	select {
	case parts := <-resized:
		if strings.Join(parts, ",") != "size,1920,1080" {
			t.Fatalf("unexpected forwarded resize: %v", parts)
		}
	case <-ctx.Done():
		t.Fatal("resize not forwarded to guacd")
	}
	ws.Write(ctx, websocket.MessageText, []byte(guac.Encode("disconnect")))
	deadline := time.Now().Add(time.Second)
	closed := false
	for time.Now().Before(deadline) {
		a.mu.Lock()
		closed = a.sessions[sid].State == "CLOSED"
		a.mu.Unlock()
		if closed {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !closed {
		t.Fatal("explicit disconnect not CLOSED")
	}
	object(t, call(a, "alice", "DELETE", "/sessions/"+sid, nil))
	ws.CloseNow()
	a.mu.Lock()
	if a.sessions[sid].Login.Password != "" {
		t.Error("login retained")
	}
	a.mu.Unlock()
}
func TestDevelopmentIgnoresGatewayHeaders(t *testing.T) {
	a := testApp(t)
	a.cfg.Gateway = false
	a.cfg.DevToken = "long-local-test-development-token"
	if call(a, "alice", "GET", "/me", nil).Code != 401 {
		t.Fatal("forged gateway headers accepted")
	}
}
func TestBackupAndExclusiveDataLock(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	if _, e := New(a.cfg); e == nil {
		t.Fatal("duplicate data owner")
	}
	dest := filepath.Join(t.TempDir(), "backup")
	if e := a.Backup(dest); e != nil {
		t.Fatal(e)
	}
	backup, e := New(Config{DataDir: dest, KeyPath: filepath.Join(dest, "master.key"), Origin: "https://localhost", Gateway: true})
	if e != nil {
		t.Fatal(e)
	}
	defer backup.Close()
	var c Connection
	if e = backup.get("connections", id, "alice", &c); e != nil {
		t.Fatal(e)
	}
	login, e := backup.login(c, "alice")
	if e != nil || login.Password != "sensitive-password" {
		t.Fatal("restored credential invalid", e)
	}
}
func TestSettingsAdministratorAndPersistence(t *testing.T) {
	a := testApp(t)
	if call(a, "alice", "PATCH", "/settings", Limits{1, 2, 5, 1, 1}).Code != 403 {
		t.Fatal("unprivileged settings accepted")
	}
	r := httptest.NewRequest("PATCH", Prefix+"/api/v1/settings", strings.NewReader(`{"perUserLimit":1,"totalLimit":2,"idleTimeoutMinutes":5,"maxDurationHours":1,"version":1}`))
	r.Header.Set("X-Trim-Userid", "admin")
	r.Header.Set("X-Trim-Isadmin", "true")
	r.Header.Set("Origin", a.cfg.Origin)
	r.Header.Set("X-CSRF-Token", a.csrfForOrigin("admin", a.cfg.Origin))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	object(t, w)
	if a.limits.PerUser != 1 {
		t.Fatal("limits not updated")
	}
}
