package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (a *App) EngineExited() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopping {
		return
	}
	for _, s := range a.sessions {
		if !terminal(s.State) {
			a.closeLocked(s, "ENGINE_RESTART", true)
		}
	}
	a.audit("system", "ENGINE_EXIT", "")
}

// Backup creates a consistent SQLite snapshot and pairs it with the key.
// The destination is operator controlled and never exposed over HTTP.
func (a *App) Backup(destination string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if e := os.Mkdir(destination, 0700); e != nil {
		return e
	}
	dbPath := filepath.Join(destination, "app.db")
	_, e := a.db.Exec("VACUUM INTO ?", dbPath)
	if e != nil {
		return e
	}
	os.Chmod(dbPath, 0600)
	key, e := os.ReadFile(a.cfg.KeyPath)
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(destination, "master.key"), key, 0600); e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(destination, "metadata.txt"), []byte(fmt.Sprintf("version=%s\nschema=1\ncreatedAt=%s\n", a.cfg.Version, time.Now().UTC().Format(time.RFC3339))), 0600)
}
