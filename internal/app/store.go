package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

func openDB(path string) (*sql.DB, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY,applied_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS objects(kind TEXT NOT NULL,id TEXT NOT NULL,owner TEXT NOT NULL,version INTEGER NOT NULL,data BLOB NOT NULL,PRIMARY KEY(kind,id));
 CREATE INDEX IF NOT EXISTS objects_owner ON objects(kind,owner);
 CREATE TABLE IF NOT EXISTS app_settings(key TEXT PRIMARY KEY,value BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS audit_events(id TEXT PRIMARY KEY,owner TEXT NOT NULL,action TEXT NOT NULL,object_id TEXT NOT NULL,created_at TEXT NOT NULL);
 INSERT OR IGNORE INTO schema_migrations VALUES(1,datetime('now'));`)
	if e != nil {
		db.Close()
		return nil, e
	}
	var version int
	if e = db.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); e != nil || version != 1 {
		db.Close()
		return nil, fmt.Errorf("unsupported database schema")
	}
	os.Chmod(path, 0600)
	return db, nil
}
func (a *App) get(kind, id, owner string, out any) error {
	var data []byte
	e := a.db.QueryRow("SELECT data FROM objects WHERE kind=? AND id=? AND owner=?", kind, id, owner).Scan(&data)
	if e != nil {
		return e
	}
	return json.Unmarshal(data, out)
}
func (a *App) list(kind, owner string, limit, offset int) ([]json.RawMessage, error) {
	rows, e := a.db.Query("SELECT data FROM objects WHERE kind=? AND owner=? ORDER BY rowid DESC LIMIT ? OFFSET ?", kind, owner, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []json.RawMessage{}
	for rows.Next() {
		var data []byte
		if e = rows.Scan(&data); e != nil {
			return nil, e
		}
		items = append(items, json.RawMessage(data))
	}
	return items, rows.Err()
}
func (a *App) save(kind, id, owner string, version int, obj any, create bool) error {
	data, e := json.Marshal(obj)
	if e != nil {
		return e
	}
	if create {
		_, e = a.db.Exec("INSERT INTO objects VALUES(?,?,?,?,?)", kind, id, owner, 1, data)
		return e
	}
	res, e := a.db.Exec("UPDATE objects SET version=?,data=? WHERE kind=? AND id=? AND owner=? AND version=?", version+1, data, kind, id, owner, version)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("VERSION_CONFLICT")
	}
	return e
}
func (a *App) audit(owner, action, id string) {
	a.db.Exec("INSERT INTO audit_events VALUES(?,?,?,?,?)", randomID(), owner, action, id, time.Now().UTC().Format(time.RFC3339Nano))
}
func (a *App) persist(s *Session) {
	a.save("sessions", s.ID, s.Owner, 0, s, true)
	data, _ := json.Marshal(s)
	a.db.Exec("UPDATE objects SET data=? WHERE kind='sessions' AND id=? AND owner=?", data, s.ID, s.Owner)
}
