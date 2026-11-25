// Package engine owns exactly one foreground guacd process group.
package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"remote-desktop/internal/guac"
	"sync"
	"syscall"
	"time"
)

type Supervisor struct {
	mu            sync.Mutex
	cmd           *exec.Cmd
	cancel        context.CancelFunc
	done          chan struct{}
	exit          func()
	path, address string
	runtimeEnv    []string
	stopping      bool
}

func Start(path, address, runtimeDir string, onExit func()) (*Supervisor, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("engine path must be absolute")
	}
	runtimeEnv, err := privateRuntime(runtimeDir)
	if err != nil {
		return nil, err
	}
	s := &Supervisor{path: path, address: address, runtimeEnv: runtimeEnv, exit: onExit, done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	if e := s.launch(ctx); e != nil {
		cancel()
		return nil, e
	}
	go s.watch(ctx)
	return s, nil
}
func (s *Supervisor) launch(ctx context.Context) error {
	args := []string{"-f", "-L", "debug"}
	if filepath.IsAbs(s.address) {
		if err := clearStaleSocket(s.address); err != nil {
			return err
		}
		args = append(args, "-b", s.address)
	} else {
		host, port, err := net.SplitHostPort(s.address)
		if err != nil {
			return err
		}
		probe, err := net.Listen("tcp", s.address)
		if err != nil {
			return fmt.Errorf("private engine port occupied: %w", err)
		}
		probe.Close()
		args = append(args, "-b", host, "-l", port)
	}
	cmd := exec.Command(s.path, args...)
	cmd.SysProcAttr = processAttrs()
	// A forked worker must not hold Wait open through inherited log pipes.
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(append(os.Environ(), s.runtimeEnv...), "LD_LIBRARY_PATH="+filepath.Join(filepath.Dir(s.path), "../lib"), "SSL_CERT_DIR="+filepath.Join(filepath.Dir(s.path), "../ca-certificates"), "OPENSSL_MODULES="+filepath.Join(filepath.Dir(s.path), "../lib/ossl-modules"))
	// FreeRDP forwards warnings/errors through guacd at debug level. Keep
	// that transport enabled, but suppress FreeRDP informational/debug events.
	cmd.Env = append(cmd.Env, "WLOG_LEVEL=WARN")
	diagnostics := &diagnosticWriter{emit: func(code string) { log.Printf("remote-desktop engine_diagnostic=%s", code) }}
	cmd.Stdout = diagnostics
	cmd.Stderr = diagnostics
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil || s.stopping {
		return fmt.Errorf("engine stopping")
	}
	if e := cmd.Start(); e != nil {
		return e
	}
	s.cmd = cmd
	return nil
}

// The app holds the data lock and the parent directory is private to its UID.
func clearStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("engine socket path is occupied: %s", path)
	}
	conn, err := net.DialTimeout("unix", path, 250*time.Millisecond)
	if err == nil {
		conn.Close()
		return fmt.Errorf("engine socket is already listening: %s", path)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return err
	}
	return os.Remove(path)
}

func (s *Supervisor) watch(ctx context.Context) {
	defer close(s.done)
	for attempt := 0; ; attempt++ {
		s.mu.Lock()
		cmd := s.cmd
		s.mu.Unlock()
		cmd.Wait()
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		s.exit()
		if ctx.Err() != nil || attempt >= 3 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
		if e := s.launch(ctx); e != nil {
			return
		}
	}
}
func (s *Supervisor) Stop() {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		<-s.done
		return
	}
	s.stopping = true
	s.cancel()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	select {
	case <-s.done:
	case <-time.After(8 * time.Second):
		if cmd != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-s.done
	}
}
func (s *Supervisor) WaitReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return fmt.Errorf("engine stopped")
		case <-ticker.C:
			probeCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			c, e := guac.DialContext(probeCtx, s.address)
			cancel()
			if e == nil {
				c.Close()
				return nil
			}

		}
	}
}
