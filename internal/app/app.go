package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"remote-desktop/internal/rdpcert"
	"remote-desktop/internal/secret"
	"remote-desktop/internal/wol"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	DataDir, KeyPath, WebDir, Origin, DevToken, Guacd, Version string
	Gateway                                                    bool
	CertificateRoots                                           *x509.CertPool
	CertificateProbe                                           func(context.Context, Connection) (rdpcert.Info, error)
}
type App struct {
	origins           map[string]struct{}
	certificateRoots  *x509.CertPool
	certificateChecks map[[32]byte]*certificateCheck
	certificateProbes map[string]time.Time
	limits            Limits
	lockFile          *os.File
	mu                sync.Mutex
	db                *sql.DB
	vault             *secret.Vault
	cfg               Config
	csrfKey           []byte
	sessions          map[string]*Session
	ephemerals        map[[32]byte]ephemeral
	probes            map[string]time.Time
	wakeSender        func(context.Context, string, wol.Config) ([]string, error)
	wakes             map[string]time.Time
	stopping          bool
	done              chan struct{}
}

func randomID() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func New(c Config) (*App, error) {
	origins := map[string]struct{}{}
	if !c.Gateway {
		var valid bool
		origins, valid = configuredOrigins(c.Origin)
		if !valid {
			return nil, fmt.Errorf("invalid development origin")
		}
	}
	if e := os.MkdirAll(c.DataDir, 0700); e != nil {
		return nil, e
	}
	lock, e := os.OpenFile(filepath.Join(c.DataDir, "app.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		lock.Close()
		return nil, fmt.Errorf("application data already in use")
	}
	success := false
	defer func() {
		if !success {
			lock.Close()
		}
	}()
	if _, e := os.Stat(filepath.Join(c.DataDir, "app.db")); e == nil {
		if _, e = os.Stat(c.KeyPath); e != nil {
			return nil, fmt.Errorf("database exists but master key is missing; restore matching backup")
		}
	}
	v, e := secret.Open(c.KeyPath)
	if e != nil {
		return nil, e
	}
	db, e := openDB(filepath.Join(c.DataDir, "app.db"))
	if e != nil {
		return nil, e
	}
	a := &App{wakeSender: wol.Send, wakes: map[string]time.Time{}, origins: origins, lockFile: lock, db: db, vault: v, cfg: c, csrfKey: []byte(randomID()), sessions: map[string]*Session{}, ephemerals: map[[32]byte]ephemeral{}, probes: map[string]time.Time{}, done: make(chan struct{}), certificateRoots: certificateRoots(c), certificateChecks: map[[32]byte]*certificateCheck{}, certificateProbes: map[string]time.Time{}}
	rows, e := db.Query("SELECT id,owner,data FROM objects WHERE kind='sessions'")
	if e != nil {
		db.Close()
		return nil, e
	}
	type old struct {
		id, owner string
		s         Session
	}
	var stale []old
	for rows.Next() {
		var x old
		var b []byte
		if e = rows.Scan(&x.id, &x.owner, &b); e != nil {
			rows.Close()
			db.Close()
			return nil, e
		}
		if json.Unmarshal(b, &x.s) == nil && !terminal(x.s.State) {
			stale = append(stale, x)
		}
	}
	rows.Close()
	for _, x := range stale {
		x.s.Owner = x.owner
		x.s.State = "INTERRUPTED"
		x.s.Reason = "APP_RESTART"
		t := time.Now().UTC()
		x.s.EndedAt = &t
		a.persist(&x.s)
	}
	a.limits = Limits{2, 4, 30, 8, 1}
	var saved []byte
	if e = a.db.QueryRow("SELECT value FROM app_settings WHERE key='limits'").Scan(&saved); e == nil {
		if e = json.Unmarshal(saved, &a.limits); e != nil {
			a.db.Close()
			return nil, e
		}
	}
	success = true
	go a.sweep()
	return a, nil
}
func terminal(s string) bool { return s == "CLOSED" || s == "FAILED" || s == "INTERRUPTED" }
func (a *App) sweep() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-a.done:
			return
		case now := <-t.C:
			a.mu.Lock()
			if a.stopping {
				a.mu.Unlock()
				return
			}
			for k, v := range a.ephemerals {
				if now.After(v.Expires) {
					delete(a.ephemerals, k)
				}
			}
			for _, s := range a.sessions {
				if s.State == "CREATED" && now.After(s.ExpiresAt) {
					a.closeLocked(s, "TICKET_EXPIRED", false)
				} else if s.State == "ACTIVE" && now.Sub(s.LastInput) > time.Duration(a.limits.IdleMinutes)*time.Minute {
					a.closeLocked(s, "IDLE_TIMEOUT", false)
				} else if !terminal(s.State) && now.Sub(s.CreatedAt) > time.Duration(a.limits.MaxHours)*time.Hour {
					a.closeLocked(s, "MAX_DURATION", false)
				}
				if terminal(s.State) && s.EndedAt != nil && now.Sub(*s.EndedAt) > time.Minute {
					delete(a.sessions, s.ID)
				}
			}
			a.db.Exec("DELETE FROM audit_events WHERE created_at < ? OR id IN (SELECT id FROM audit_events ORDER BY created_at DESC LIMIT -1 OFFSET 10000)", now.AddDate(0, 0, -90).UTC().Format(time.RFC3339Nano))
			a.db.Exec("DELETE FROM objects WHERE kind='sessions' AND json_extract(data,'$.state') IN ('CLOSED','FAILED','INTERRUPTED') AND json_extract(data,'$.createdAt') < ?", now.AddDate(0, 0, -90).UTC().Format(time.RFC3339Nano))
			for k, v := range a.probes {
				if now.Sub(v) > time.Minute {
					delete(a.probes, k)
				}
			}
			a.mu.Unlock()
		}
	}
}
func (a *App) Close() {
	a.mu.Lock()
	if a.stopping {
		a.mu.Unlock()
		return
	}
	a.stopping = true
	close(a.done)
	for _, s := range a.sessions {
		a.closeLocked(s, "APP_STOP", false)
	}
	a.ephemerals = map[[32]byte]ephemeral{}
	a.mu.Unlock()
	a.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	a.db.Close()
	a.lockFile.Close()
}
func (a *App) closeLocked(s *Session, reason string, failed bool) {
	if terminal(s.State) {
		return
	}
	s.State = "CLOSING"
	if s.Cancel != nil {
		s.Cancel()
	}
	s.Ticket = [32]byte{}
	if s.Timer != nil {
		s.Timer.Stop()
	}
	if s.Upstream != nil {
		s.Upstream.Close()
	}
	if s.WS != nil {
		s.WS.CloseNow()
	}
	s.Login = Credential{}
	s.State = "CLOSED"
	if failed {
		s.State = "FAILED"
	}
	s.Reason = reason
	t := time.Now().UTC()
	s.EndedAt = &t
	a.persist(s)
	a.audit(s.Owner, "SESSION_"+reason, s.ID)
}
func (a *App) csrf(uid string) string {
	h := hmac.New(sha256.New, a.csrfKey)
	h.Write([]byte(uid))
	return hex.EncodeToString(h.Sum(nil))
}
func (a *App) csrfForOrigin(uid, origin string) string {
	h := hmac.New(sha256.New, a.csrfKey)
	h.Write([]byte(uid))
	h.Write([]byte{0})
	h.Write([]byte(origin))
	return hex.EncodeToString(h.Sum(nil))
}
func (a *App) identity(r *http.Request) (User, bool) {
	if a.cfg.Gateway {
		uid := r.Header.Get("X-Trim-Userid")
		return User{UID: uid, Name: r.Header.Get("X-Trim-Username"), Admin: r.Header.Get("X-Trim-Isadmin") == "true"}, uid != ""
	}
	c, e := r.Cookie("rd_dev")
	if e != nil || !hmac.Equal([]byte(c.Value), []byte(a.csrf("dev-login:"+a.cfg.DevToken))) {
		return User{}, false
	}
	return User{UID: "local-developer", Name: "本地开发用户", Admin: true}, true
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, msg string) {
	respond(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg, "requestId": w.Header().Get("X-Request-ID")}})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, "INVALID_REQUEST", "请求字段不合法")
		return false
	}
	if e := d.Decode(new(any)); e != io.EOF {
		fail(w, 400, "INVALID_REQUEST", "请求只能包含一个 JSON 对象")
		return false
	}
	return true
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Request-ID", randomID()[:16])
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'")
	p := strings.TrimPrefix(r.URL.Path, Prefix+"/api/v1")
	if !strings.HasPrefix(r.URL.Path, Prefix+"/api/v1/") {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		if r.URL.Path == Prefix {
			http.Redirect(w, r, Prefix+"/", 302)
			return
		}
		if !strings.HasPrefix(r.URL.Path, Prefix+"/") {
			http.NotFound(w, r)
			return
		}
		http.StripPrefix(Prefix+"/", http.FileServer(http.Dir(a.cfg.WebDir))).ServeHTTP(w, r)
		return
	}
	if p == "/health" && r.Method == "GET" {
		respond(w, 200, map[string]string{"status": "ok"})
		return
	}
	if p == "/dev-login" && !a.cfg.Gateway && r.Method == "POST" {
		if !a.originAllowed(r) {
			fail(w, 403, "FORBIDDEN", "来源无效")
			return
		}
		var in struct {
			Token string `json:"token"`
		}
		if !decode(w, r, &in) {
			return
		}
		if a.cfg.DevToken == "" || !hmac.Equal([]byte(in.Token), []byte(a.cfg.DevToken)) {
			fail(w, 401, "AUTH_REQUIRED", "开发访问令牌不正确")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "rd_dev", Value: a.csrf("dev-login:" + a.cfg.DevToken), Path: Prefix, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: strings.HasPrefix(a.cfg.Origin, "https:"), MaxAge: 28800})
		respond(w, 200, map[string]bool{"ok": true})
		return
	}
	u, ok := a.identity(r)
	if !ok {
		message := "飞牛登录状态已失效，请登录飞牛后重新打开应用"
		if !a.cfg.Gateway {
			message = "请输入本地开发访问令牌"
		}
		respond(w, 401, map[string]any{"error": map[string]any{
			"code": "AUTH_REQUIRED", "message": message,
			"devLoginAllowed": !a.cfg.Gateway, "requestId": w.Header().Get("X-Request-ID"),
		}})
		return
	}
	if p == "/me" && r.Method == "GET" {
		origin, valid := a.bootstrapOrigin(r)
		if !valid {
			fail(w, 403, "ORIGIN_REJECTED", "无法初始化页面来源，请从飞牛应用入口重新打开")
			return
		}
		u.CSRF = a.csrfForOrigin(u.UID, origin)
		respond(w, 200, u)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if !a.originAllowed(r) {
			fail(w, 403, "ORIGIN_REJECTED", "请求来源无效，请从飞牛应用入口重新打开")
			return
		}
		origin, _ := browserOrigin(r)
		if !hmac.Equal([]byte(r.Header.Get("X-CSRF-Token")), []byte(a.csrfForOrigin(u.UID, origin))) {
			fail(w, 403, "CSRF_EXPIRED", "页面安全令牌已失效，请刷新后重试")
			return
		}
	}
	if p == "/tunnel" {
		if !a.originAllowed(r) {
			fail(w, 403, "FORBIDDEN", "来源无效")
			return
		}
		a.tunnel(w, r, u)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopping {
		fail(w, 503, "APP_STOPPING", "应用正在停止")
		return
	}
	a.route(w, r, u, p)
}
func page(r *http.Request) (int, int) {
	l, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	o, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if l < 1 || l > 100 {
		l = 50
	}
	if o < 0 {
		o = 0
	}
	return l, o
}
func (a *App) route(w http.ResponseWriter, r *http.Request, u User, p string) {
	method := r.Method
	if p == "/certificate-trust" && method == "GET" {
		a.listCertificateTrust(w, r, u)
		return
	}
	if p == "/settings" {
		a.settingsAPI(w, r, u)
		return
	}

	if p == "/capabilities" && method == "GET" {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		c, e := (&net.Dialer{}).DialContext(ctx, "tcp", a.cfg.Guacd)
		if e == nil {
			c.Close()
		}
		respond(w, 200, map[string]any{"version": a.cfg.Version, "engineReachable": e == nil, "engineVerified": false, "rdp": "待实机验证", "vnc": "待实机验证（VNC 密码模式）", "clipboard": "显式文本面板；兼容性待验证", "fileTransfer": false, "audio": false, "perUserLimit": a.limits.PerUser, "totalLimit": a.limits.Total, "idleTimeoutMinutes": a.limits.IdleMinutes, "maxDurationHours": a.limits.MaxHours})
		return
	}
	if p == "/wake-history" && method == "GET" {
		a.wakeHistoryAPI(w, r, u)
		return
	}
	if p == "/audit-events" && method == "GET" {
		l, o := page(r)
		rows, e := a.db.Query("SELECT id,action,object_id,created_at FROM audit_events WHERE owner=? ORDER BY created_at DESC LIMIT ? OFFSET ?", u.UID, l, o)
		if e != nil {
			fail(w, 500, "STORE_ERROR", "读取操作记录失败")
			return
		}
		defer rows.Close()
		items := []map[string]string{}
		for rows.Next() {
			var id, action, obj, at string
			rows.Scan(&id, &action, &obj, &at)
			items = append(items, map[string]string{"id": id, "action": action, "objectId": obj, "createdAt": at})
		}
		respond(w, 200, map[string]any{"items": items})
		return
	}
	seg := strings.Split(strings.Trim(p, "/"), "/")
	if len(seg) == 0 {
		http.NotFound(w, r)
		return
	}
	kind := seg[0]
	id := ""
	if len(seg) > 1 {
		id = seg[1]
	}
	if kind == "connections" && len(seg) == 3 && (seg[2] == "certificate" || seg[2] == "trust") {
		a.certificateAPI(w, r, u, id, seg[2])
		return
	}
	if kind == "sessions" {
		a.sessionAPI(w, r, u, id, seg)
		return
	}
	if kind == "ephemeral-credentials" && method == "POST" {
		var in struct {
			ConnectionID string `json:"connectionId"`
			Username     string `json:"username"`
			Password     string `json:"password"`
			Domain       string `json:"domain"`
		}
		if !decode(w, r, &in) {
			return
		}
		var c Connection
		if a.get("connections", in.ConnectionID, u.UID, &c) != nil {
			fail(w, 404, "NOT_FOUND", "连接不存在")
			return
		}
		count := 0
		for _, v := range a.ephemerals {
			if v.Owner == u.UID {
				count++
			}
		}
		if count >= 8 {
			fail(w, 429, "LIMIT_EXCEEDED", "临时凭据过多，请稍后重试")
			return
		}
		ref := randomID()
		expires := time.Now().UTC().Add(60 * time.Second)
		a.ephemerals[sha256.Sum256([]byte(ref))] = ephemeral{u.UID, c.ID, Credential{Username: in.Username, Password: in.Password, Domain: in.Domain}, expires}
		respond(w, 201, map[string]any{"ref": ref, "expiresAt": expires})
		return
	}
	if kind != "connections" && kind != "credentials" && kind != "groups" {
		http.NotFound(w, r)
		return
	}
	if kind == "connections" && len(seg) == 3 && seg[2] == "wake" {
		a.wakeAPI(w, r, u, id)
		return
	}
	if kind == "connections" && len(seg) == 3 && seg[2] == "probe" && method == "POST" {
		a.probeAPI(w, r, u, id)
		return
	}
	if len(seg) > 2 {
		http.NotFound(w, r)
		return
	}
	if method == "GET" {
		if id == "" {
			l, o := page(r)
			items, e := a.list(kind, u.UID, l, o)
			if e != nil {
				fail(w, 500, "STORE_ERROR", "读取失败")
				return
			}
			if kind == "credentials" {
				for i, item := range items {
					var meta Credential
					if json.Unmarshal(item, &meta) != nil {
						fail(w, 500, "STORE_ERROR", "读取凭据失败")
						return
					}
					login, err := a.login(Connection{CredentialID: meta.ID}, u.UID)
					if err != nil {
						fail(w, 500, "SECRET_ERROR", "读取凭据失败")
						return
					}
					meta.Username, meta.Domain = login.Username, login.Domain
					meta.Password = ""
					items[i], _ = json.Marshal(meta)
				}
			}
			respond(w, 200, map[string]any{"items": items, "limit": l, "offset": o})
			return
		}
		if kind == "credentials" {
			fail(w, 405, "METHOD_NOT_ALLOWED", "仅允许列出凭据元数据")
			return
		}
		var obj json.RawMessage
		if a.get(kind, id, u.UID, &obj) != nil {
			fail(w, 404, "NOT_FOUND", "资源不存在")
			return
		}
		respond(w, 200, obj)
		return
	}
	if method == "POST" && id == "" || method == "PATCH" && id != "" {
		a.writeObject(w, r, u, kind, id)
		return
	}
	if method == "DELETE" && id != "" {
		var obj json.RawMessage
		if a.get(kind, id, u.UID, &obj) != nil {
			fail(w, 404, "NOT_FOUND", "资源不存在")
			return
		}
		if kind == "credentials" {
			items, _ := a.list("connections", u.UID, 10000, 0)
			for _, b := range items {
				var c Connection
				json.Unmarshal(b, &c)
				if c.CredentialID == id {
					fail(w, 409, "CREDENTIAL_IN_USE", "请先解除连接对该凭据的引用")
					return
				}
			}
		}
		if kind == "groups" {
			items, _ := a.list("connections", u.UID, 10000, 0)
			for _, b := range items {
				var c Connection
				json.Unmarshal(b, &c)
				if c.GroupID == id {
					c.GroupID = ""
					v := c.Version
					c.Version++
					if e := a.save("connections", c.ID, u.UID, v, c, false); e != nil {
						fail(w, 500, "STORE_ERROR", "移动连接失败")
						return
					}
					a.invalidate(u.UID, c.ID, false)
				}
			}
		}
		if kind == "connections" {
			a.invalidate(u.UID, id, true)
			if _, e := a.db.Exec("DELETE FROM objects WHERE kind=? AND id=? AND owner=?", trustKind, id, u.UID); e != nil {
				fail(w, 500, "STORE_ERROR", "删除信任记录失败")
				return
			}
		}
		_, e := a.db.Exec("DELETE FROM objects WHERE kind=? AND id=? AND owner=?", kind, id, u.UID)
		if e != nil {
			fail(w, 500, "STORE_ERROR", "删除失败")
			return
		}
		if kind == "credentials" {
			a.db.Exec("DELETE FROM objects WHERE kind='secrets' AND id=? AND owner=?", id, u.UID)
		}
		a.audit(u.UID, "DELETE_"+kind, id)
		respond(w, 200, map[string]bool{"ok": true})
		return
	}
	fail(w, 405, "METHOD_NOT_ALLOWED", "不支持该操作")
}
func (a *App) invalidate(owner, id string, active bool) {
	a.clearCertificateChecks(owner, id)
	for _, s := range a.sessions {
		if s.Owner == owner && s.ConnectionID == id && !terminal(s.State) && (active || s.State == "CREATED") {
			a.closeLocked(s, "PERMISSION_REVOKED", false)
		}
	}
	for k, e := range a.ephemerals {
		if e.Owner == owner && e.ConnectionID == id {
			delete(a.ephemerals, k)
		}
	}
}
func (a *App) writeObject(w http.ResponseWriter, r *http.Request, u User, kind, id string) {
	create := id == ""
	if create {
		id = randomID()
	}
	var version int
	var obj any
	switch kind {
	case "connections":
		var input connectionInput
		if !decode(w, r, &input) {
			return
		}
		c := input.Connection
		if input.Credential != nil {
			login := input.Credential
			if c.CredentialID != "" || login.Password == "" || len(login.Password) > 16384 || len(login.Username) > 512 || len(login.Domain) > 512 || (c.Protocol == "rdp" && strings.TrimSpace(login.Username) == "") || (c.Protocol == "vnc" && (login.Username != "" || login.Domain != "")) {
				fail(w, 400, "INVALID_CREDENTIAL", "请填写有效的登录凭据；VNC 模式仅填写密码")
				return
			}
		}
		if !create {
			var old Connection
			if a.get(kind, id, u.UID, &old) != nil {
				fail(w, 404, "NOT_FOUND", "连接不存在")
				return
			}
		}
		if err := c.WOL.Validate(); err != nil {
			fail(w, 400, "INVALID_WOL", err.Error())
			return
		}
		version = c.Version
		c.ID = id
		c.Name = strings.TrimSpace(c.Name)
		// VNC is platform-independent. Retain legacy target values on input,
		// but never persist them as an operating-system detection result.
		if c.Protocol == "vnc" && (c.TargetOS == "" || c.TargetOS == "auto" || c.TargetOS == "windows" || c.TargetOS == "macos" || c.TargetOS == "linux") {
			c.TargetOS = "auto"
		}
		if c.Protocol == "rdp" && c.TargetOS == "" {
			c.TargetOS = "windows"
		}
		if c.Name == "" || len(c.Name) > 120 || len(c.Host) > 253 || strings.TrimSpace(c.Host) != c.Host || strings.ContainsAny(c.Host, "/\\ :@\t\r\n") && net.ParseIP(c.Host) == nil || c.Host == "" || c.Port < 1 || c.Port > 65535 || !(c.Protocol == "rdp" && c.TargetOS == "windows" || c.Protocol == "vnc" && c.TargetOS == "auto") {
			fail(w, 400, "INVALID_CONNECTION", "请检查名称、目标地址、端口和协议")
			return
		}
		if c.Settings.Security == "" {
			c.Settings.Security = "nla"
		}
		if c.Settings.CertificatePolicy == "" {
			c.Settings.CertificatePolicy = "verify"
		}
		if c.Settings.Clipboard == "" {
			c.Settings.Clipboard = "disabled"
		}
		if c.Settings.AuthMode == "" {
			c.Settings.AuthMode = "password"
		}
		if c.Settings.Security != "nla" || c.Settings.CertificatePolicy != "verify" || (c.Settings.Clipboard != "text" && c.Settings.Clipboard != "disabled") || c.Settings.AuthMode != "password" {
			fail(w, 400, "INVALID_SETTINGS", "仅开放 NLA、正常证书校验和 VNC 密码模式")
			return
		}
		for k, ref := range map[string]string{"groups": c.GroupID, "credentials": c.CredentialID} {
			if ref != "" {
				var b json.RawMessage
				if a.get(k, ref, u.UID, &b) != nil {
					fail(w, 400, "INVALID_REFERENCE", "分组或凭据不存在")
					return
				}
				if k == "credentials" {
					var cr Credential
					json.Unmarshal(b, &cr)
					if cr.Type != c.Protocol {
						fail(w, 400, "INVALID_REFERENCE", "凭据协议与连接不匹配")
						return
					}
				}
			}
		}
		c.Version = version + 1
		if create {
			c.Version = 1
		}
		if input.Credential != nil {
			saved, e := a.saveConnectionWithCredential(c, *input.Credential, u.UID, version, create)
			if e != nil {
				fail(w, 409, "SAVE_FAILED", "连接及凭据未能保存，请刷新后重试")
				return
			}
			a.invalidate(u.UID, id, false)
			a.audit(u.UID, "SAVE_connections", id)
			respond(w, 200, saved)
			return
		}
		obj = c
	case "groups":
		var g Group
		if !decode(w, r, &g) {
			return
		}
		version = g.Version
		if strings.TrimSpace(g.Name) == "" || len(g.Name) > 120 {
			fail(w, 400, "INVALID_NAME", "请填写有效分组名称")
			return
		}
		g.ID = id
		g.Version = version + 1
		if create {
			g.Version = 1
		}
		obj = g
	case "credentials":
		var c Credential
		if !decode(w, r, &c) {
			return
		}
		version = c.Version
		if c.Name == "" || len(c.Name) > 120 || (create && c.Password == "") || (c.Type != "rdp" && c.Type != "vnc") {
			fail(w, 400, "INVALID_CREDENTIAL", "请填写名称、协议和密码")
			return
		}
		if !create {
			var old Credential
			if a.get(kind, id, u.UID, &old) != nil {
				fail(w, 404, "NOT_FOUND", "凭据不存在")
				return
			}
			if old.Type != c.Type {
				fail(w, 400, "INVALID_CREDENTIAL", "已有凭据不能改变协议类型")
				return
			}
			if c.Password == "" {
				login, err := a.login(Connection{CredentialID: id}, u.UID)
				if err != nil {
					fail(w, 500, "SECRET_ERROR", "读取原密码失败")
					return
				}
				c.Password = login.Password
			}
		}
		if len(c.Username) > 512 || len(c.Domain) > 512 || len(c.Password) > 16384 || (c.Type == "rdp" && strings.TrimSpace(c.Username) == "") || (c.Type == "vnc" && (c.Username != "" || c.Domain != "")) {
			fail(w, 400, "INVALID_CREDENTIAL", "请填写有效的登录凭据；VNC 模式仅填写密码")
			return
		}
		c.ID = id
		c.Version = version + 1
		if create {
			c.Version = 1
		}
		c.UpdatedAt = time.Now().UTC()
		payload, _ := json.Marshal(c)
		encrypted, e := a.vault.Encrypt(payload, u.UID+":"+id+":"+c.Type+":key1")
		if e != nil {
			fail(w, 500, "SECRET_ERROR", "凭据加密失败")
			return
		}
		c.Username = ""
		c.Password = ""
		c.Domain = ""
		obj = c
		// Commit ciphertext and metadata in one transaction.
		tx, e := a.db.Begin()
		if e != nil {
			fail(w, 500, "STORE_ERROR", "保存失败")
			return
		}
		defer tx.Rollback()
		data, _ := json.Marshal(c)
		if create {
			_, e = tx.Exec("INSERT INTO objects VALUES(?,?,?,?,?)", kind, id, u.UID, 1, data)
		} else {
			var res sql.Result
			res, e = tx.Exec("UPDATE objects SET version=?,data=? WHERE kind=? AND id=? AND owner=? AND version=?", c.Version, data, kind, id, u.UID, version)
			if e == nil {
				n, _ := res.RowsAffected()
				if n == 0 {
					e = errors.New("VERSION_CONFLICT")
				}
			}
		}
		if e == nil {
			_, e = tx.Exec("INSERT INTO objects VALUES('secrets',?,?,1,?) ON CONFLICT(kind,id) DO UPDATE SET data=excluded.data", id, u.UID, encrypted)
		}
		if e == nil {
			e = tx.Commit()
		}
		if e != nil {
			fail(w, 409, "VERSION_CONFLICT", "保存失败，请刷新后重试")
			return
		}
		items, _ := a.list("connections", u.UID, 10000, 0)
		for _, b := range items {
			var con Connection
			json.Unmarshal(b, &con)
			if con.CredentialID == id {
				a.invalidate(u.UID, con.ID, false)
			}
		}
		a.audit(u.UID, "SAVE_"+kind, id)
		respond(w, 200, obj)
		return
	}
	if e := a.save(kind, id, u.UID, version, obj, create); e != nil {
		fail(w, 409, "VERSION_CONFLICT", "内容已改变，请刷新后重试")
		return
	}
	if kind == "connections" {
		a.invalidate(u.UID, id, false)
	}
	a.audit(u.UID, "SAVE_"+kind, id)
	respond(w, 200, obj)
}
func (a *App) login(c Connection, owner string) (Credential, error) {
	var meta Credential
	if e := a.get("credentials", c.CredentialID, owner, &meta); e != nil {
		return Credential{}, e
	}
	var data []byte
	if e := a.db.QueryRow("SELECT data FROM objects WHERE kind='secrets' AND id=? AND owner=?", c.CredentialID, owner).Scan(&data); e != nil {
		return Credential{}, e
	}
	plain, e := a.vault.Decrypt(data, owner+":"+c.CredentialID+":"+meta.Type+":key1")
	if e != nil {
		return Credential{}, e
	}
	var cr Credential
	e = json.Unmarshal(plain, &cr)
	return cr, e
}

var _ = os.ErrNotExist
