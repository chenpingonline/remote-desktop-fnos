package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

// healthcheck is a standalone CLI operation. It never opens the database,
// acquires the service lock, or starts an engine.
func healthcheck(path string) error {
	transport := &http.Transport{DisableKeepAlives: true, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", path)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get("http://localhost/app/remote-desktop/api/v1/health")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %d", response.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1024)).Decode(&body); err != nil {
		return err
	}
	if body.Status != "ok" {
		return fmt.Errorf("health endpoint is not ready")
	}
	return nil
}

// probeSocket distinguishes a confirmed stale socket from an inaccessible one.
// Only exit 3 permits lifecycle code to remove a stale socket path.
func probeSocket(path string) int {
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err == nil {
		conn.Close()
		return 0
	}
	if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
		return 3
	}
	return 1
}
