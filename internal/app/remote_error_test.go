package app

import "testing"

func TestRemoteFailuresRemainDistinct(t *testing.T) {
	for _, tc := range []struct{ message, code, want string }{
		{"Authentication failure (invalid credentials?)", "769", "REMOTE_AUTH_FAILED"},
		{"Security negotiation failed (wrong security type?)", "769", "REMOTE_SECURITY_FAILED"},
		{"Access denied by server (account locked/disabled?)", "771", "REMOTE_ACCESS_DENIED"},
		{"Credentials expired.", "771", "REMOTE_CREDENTIAL_EXPIRED"},
		{"Insufficient privileges.", "771", "REMOTE_ACCESS_DENIED"},
		{"SSL/TLS connection failed (untrusted/self-signed certificate?)", "519", "REMOTE_TLS_FAILED"},
		{"DNS lookup failed (incorrect hostname?)", "519", "REMOTE_DNS_FAILED"},
		{"Server refused connection (wrong security type?)", "519", "REMOTE_SECURITY_FAILED"},
		{"Connection failed (server unreachable?)", "519", "TARGET_UNREACHABLE"},
		{"private-username-and-secret", "769", "REMOTE_AUTH_OR_SECURITY_FAILED"},
		{"private-username-and-secret", "519", "REMOTE_CONNECTION_FAILED"},
	} {
		reason, status := remoteFailure([]string{"error", tc.message, tc.code})
		if reason != tc.want || status != tc.code {
			t.Fatalf("code %s: got %s/%s want %s", tc.code, reason, status, tc.want)
		}
	}
	for _, parts := range [][]string{{"error"}, {"error", "private", "bad\nsecret"}, {"error", "private", "9999999999999999999999"}, {"error", "private", "-1"}} {
		reason, status := remoteFailure(parts)
		if reason != "REMOTE_CONNECTION_FAILED" || status != "256" {
			t.Fatal("invalid upstream error accepted")
		}
	}
}
