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

func TestClearStaleSocket(t *testing.T) {
	// Darwin limits socket paths to 103 bytes, including the test root.
	dir, err := os.MkdirTemp("/tmp", "rd-sock-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "guacd.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	listener.SetUnlinkOnClose(false)
	defer listener.Close()
	if err := clearStaleSocket(path); err == nil {
		t.Fatal("accepted a live listener")
	}
	listener.Close()
	if err := clearStaleSocket(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("stale socket remains: %v", err)
	}
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	link := path + ".link"
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	for _, occupied := range []string{path, link} {
		if err := clearStaleSocket(occupied); err == nil {
			t.Fatalf("accepted occupied path %s", occupied)
		}
		if _, err := os.Lstat(occupied); err != nil {
			t.Fatal(err)
		}
	}
}
