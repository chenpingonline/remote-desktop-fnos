package engine

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrivateRuntimeOverridesInheritedHome(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "missing-home"))
	root := filepath.Join(t.TempDir(), "engine")
	output := filepath.Join(t.TempDir(), "env.txt")
	t.Setenv("RD_TEST_OUTPUT", output)
	script := filepath.Join(t.TempDir(), "fake-guacd")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$HOME\" \"$XDG_CONFIG_HOME\" \"$XDG_CACHE_HOME\" \"$XDG_DATA_HOME\" > \"$RD_TEST_OUTPUT\"\nexec sleep 60\n"), 0700); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := l.Addr().String()
	l.Close()
	s, err := Start(script, address, root, func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	var data []byte
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		data, err = os.ReadFile(output)
		if err == nil && len(data) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Stop()
	expected := []string{}
	for _, child := range []string{"home", "config", "cache", "data"} {
		path := filepath.Join(root, child)
		expected = append(expected, path)
		info, e := os.Stat(path)
		if e != nil || info.Mode().Perm() != 0700 {
			t.Fatal("runtime directory is missing or not private", child, e)
		}
	}
	if string(data) != strings.Join(expected, "\n")+"\n" {
		t.Fatal("child did not receive dedicated runtime directories")
	}
	if _, err := os.Stat(os.Getenv("HOME")); !os.IsNotExist(err) {
		t.Fatal("inherited home was modified")
	}
}
func TestPrivateRuntimeRejectsSymlink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "engine")
	if err := os.Symlink(t.TempDir(), root); err != nil {
		t.Fatal(err)
	}
	if _, err := privateRuntime(root); err == nil {
		t.Fatal("symlink runtime accepted")
	}
}
