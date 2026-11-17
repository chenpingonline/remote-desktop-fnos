package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

func privateRuntime(root string) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("private engine runtime directory is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if inherited, e := os.UserHomeDir(); e != nil || syscall.Access(inherited, 2) != nil {
		log.Print("remote-desktop engine_diagnostic=INHERITED_HOME_UNWRITABLE")
	}
	var env []string
	for _, entry := range []struct{ name, child string }{{"", ""}, {"HOME", "home"}, {"XDG_CONFIG_HOME", "config"}, {"XDG_CACHE_HOME", "cache"}, {"XDG_DATA_HOME", "data"}} {
		dir := filepath.Join(absolute, entry.child)
		if err = os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("create private engine runtime: %w", err)
		}
		info, e := os.Lstat(dir)
		if e != nil || !info.IsDir() {
			return nil, fmt.Errorf("private engine runtime must be a directory")
		}
		if err = os.Chmod(dir, 0700); err != nil {
			return nil, err
		}
		if entry.name != "" {
			env = append(env, entry.name+"="+dir)
		}
	}
	return env, nil
}
