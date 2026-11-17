package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func socketTestPath(t *testing.T, name string) string {
	t.Helper()
	dir, e := os.MkdirTemp("", "rd-health-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, name)
}
func TestLifecycleSocketChecks(t *testing.T) {
	path := socketTestPath(t, "app.sock")
	if got := probeSocket(path); got != 3 {
		t.Fatal("missing socket", got)
	}
	l, e := net.Listen("unix", path)
	if e != nil {
		t.Fatal(e)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/remote-desktop/api/v1/health" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	})}
	go srv.Serve(l)
	if got := probeSocket(path); got != 0 {
		t.Fatal("live socket", got)
	}
	if e = healthcheck(path); e != nil {
		t.Fatal(e)
	}
	srv.Close()
	if got := probeSocket(path); got != 3 {
		t.Fatal("stale socket", got)
	}
	if e = healthcheck(path); e == nil {
		t.Fatal("stale socket passed health")
	}
	os.Remove(path)
}
func TestHealthRejectsUnrelatedListener(t *testing.T) {
	path := socketTestPath(t, "other.sock")
	l, e := net.Listen("unix", path)
	if e != nil {
		t.Fatal(e)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"other-service"}`)) })}
	defer srv.Close()
	go srv.Serve(l)
	if probeSocket(path) != 0 {
		t.Fatal("listener not detected")
	}
	if healthcheck(path) == nil {
		t.Fatal("unrelated service accepted")
	}
}
