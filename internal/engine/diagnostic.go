package engine

import (
	"bytes"
	"sync"
)

// diagnosticWriter drops raw engine output. Only fixed non-sensitive categories
// reach the callback. Lines longer than 4 KiB are discarded, not accumulated.
type diagnosticWriter struct {
	mu       sync.Mutex
	pending  []byte
	dropping bool
	emit     func(string)
}

func (w *diagnosticWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	for _, b := range p {
		if b == '\n' {
			if !w.dropping {
				if code := diagnosticCode(w.pending); code != "" {
					w.emit(code)
				}
			}
			w.pending = w.pending[:0]
			w.dropping = false
		} else if !w.dropping {
			if len(w.pending) >= 4096 {
				w.pending = w.pending[:0]
				w.dropping = true
			} else {
				w.pending = append(w.pending, b)
			}
		}
	}
	return n, nil
}
func diagnosticCode(line []byte) string {
	for _, candidate := range []struct{ text, code string }{
		{"Unable to load client plugin", "ENGINE_PLUGIN_LOAD_FAILED"},
		{"winpr_int_assert", "ENGINE_ASSERTION_FAILED"},
		{"freerdp_pre_connect failed", "ENGINE_PRECONNECT_FAILED"},
		{"freerdp_post_connect failed", "ENGINE_POSTCONNECT_FAILED"},
		{"certificate store initialization failed", "CERTIFICATE_STORE_INIT_FAILED"},
		{"home directory for storage", "INHERITED_HOME_UNWRITABLE"},
		{"Certificate validation failed", "CERTIFICATE_REJECTED"},
		{"Certificate verification failure", "CERTIFICATE_VERIFY_FAILED"},
		{"OpenSSL LEGACY provider failed to load", "CRYPTO_PROVIDER_MISSING"},
		{"SSL_CERT_NOT_ON_SERVER", "SERVER_CERTIFICATE_MISSING"},
		{"HYBRID_REQUIRED_BY_SERVER", "NLA_REQUIRED_BY_SERVER"},
		{"SSL_REQUIRED_BY_SERVER", "TLS_REQUIRED_BY_SERVER"},
		{"SSL_NOT_ALLOWED_BY_SERVER", "TLS_NOT_ALLOWED_BY_SERVER"},
		{"ERRCONNECT_SECURITY_NEGO_CONNECT_FAILED", "SECURITY_NEGOTIATION_FAILED"},
		{"ERRCONNECT_TLS_CONNECT_FAILED", "TLS_CONNECT_FAILED"},
		{"ERRCONNECT_LOGON_FAILURE", "LOGON_FAILURE"},
		{"ERRCONNECT_AUTHENTICATION_FAILED", "AUTHENTICATION_FAILED"},
		{"ERRCONNECT_CONNECT_TRANSPORT_FAILED", "TRANSPORT_FAILED"},
	} {
		if bytes.Contains(line, []byte(candidate.text)) {
			return candidate.code
		}
	}
	return ""
}
