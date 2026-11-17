package app

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"remote-desktop/internal/rdpcert"
	"strings"
	"time"
)

const trustKind = "certificate-trust"

type certificateTrust struct {
	ConnectionID string       `json:"connectionId"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	Certificate  rdpcert.Info `json:"certificate"`
	Version      int          `json:"version"`
	TrustedAt    time.Time    `json:"trustedAt"`
}
type certificateCheck struct {
	Owner, Origin    string
	Connection       Connection
	Certificate      rdpcert.Info
	Previous         *certificateTrust
	Ready, Confirmed bool
	Expires          time.Time
}

func certificateRoots(c Config) *x509.CertPool {
	if c.CertificateRoots != nil {
		return c.CertificateRoots
	}
	roots, e := x509.SystemCertPool()
	if e != nil {
		roots = x509.NewCertPool()
	}
	if c.WebDir != "" {
		if pem, e := os.ReadFile(filepath.Join(filepath.Dir(c.WebDir), "ca-certificates", "ca-certificates.crt")); e == nil {
			roots.AppendCertsFromPEM(pem)
		}
	}
	return roots
}
func (a *App) inspectCertificate(ctx context.Context, c Connection) (rdpcert.Info, error) {
	if a.cfg.CertificateProbe != nil {
		return a.cfg.CertificateProbe(ctx, c)
	}
	return rdpcert.Probe(ctx, c.Host, c.Port, a.certificateRoots)
}
func sameTarget(a, b Connection) bool {
	return strings.EqualFold(a.Host, b.Host) && a.Port == b.Port && a.Version == b.Version && a.Protocol == b.Protocol
}
func trustTarget(t *certificateTrust, c Connection) bool {
	return t != nil && strings.EqualFold(t.Host, c.Host) && t.Port == c.Port
}
func (a *App) trust(c Connection, owner string) (*certificateTrust, error) {
	var t certificateTrust
	err := a.get(trustKind, c.ID, owner, &t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
func sameTrust(a, b *certificateTrust) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Version == b.Version && a.Certificate.Fingerprint == b.Certificate.Fingerprint && a.Host == b.Host && a.Port == b.Port
}
func checkState(check *certificateCheck) string {
	if !check.Certificate.CanTrust {
		return "invalid"
	}
	if check.Previous != nil {
		if trustTarget(check.Previous, check.Connection) && check.Previous.Certificate.Fingerprint == check.Certificate.Fingerprint {
			return "trusted"
		}
		return "changed"
	}
	if check.Certificate.Verified {
		return "valid"
	}
	return "untrusted"
}
func (a *App) expireCertificateChecks() {
	now := time.Now()
	for key, check := range a.certificateChecks {
		if now.After(check.Expires) {
			delete(a.certificateChecks, key)
		}
	}
}
func (a *App) clearCertificateChecks(owner, id string) {
	for key, check := range a.certificateChecks {
		if check.Owner == owner && check.Connection.ID == id {
			delete(a.certificateChecks, key)
		}
	}
}

// Called under a.mu, with the same identity/CSRF/origin protection as other writes.
func (a *App) certificateAPI(w http.ResponseWriter, r *http.Request, u User, id, action string) {
	var c Connection
	if id == "" || a.get("connections", id, u.UID, &c) != nil {
		fail(w, 404, "NOT_FOUND", "连接不存在")
		return
	}

	if action == "trust" && r.Method == "GET" {
		stored, err := a.trust(c, u.UID)
		if err != nil {
			fail(w, 500, "STORE_ERROR", "读取证书信任失败")
			return
		}
		respond(w, 200, map[string]any{"trust": stored})
		return
	}
	if action == "trust" && r.Method == "DELETE" {
		if _, e := a.db.Exec("DELETE FROM objects WHERE kind=? AND id=? AND owner=?", trustKind, c.ID, u.UID); e != nil {
			fail(w, 500, "STORE_ERROR", "撤销信任失败")
			return
		}
		a.clearCertificateChecks(u.UID, c.ID)
		a.invalidate(u.UID, c.ID, true)
		a.audit(u.UID, "CERTIFICATE_TRUST_REVOKED", c.ID)
		respond(w, 200, map[string]bool{"ok": true})
		return
	}
	if c.Protocol != "rdp" {
		fail(w, 400, "INVALID_PROTOCOL", "仅 RDP 连接使用证书信任")
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "METHOD_NOT_ALLOWED", "不支持该操作")
		return
	}
	a.expireCertificateChecks()
	origin, _ := browserOrigin(r)
	if action == "certificate" {
		if time.Since(a.certificateProbes[u.UID]) < time.Second {
			fail(w, 429, "RATE_LIMIT", "请稍候再检查证书")
			return
		}
		count := 0
		for _, check := range a.certificateChecks {
			if check.Owner == u.UID {
				count++
			}
		}
		if count >= 16 {
			fail(w, 429, "RATE_LIMIT", "待确认的证书过多，请稍后重试")
			return
		}
		a.certificateProbes[u.UID] = time.Now()
		a.mu.Unlock()
		info, e := a.inspectCertificate(r.Context(), c)
		a.mu.Lock()
		if e != nil {
			fail(w, 422, "CERTIFICATE_PROBE_FAILED", e.Error())
			return
		}
		var current Connection
		if a.stopping || a.get("connections", id, u.UID, &current) != nil || !sameTarget(c, current) {
			fail(w, 409, "CONNECTION_CHANGED", "连接已改变，请重新检查证书")
			return
		}
		if len(info.Fingerprint) != 64 {
			fail(w, 502, "CERTIFICATE_PROBE_FAILED", "无法取得有效的证书指纹")
			return
		}
		previous, err := a.trust(c, u.UID)
		if err != nil {
			fail(w, 500, "STORE_ERROR", "读取证书信任失败")
			return
		}
		check := &certificateCheck{Owner: u.UID, Origin: origin, Connection: c, Certificate: info, Previous: previous, Expires: time.Now().Add(2 * time.Minute)}
		state := checkState(check)
		check.Ready = state == "valid" || state == "trusted"
		token := randomID()
		a.certificateChecks[sha256.Sum256([]byte(token))] = check
		respond(w, 200, map[string]any{"status": state, "certificate": info, "previous": check.Previous, "check": token, "expiresAt": check.Expires})
		return
	}
	if action == "trust" {
		var in struct {
			Check string `json:"check"`
		}
		if !decode(w, r, &in) {
			return
		}
		check := a.certificateChecks[sha256.Sum256([]byte(in.Check))]
		if check == nil || check.Owner != u.UID || check.Origin != origin || !sameTarget(check.Connection, c) || time.Now().After(check.Expires) || check.Confirmed {
			fail(w, 409, "CERTIFICATE_CHECK_EXPIRED", "证书确认信息已失效，请重新检查")
			return
		}
		if !check.Certificate.CanTrust || time.Now().Before(check.Certificate.NotBefore) || !time.Now().Before(check.Certificate.NotAfter) {
			fail(w, 409, "CERTIFICATE_INVALID", "证书已过期或尚未生效，请先更新远端证书")
			return
		}
		current, err := a.trust(c, u.UID)
		if err != nil {
			fail(w, 500, "STORE_ERROR", "读取证书信任失败")
			return
		}
		if !sameTrust(current, check.Previous) {
			fail(w, 409, "TRUST_CHANGED", "信任记录已改变，请重新检查证书")
			return
		}
		version := 1
		if current != nil {
			version = current.Version + 1
		}
		saved := &certificateTrust{ConnectionID: c.ID, Host: c.Host, Port: c.Port, Certificate: check.Certificate, Version: version, TrustedAt: time.Now().UTC()}
		payload, e := json.Marshal(saved)
		if e != nil {
			fail(w, 500, "STORE_ERROR", "保存信任失败")
			return
		}
		result, e := a.db.Exec("INSERT INTO objects(kind,id,owner,version,data) VALUES(?,?,?,?,?) ON CONFLICT(kind,id) DO UPDATE SET version=excluded.version,data=excluded.data WHERE owner=excluded.owner", trustKind, c.ID, u.UID, version, payload)
		if e != nil {
			fail(w, 500, "STORE_ERROR", "保存信任失败")
			return
		}
		if count, err := result.RowsAffected(); err != nil || count != 1 {
			fail(w, 500, "STORE_ERROR", "保存信任失败")
			return
		}
		// Confirmation invalidates other pending checks and sessions for this target.
		a.clearCertificateChecks(u.UID, c.ID)
		a.invalidate(u.UID, c.ID, true)
		check.Previous = saved
		check.Ready = true
		check.Confirmed = true
		a.certificateChecks[sha256.Sum256([]byte(in.Check))] = check
		a.audit(u.UID, "CERTIFICATE_TRUST_SAVED", c.ID)
		respond(w, 200, map[string]any{"trust": saved, "check": in.Check})
		return
	}
	http.NotFound(w, r)
}
func (a *App) consumeCertificateCheck(token string, c Connection, u User, r *http.Request) (string, error) {
	a.expireCertificateChecks()
	key := sha256.Sum256([]byte(token))
	check := a.certificateChecks[key]
	origin, _ := browserOrigin(r)
	if check == nil || check.Owner != u.UID || check.Origin != origin || !sameTarget(check.Connection, c) || !check.Ready || time.Now().After(check.Expires) {
		return "", fmt.Errorf("请重新检查并确认远端证书")
	}
	current, err := a.trust(c, u.UID)
	if err != nil {
		return "", fmt.Errorf("无法读取证书信任，请稍后重试")
	}
	if !sameTrust(current, check.Previous) {
		return "", fmt.Errorf("证书信任已改变，请重新检查")
	}
	if time.Now().Before(check.Certificate.NotBefore) || !time.Now().Before(check.Certificate.NotAfter) {
		return "", fmt.Errorf("远端证书已过期或尚未生效")
	}
	delete(a.certificateChecks, key)
	return check.Certificate.Fingerprint, nil
}

func (a *App) listCertificateTrust(w http.ResponseWriter, r *http.Request, u User) {
	limit, offset := page(r)
	rows, err := a.db.Query("SELECT t.data,c.data FROM objects t JOIN objects c ON c.kind='connections' AND c.id=t.id AND c.owner=t.owner WHERE t.kind=? AND t.owner=? ORDER BY t.rowid DESC LIMIT ? OFFSET ?", trustKind, u.UID, limit, offset)
	if err != nil {
		fail(w, 500, "STORE_ERROR", "读取证书信任失败")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var tb, cb []byte
		var trust certificateTrust
		var conn Connection
		if err = rows.Scan(&tb, &cb); err != nil {
			break
		}
		if err = json.Unmarshal(tb, &trust); err != nil {
			break
		}
		if err = json.Unmarshal(cb, &conn); err != nil {
			break
		}
		items = append(items, map[string]any{"connectionId": conn.ID, "name": conn.Name, "host": trust.Host, "port": trust.Port, "certificate": trust.Certificate, "trustedAt": trust.TrustedAt})
	}
	if err != nil || rows.Err() != nil {
		fail(w, 500, "STORE_ERROR", "读取证书信任失败")
		return
	}
	respond(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}
