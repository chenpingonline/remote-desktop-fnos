package app

import "strconv"

// Only classify fixed messages emitted by the pinned Guacamole 1.6.0 engine.
// Never persist or log arbitrary upstream text, which may contain private data.
func remoteFailure(parts []string) (reason, status string) {
	reason, status = "REMOTE_CONNECTION_FAILED", "256"
	if len(parts) != 3 || parts[0] != "error" {
		return
	}
	code, err := strconv.Atoi(parts[2])
	if err != nil || code < 0 || code > 65535 {
		return
	}
	status = strconv.Itoa(code)
	switch code {
	case 769:
		reason = "REMOTE_AUTH_OR_SECURITY_FAILED"
		switch parts[1] {
		case "Security negotiation failed (wrong security type?)":
			reason = "REMOTE_SECURITY_FAILED"
		case "Authentication failure (invalid credentials?)":
			reason = "REMOTE_AUTH_FAILED"
		}
	case 771:
		reason = "REMOTE_ACCESS_DENIED"
		if parts[1] == "Credentials expired." {
			reason = "REMOTE_CREDENTIAL_EXPIRED"
		}
	case 519:
		switch parts[1] {
		case "SSL/TLS connection failed (untrusted/self-signed certificate?)":
			reason = "REMOTE_TLS_FAILED"
		case "DNS lookup failed (incorrect hostname?)":
			reason = "REMOTE_DNS_FAILED"
		case "Server refused connection (wrong security type?)":
			reason = "REMOTE_SECURITY_FAILED"
		case "Connection failed (server unreachable?)":
			reason = "TARGET_UNREACHABLE"
		}
	case 517:
		reason = "REMOTE_CONNECTION_FAILED"
	}
	return
}
