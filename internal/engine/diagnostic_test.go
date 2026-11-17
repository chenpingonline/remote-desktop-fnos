package engine

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDiagnosticWriterSanitizesAndBounds(t *testing.T) {
	var codes []string
	w := &diagnosticWriter{emit: func(code string) { codes = append(codes, code) }}
	w.Write([]byte("secret username and password\nWARN Certificate veri"))
	w.Write([]byte("fication failure 'private certificate subject'\nINFO Certificate validation failed\n"))
	w.Write([]byte(strings.Repeat("x", 5000) + "ERRCONNECT_LOGON_FAILURE secret\n"))
	w.Write([]byte("ERRCONNECT_SECURITY_NEGO_CONNECT_FAILED secret\n"))
	if strings.Join(codes, ",") != "CERTIFICATE_VERIFY_FAILED,CERTIFICATE_REJECTED,SECURITY_NEGOTIATION_FAILED" {
		t.Fatalf("unexpected diagnostic categories: %v", codes)
	}
	if len(w.pending) != 0 {
		t.Fatal("pending raw data retained")
	}
}

func TestFreeRDP3FailureDiagnosticsRemainSanitized(t *testing.T) {
	for _, tc := range []struct{ raw, code string }{
		{"Unable to load client plugin private-path", "ENGINE_PLUGIN_LOAD_FAILED"},
		{"DEBUG: stack private-username winpr_int_assert private-path", "ENGINE_ASSERTION_FAILED"},
		{"freerdp_pre_connect failed: private-target", "ENGINE_PRECONNECT_FAILED"},
		{"freerdp_post_connect failed: private-target", "ENGINE_POSTCONNECT_FAILED"},
	} {
		var codes []string
		w := &diagnosticWriter{emit: func(code string) { codes = append(codes, code) }}
		w.Write([]byte(tc.raw + "\n"))
		if len(codes) != 1 || codes[0] != tc.code || len(w.pending) != 0 {
			t.Fatalf("incorrect or unsanitized FreeRDP 3 diagnostic: %v", codes)
		}
	}
}

func TestConnectionProgressDiagnosticsAreDropped(t *testing.T) {
	for _, line := range []string{
		"Creating new client for protocol rdp",
		"RDP_STAGE_CONTEXT_BEGIN", "RDP_STAGE_LOGIN_COMPLETE",
		"RDP_AUTH_NTLM_PASSWORD_ACCOUNT", "Using package: Negotiate",
		"Acquired client credentials", "Using direct NTLM",
		"Authentication in progress...", "Authentication complete (output token size: 1)",
		"NLA_STATE_FINAL", "CONNECTION_STATE_FINALIZATION",
	} {
		if code := diagnosticCode([]byte(line)); code != "" {
			t.Fatalf("connection progress leaked as %s", code)
		}
	}
}

// The subprocess boundary must preserve categories while dropping raw text.
func TestOwnedEngineDiagnosticPipe(t *testing.T) {
	t.Setenv("WLOG_LEVEL", "DEBUG")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	script := filepath.Join(t.TempDir(), "fake-guacd")
	if err = os.WriteFile(script, []byte("#!/bin/sh\n[ \"$WLOG_LEVEL\" = WARN ] || exit 1\nprintf 'Certificate validation failed private-password\\n' >&2\nexec sleep 60\n"), 0700); err != nil {
		t.Fatal(err)
	}
	sink := &diagnosticSink{ready: make(chan struct{}, 1)}
	previous := log.Writer()
	log.SetOutput(sink)
	defer log.SetOutput(previous)
	supervisor, err := Start(script, address, t.TempDir(), func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Stop()
	select {
	case <-sink.ready:
	case <-time.After(3 * time.Second):
		t.Fatal("diagnostic output did not cross process boundary")
	}
	supervisor.Stop()
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if !strings.Contains(sink.text, "engine_diagnostic=CERTIFICATE_REJECTED") || strings.Contains(sink.text, "private-password") {
		t.Fatal("diagnostic was missing or unsanitized")
	}
}

type diagnosticSink struct {
	mu    sync.Mutex
	text  string
	ready chan struct{}
}

func (s *diagnosticSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text += string(p)
	select {
	case s.ready <- struct{}{}:
	default:
	}
	return len(p), nil
}
