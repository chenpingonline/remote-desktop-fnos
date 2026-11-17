package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"remote-desktop/internal/rdpcert"
	"strings"
	"testing"
	"time"
)

func setCertificate(a *App, verified bool, hex string) {
	a.cfg.CertificateProbe = func(context.Context, Connection) (rdpcert.Info, error) { return testCertificate(verified, hex), nil }
}
func inspectFor(t *testing.T, a *App, id, owner string) map[string]any {
	t.Helper()
	a.certificateProbes[owner] = time.Time{}
	return object(t, call(a, owner, "POST", "/connections/"+id+"/certificate", nil))
}
func confirmFor(t *testing.T, a *App, id, owner, token string) {
	t.Helper()
	object(t, call(a, owner, "POST", "/connections/"+id+"/trust", map[string]string{"check": token}))
}
func sessionWithCertificate(a *App, id, token string) *httptest.ResponseRecorder {
	return call(a, "alice", "POST", "/sessions", map[string]any{"connectionId": id, "display": Display{800, 600, 96}, "certificateCheck": token})
}
func TestCertificateConfirmationAndUserIsolation(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	setCertificate(a, false, "a")
	inspection := inspectFor(t, a, id, "alice")
	token := inspection["check"].(string)
	if inspection["status"] != "untrusted" || sessionWithCertificate(a, id, token).Code != 409 {
		t.Fatal("unknown certificate allowed")
	}
	if call(a, "bob", "POST", "/connections/"+id+"/trust", map[string]string{"check": token}).Code != 404 {
		t.Fatal("cross owner confirmation")
	}
	confirmFor(t, a, id, "alice", token)
	if call(a, "alice", "POST", "/connections/"+id+"/trust", map[string]string{"check": token}).Code != 409 {
		t.Fatal("confirmation replay accepted")
	}
	s := object(t, sessionWithCertificate(a, id, token))
	session := a.sessions[s["id"].(string)]
	if params(session)["cert-fingerprints"] != "sha256:"+strings.Repeat("a", 64) || params(session)["ignore-cert"] != "false" {
		t.Fatal("session pin not bound")
	}
	if sessionWithCertificate(a, id, token).Code != 409 {
		t.Fatal("session authorization replay")
	}
	other := object(t, call(a, "bob", "POST", "/connections", map[string]any{"name": "same host", "protocol": "rdp", "targetOs": "windows", "host": "localhost", "port": 3389}))
	second := inspectFor(t, a, other["id"].(string), "bob")
	if second["status"] != "untrusted" {
		t.Fatal("another user inherited trust")
	}
	if len(object(t, call(a, "bob", "GET", "/certificate-trust", nil))["items"].([]any)) != 0 {
		t.Fatal("trust list leaked")
	}
	if len(object(t, call(a, "alice", "GET", "/certificate-trust", nil))["items"].([]any)) != 1 {
		t.Fatal("missing trust management entry")
	}
}
func TestCertificateChangeAndRevocationInvalidateSessions(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	setCertificate(a, false, "a")
	token := inspectFor(t, a, id, "alice")["check"].(string)
	confirmFor(t, a, id, "alice", token)
	sid := object(t, sessionWithCertificate(a, id, token))["id"].(string)
	a.sessions[sid].State = "ACTIVE"
	setCertificate(a, true, "b") // Even a new CA-valid leaf cannot override an existing pin.
	changed := inspectFor(t, a, id, "alice")
	token = changed["check"].(string)
	if changed["status"] != "changed" || sessionWithCertificate(a, id, token).Code != 409 {
		t.Fatal("changed certificate auto accepted")
	}
	confirmFor(t, a, id, "alice", token)
	if !terminal(a.sessions[sid].State) {
		t.Fatal("old session survived changed trust")
	}
	sid = object(t, sessionWithCertificate(a, id, token))["id"].(string)
	a.sessions[sid].State = "ACTIVE"
	pending := inspectFor(t, a, id, "alice")["check"].(string)
	object(t, call(a, "alice", "DELETE", "/connections/"+id+"/trust", nil))
	if !terminal(a.sessions[sid].State) || sessionWithCertificate(a, id, pending).Code != 409 {
		t.Fatal("revoked authorization still valid")
	}
	if object(t, call(a, "alice", "GET", "/connections/"+id+"/trust", nil))["trust"] != nil {
		t.Fatal("trust not removed")
	}
}
func TestCertificateChecksBindTargetExpiryAndOrigin(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	setCertificate(a, false, "a")
	token := inspectFor(t, a, id, "alice")["check"].(string)
	body, _ := json.Marshal(map[string]string{"check": token})
	r := httptest.NewRequest("POST", Prefix+"/api/v1/connections/"+id+"/trust", bytes.NewReader(body))
	r.Header.Set("X-Trim-Userid", "alice")
	r.Header.Set("Origin", "https://other-nas.example")
	r.Header.Set("X-CSRF-Token", a.csrfForOrigin("alice", "https://other-nas.example"))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 409 {
		t.Fatal("cross-origin approval accepted", w.Code)
	}
	a.certificateChecks[sha256.Sum256([]byte(token))].Expires = time.Now().Add(-time.Second)
	if call(a, "alice", "POST", "/connections/"+id+"/trust", map[string]string{"check": token}).Code != 409 {
		t.Fatal("expired approval accepted")
	}
	token = inspectFor(t, a, id, "alice")["check"].(string)
	connection := object(t, call(a, "alice", "GET", "/connections/"+id, nil))
	connection["host"] = "other.example"
	object(t, call(a, "alice", "PATCH", "/connections/"+id, connection))
	if call(a, "alice", "POST", "/connections/"+id+"/trust", map[string]string{"check": token}).Code != 409 {
		t.Fatal("approval survived target edit")
	}
}
func TestCertificateGatePrecedesSecretDecryption(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	a.db.Exec("UPDATE objects SET data=? WHERE kind='secrets'", []byte("corrupted"))
	response := sessionWithCertificate(a, id, "")
	if response.Code != 409 || !strings.Contains(response.Body.String(), "CERTIFICATE_CONFIRMATION_REQUIRED") {
		t.Fatal("credentials accessed before certificate authorization", response.Body.String())
	}
}
func TestExpiredCertificateAndProbeRateLimit(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	a.cfg.CertificateProbe = func(context.Context, Connection) (rdpcert.Info, error) {
		info := testCertificate(false, "a")
		info.CanTrust = false
		info.NotAfter = time.Now().Add(-time.Second)
		return info, nil
	}
	inspected := inspectFor(t, a, id, "alice")
	if inspected["status"] != "invalid" {
		t.Fatal("expired certificate state")
	}
	if call(a, "alice", "POST", "/connections/"+id+"/trust", map[string]any{"check": inspected["check"]}).Code != 409 {
		t.Fatal("expired certificate trusted")
	}
	if call(a, "alice", "POST", "/connections/"+id+"/certificate", nil).Code != 429 {
		t.Fatal("probe not rate limited")
	}
}

func TestCertificateTrustPersistsAcrossRestart(t *testing.T) {
	d := t.TempDir()
	cfg := Config{DataDir: d, KeyPath: filepath.Join(d, "key"), Origin: "https://localhost", Gateway: true, Guacd: "127.0.0.1:1", CertificateProbe: func(context.Context, Connection) (rdpcert.Info, error) { return testCertificate(false, "a"), nil }}
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	id := fixture(t, a)
	token := inspectFor(t, a, id, "alice")["check"].(string)
	confirmFor(t, a, id, "alice", token)
	a.Close()
	a, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	inspection := inspectFor(t, a, id, "alice")
	if inspection["status"] != "trusted" {
		t.Fatal("saved trust did not survive restart")
	}
	object(t, sessionWithCertificate(a, id, inspection["check"].(string)))
	object(t, call(a, "alice", "DELETE", "/connections/"+id+"/trust", nil))
	if inspectFor(t, a, id, "alice")["status"] != "untrusted" {
		t.Fatal("revoked trust reused")
	}
}
