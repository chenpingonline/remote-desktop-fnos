package engine

import (
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestPortConflictAndOwnedStop(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := l.Addr().String()
	script := filepath.Join(t.TempDir(), "fake-guacd")
	os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 60\n"), 0700)
	if _, e = Start(script, addr, t.TempDir(), func() {}); e == nil {
		t.Fatal("occupied port accepted")
	}
	l.Close()
	var calls atomic.Int32
	s, e := Start(script, addr, t.TempDir(), func() { calls.Add(1) })
	if e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	s.Stop()
	s.Stop()
	if time.Since(start) > 2*time.Second || calls.Load() != 1 {
		t.Fatal("stop not idempotent", calls.Load())
	}
}
