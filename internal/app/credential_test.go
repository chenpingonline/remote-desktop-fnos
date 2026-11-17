package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCredentialDisplayAndPasswordPreservation(t *testing.T) {
	a := testApp(t)
	created := object(t, call(a, "alice", "POST", "/credentials", map[string]any{"name": "work", "type": "rdp", "username": "alice", "domain": "WORK", "password": "original-secret"}))
	id := created["id"].(string)
	list := call(a, "alice", "GET", "/credentials", nil)
	if list.Code != 200 || strings.Contains(list.Body.String(), "original-secret") {
		t.Fatal("invalid credential listing")
	}
	var result struct {
		Items []Credential `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Username != "alice" || result.Items[0].Domain != "WORK" || result.Items[0].Password != "" {
		t.Fatalf("account metadata missing or password exposed")
	}
	updated := object(t, call(a, "alice", "PATCH", "/credentials/"+id, map[string]any{"name": "renamed", "type": "rdp", "username": "bob", "domain": "", "password": "", "version": 1}))
	login, err := a.login(Connection{CredentialID: id}, "alice")
	if err != nil || login.Password != "original-secret" || login.Username != "bob" || login.Domain != "" {
		t.Fatal("blank password did not preserve password while updating account")
	}
	object(t, call(a, "alice", "PATCH", "/credentials/"+id, map[string]any{"name": "renamed", "type": "rdp", "username": "bob", "password": "replacement-secret", "version": updated["version"]}))
	if call(a, "alice", "PATCH", "/credentials/"+id, map[string]any{"name": "stale", "type": "rdp", "username": "stale", "password": "bad-secret", "version": 1}).Code != http.StatusConflict {
		t.Fatal("stale write accepted")
	}
	login, err = a.login(Connection{CredentialID: id}, "alice")
	if err != nil || login.Password != "replacement-secret" {
		t.Fatal("replacement lost on conflict")
	}
	other := object(t, call(a, "bob", "GET", "/credentials", nil))
	if len(other["items"].([]any)) != 0 {
		t.Fatal("cross owner list")
	}
	if call(a, "bob", "PATCH", "/credentials/"+id, map[string]any{"name": "intrusion", "type": "rdp", "username": "bob", "version": 3}).Code != http.StatusNotFound {
		t.Fatal("cross owner update")
	}
}
