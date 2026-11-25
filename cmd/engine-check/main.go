// engine-check loads both real guacd plugins against a deliberately closed port.
// It proves local engine/plugin initialization only, never remote compatibility.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"remote-desktop/internal/engine"
	"remote-desktop/internal/guac"
	"time"
)

func main() {
	path := flag.String("engine", "/native/bin/guacd", "owned engine path")
	flag.Parse()
	runtimeDir, e := os.MkdirTemp("", "rdp-engine-check-")
	if e != nil {
		panic(e)
	}
	defer os.RemoveAll(runtimeDir)
	address := filepath.Join(runtimeDir, "guacd.sock")
	s, e := engine.Start(*path, address, runtimeDir, func() {})
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	defer s.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = s.WaitReady(ctx); e != nil {
		panic(e)
	}
	for _, protocol := range []string{"rdp", "vnc"} {
		c, e := guac.DialContext(ctx, address)
		if e != nil {
			panic(e)
		}
		c.SetDeadline(time.Now().Add(8 * time.Second))
		d, e := guac.Handshake(c, protocol, map[string]string{"hostname": "127.0.0.1", "port": "1", "username": "test", "password": "test", "security": "nla", "ignore-cert": "false", "disable-audio": "true"}, 800, 600, 96, "UTC")
		if e != nil {
			c.Close()
			panic(fmt.Sprintf("%s plugin handshake: %v", protocol, e))
		}
		rejected := false
		for i := 0; i < 100; i++ {
			p, err := d.Read()
			if err != nil {
				break
			}
			if len(p) > 0 && p[0] == "error" {
				rejected = true
				break
			}
		}
		c.Close()
		if !rejected {
			panic(fmt.Sprintf("%s did not reject closed target", protocol))
		}
		fmt.Printf("%s: plugin initialized; closed target rejected as expected\n", protocol)
	}
}
