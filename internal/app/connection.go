package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type inlineCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
}
type connectionInput struct {
	Connection
	Credential *inlineCredential `json:"credential,omitempty"`
}

// Credential metadata, ciphertext and the connection are committed together.
func (a *App) saveConnectionWithCredential(c Connection, login inlineCredential, owner string, previousVersion int, create bool) (Connection, error) {
	cr := Credential{ID: randomID(), Name: c.Name, Type: c.Protocol, Version: 1, UpdatedAt: time.Now().UTC(), Username: login.Username, Password: login.Password, Domain: login.Domain}
	payload, e := json.Marshal(cr)
	if e != nil {
		return c, e
	}
	ciphertext, e := a.vault.Encrypt(payload, owner+":"+cr.ID+":"+cr.Type+":key1")
	if e != nil {
		return c, e
	}
	cr.Username = ""
	cr.Password = ""
	cr.Domain = ""
	metadata, e := json.Marshal(cr)
	if e != nil {
		return c, e
	}
	c.CredentialID = cr.ID
	data, e := json.Marshal(c)
	if e != nil {
		return c, e
	}
	tx, e := a.db.Begin()
	if e != nil {
		return c, e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT INTO objects VALUES('credentials',?,?,1,?)", cr.ID, owner, metadata); e != nil {
		return c, e
	}
	if _, e = tx.Exec("INSERT INTO objects VALUES('secrets',?,?,1,?)", cr.ID, owner, ciphertext); e != nil {
		return c, e
	}
	if create {
		_, e = tx.Exec("INSERT INTO objects VALUES('connections',?,?,1,?)", c.ID, owner, data)
	} else {
		var result sql.Result
		result, e = tx.Exec("UPDATE objects SET version=?,data=? WHERE kind='connections' AND id=? AND owner=? AND version=?", c.Version, data, c.ID, owner, previousVersion)
		if e == nil {
			var n int64
			n, e = result.RowsAffected()
			if e == nil && n != 1 {
				e = errors.New("VERSION_CONFLICT")
			}
		}
	}
	if e != nil {
		return c, e
	}
	if e = tx.Commit(); e != nil {
		return c, e
	}
	a.audit(owner, "SAVE_credentials", cr.ID)
	return c, nil
}
