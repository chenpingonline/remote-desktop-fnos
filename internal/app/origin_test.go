package app

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func bootstrap(t *testing.T, a *App, uid, origin string) string {
	t.Helper()
	r := httptest.NewRequest("GET", Prefix+"/api/v1/me", nil)
	r.Host = "internal-gateway"
	r.Header.Set("X-Trim-Userid", uid)
	r.Header.Set("X-Remote-Desktop-Origin", origin)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	var u User
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &u)
	return u.CSRF
}
func TestAutomaticOriginBindingWithoutConfiguration(t *testing.T) {
	a := testApp(t)
	// No origin configuration is used in gateway mode, including changed Host.
	a.cfg.Origin = ""
	a.origins = nil
	for _, origin := range []string{"http://nas-a.example", "http://192.168.1.20:5666", "http://[::1]:5666", "https://nas-a.example", "https://nas-b.example:5667", "https://192.168.1.20:5667", "https://[::1]:5667"} {
		token := bootstrap(t, a, "alice", origin)
		r := httptest.NewRequest("POST", Prefix+"/api/v1/groups", strings.NewReader(`{"name":"test"}`))
		r.Host = "rewritten.internal"
		r.Header.Set("X-Trim-Userid", "alice")
		r.Header.Set("Origin", origin)
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(origin, w.Code, w.Body.String())
		}
	}
}
func TestOriginBoundTokenRejectsCrossOriginAndUser(t *testing.T) {
	a := testApp(t)
	token := bootstrap(t, a, "alice", "https://nas-a.example")
	for _, c := range []struct{ uid, origin, fetchSite string }{{"alice", "https://evil.example", ""}, {"bob", "https://nas-a.example", ""}, {"alice", "https://nas-a.example", "cross-site"}, {"alice", "null", ""}} {
		r := httptest.NewRequest("POST", Prefix+"/api/v1/groups", strings.NewReader(`{"name":"must-not-save"}`))
		r.Header.Set("X-Trim-Userid", c.uid)
		r.Header.Set("Origin", c.origin)
		r.Header.Set("X-Remote-Desktop-Origin", "https://nas-a.example")
		r.Header.Set("X-Forwarded-Host", "nas-a.example")
		r.Header.Set("Sec-Fetch-Site", c.fetchSite)
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	rows, _ := a.list("groups", "alice", 100, 0)
	if len(rows) != 0 {
		t.Fatal("rejected request mutated data")
	}
}
func TestBootstrapNoCORSAndConflictingHeaders(t *testing.T) {
	a := testApp(t)
	for _, c := range []struct{ method, origin, declared, site string }{{"OPTIONS", "https://evil.example", "https://evil.example", "cross-site"}, {"GET", "https://evil.example", "https://nas.example", ""}, {"GET", "null", "https://nas.example", ""}, {"GET", "", "https://nas.example", "cross-site"}, {"GET", "", "", ""}} {
		r := httptest.NewRequest(c.method, Prefix+"/api/v1/me", nil)
		r.Header.Set("X-Trim-Userid", "alice")
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.declared != "" {
			r.Header.Set("X-Remote-Desktop-Origin", c.declared)
		}
		r.Header.Set("Sec-Fetch-Site", c.site)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code == 200 || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestTicketCannotBeIssuedFromAnotherOrigin(t *testing.T) {
	a := testApp(t)
	sid := createSession(t, a, fixture(t, a))
	r := httptest.NewRequest("POST", Prefix+"/api/v1/sessions/"+sid+"/ticket", strings.NewReader(`{}`))
	r.Header.Set("X-Trim-Userid", "alice")
	r.Header.Set("Origin", "https://other-nas.example")
	r.Header.Set("X-CSRF-Token", bootstrap(t, a, "alice", "https://other-nas.example"))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestOriginParsing(t *testing.T) {
	for _, s := range []string{"null", "https://a.test/path", "https://user@a.test", "https://a.test?x=1", "https://a.test#x", "https://a.test:99999", "https://a.test,https://evil.test", " https://a.test"} {
		if _, ok := normalizeOrigin(s); ok {
			t.Errorf("accepted %q", s)
		}
	}
	got, ok := normalizeOrigin("https://NAS.example:443")
	if !ok || got != "https://nas.example" {
		t.Fatal(got)
	}
}

func TestHTTPOriginTokenCannotBeReplayedAcrossSchemeOrUser(t *testing.T) {
	a := testApp(t)
	token := bootstrap(t, a, "alice", "http://nas.example:5666")
	for _, c := range []struct{ uid, origin, site string }{
		{"alice", "https://nas.example:5666", ""},
		{"bob", "http://nas.example:5666", ""},
		{"alice", "http://nas.example:5666", "cross-site"},
		{"alice", "http://evil.example:5666", ""},
	} {
		r := httptest.NewRequest("POST", Prefix+"/api/v1/groups", strings.NewReader(`{"name":"rejected"}`))
		r.Header.Set("X-Trim-Userid", c.uid)
		r.Header.Set("Origin", c.origin)
		r.Header.Set("Sec-Fetch-Site", c.site)
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("%+v: %d %s", c, w.Code, w.Body.String())
		}
	}
}

func TestLoginPromptMatchesServerMode(t *testing.T) {
	for _, gateway := range []bool{true, false} {
		t.Run(map[bool]string{true: "gateway", false: "development"}[gateway], func(t *testing.T) {
			a := testApp(t)
			a.cfg.Gateway = gateway
			r := httptest.NewRequest("GET", Prefix+"/api/v1/me", nil)
			r.Header.Set("X-Remote-Desktop-Origin", "http://nas.example:5666")
			w := httptest.NewRecorder()
			a.ServeHTTP(w, r)
			var body struct {
				Error struct {
					Code            string `json:"code"`
					DevLoginAllowed bool   `json:"devLoginAllowed"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 401 || body.Error.Code != "AUTH_REQUIRED" || body.Error.DevLoginAllowed == gateway {
				t.Fatalf("unexpected login prompt: %d %s", w.Code, w.Body.String())
			}
			if gateway {
				r = httptest.NewRequest("POST", Prefix+"/api/v1/dev-login", strings.NewReader(`{"token":"unused"}`))
				w = httptest.NewRecorder()
				a.ServeHTTP(w, r)
				if w.Code != 401 {
					t.Fatalf("gateway accepted developer login: %d", w.Code)
				}
			}
		})
	}
}
