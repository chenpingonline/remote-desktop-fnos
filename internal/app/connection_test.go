package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestInlineCredentialIsEncryptedAndAssociated(t *testing.T) {
	a := testApp(t)
	request := map[string]any{"name": "inline PC", "protocol": "rdp", "targetOs": "windows", "host": "192.168.1.24", "port": 3389, "credential": map[string]string{"username": "inline-user", "password": "inline-secret", "domain": "WORK"}}
	w := call(a, "alice", "POST", "/connections", request)
	result := object(t, w)
	if bytes.Contains(w.Body.Bytes(), []byte("inline-secret")) || result["credential"] != nil {
		t.Fatal("credential material in response")
	}
	var c Connection
	if e := a.get("connections", result["id"].(string), "alice", &c); e != nil {
		t.Fatal(e)
	}
	if c.CredentialID == "" {
		t.Fatal("credential not associated")
	}
	login, e := a.login(c, "alice")
	if e != nil || login.Password != "inline-secret" || login.Username != "inline-user" || login.Domain != "WORK" {
		t.Fatal("saved login mismatch", e)
	}
	var meta Credential
	if e = a.get("credentials", c.CredentialID, "alice", &meta); e != nil || meta.Password != "" || meta.Username != "" {
		t.Fatal("metadata contains secret", e)
	}
	rows, e := a.db.Query("SELECT data FROM objects")
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		rows.Scan(&b)
		if bytes.Contains(b, []byte("inline-secret")) {
			t.Fatal("plaintext in SQLite")
		}
	}
}
func TestInlineCredentialRollbackOnConflict(t *testing.T) {
	a := testApp(t)
	id := fixture(t, a)
	var c Connection
	a.get("connections", id, "alice", &c)
	original := c.CredentialID
	c.CredentialID = ""
	c.Version = 0
	input := connectionInput{Connection: c, Credential: &inlineCredential{Username: "new-user", Password: "must-rollback"}}
	w := call(a, "alice", "PATCH", "/connections/"+id, input)
	if w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, kind := range []string{"credentials", "secrets"} {
		var n int
		a.db.QueryRow("SELECT count(*) FROM objects WHERE kind=? AND owner='alice'", kind).Scan(&n)
		if n != 1 {
			t.Fatal("orphan after rollback", kind, n)
		}
	}
	a.get("connections", id, "alice", &c)
	if c.CredentialID != original {
		t.Fatal("original credential replaced on failure")
	}
	bad, _ := json.Marshal(map[string]any{"name": "bad", "protocol": "vnc", "targetOs": "macos", "host": "localhost", "port": 5900, "credential": map[string]string{"username": "not-ARD", "password": "vnc-secret"}})
	var request map[string]any
	json.Unmarshal(bad, &request)
	if call(a, "alice", "POST", "/connections", request).Code != 400 {
		t.Fatal("ARD credentials accepted as VNC password mode")
	}
}

func TestVNCTargetIsAutomaticAcrossPlatforms(t *testing.T) {
	a := testApp(t)
	for _, target := range []string{"", "auto", "windows", "macos", "linux"} {
		result := object(t, call(a, "alice", "POST", "/connections", map[string]any{"name": "VNC automatic", "protocol": "vnc", "targetOs": target, "host": "localhost", "port": 5900, "credential": map[string]string{"password": "vnc-test-secret"}}))
		if result["targetOs"] != "auto" {
			t.Fatalf("target %q incorrectly presented as detected: %#v", target, result)
		}
		var c Connection
		if err := a.get("connections", result["id"].(string), "alice", &c); err != nil {
			t.Fatal(err)
		}
		values := params(&Session{Config: c, Login: Credential{Password: "vnc-test-secret"}})
		if _, ok := values["username"]; ok {
			t.Fatal("VNC must not send RDP username")
		}
		if _, ok := values["domain"]; ok {
			t.Fatal("VNC must not send RDP domain")
		}
	}
	if w := call(a, "alice", "POST", "/connections", map[string]any{"name": "bad", "protocol": "vnc", "targetOs": "invented", "host": "localhost", "port": 5900}); w.Code != 400 {
		t.Fatalf("invalid target accepted: %d", w.Code)
	}
}

func TestNewConnectionClipboardDefaults(t *testing.T) {
	a := testApp(t)
	for _, tc := range []struct {
		protocol  string
		settings  map[string]any
		wantText  string
		wantFiles bool
	}{
		{"rdp", nil, "text", true},
		{"rdp", map[string]any{"clipboard": "disabled", "fileClipboard": false}, "disabled", false},
		{"vnc", nil, "text", false},
	} {
		request := map[string]any{"name": "desktop", "host": "localhost", "port": 3389, "protocol": tc.protocol, "settings": tc.settings}
		result := object(t, call(a, "alice", "POST", "/connections", request))
		settings := result["settings"].(map[string]any)
		if settings["clipboard"] != tc.wantText || (settings["fileClipboard"] == true) != tc.wantFiles {
			t.Fatalf("%s clipboard settings: %v", tc.protocol, settings)
		}
	}
}
