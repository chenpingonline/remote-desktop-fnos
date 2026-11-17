package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"remote-desktop/internal/guac"
	"strconv"
	"strings"
	"time"
)

func (a *App) sessionAPI(w http.ResponseWriter, r *http.Request, u User, id string, seg []string) {
	if r.Method == "GET" {
		if id == "" {
			l, o := page(r)
			items, e := a.list("sessions", u.UID, l, o)
			if e != nil {
				fail(w, 500, "STORE_ERROR", "读取失败")
				return
			}
			respond(w, 200, map[string]any{"items": items})
			return
		}
		var s Session
		if a.get("sessions", id, u.UID, &s) != nil {
			fail(w, 404, "NOT_FOUND", "会话不存在")
			return
		}
		respond(w, 200, s)
		return
	}
	if r.Method == "DELETE" && id != "" {
		s := a.sessions[id]
		if s == nil || s.Owner != u.UID {
			var old Session
			if a.get("sessions", id, u.UID, &old) != nil {
				fail(w, 404, "NOT_FOUND", "会话不存在")
				return
			}
			respond(w, 200, old)
			return
		}
		reason := "USER_CLOSED"
		if s.State != "ACTIVE" {
			reason = "CANCELLED"
		}
		a.closeLocked(s, reason, false)
		respond(w, 200, s)
		return
	}
	if r.Method == "POST" && len(seg) == 3 && seg[2] == "ticket" {
		s := a.sessions[id]
		if s == nil || s.Owner != u.UID {
			fail(w, 404, "NOT_FOUND", "会话不存在")
			return
		}
		if s.State != "CREATED" || time.Now().After(s.ExpiresAt) {
			fail(w, 409, "TICKET_EXPIRED", "会话已失效，请重新连接")
			return
		}
		origin, _ := browserOrigin(r)
		if s.Origin != origin {
			fail(w, 403, "ORIGIN_REJECTED", "请从创建会话的页面重新连接")
			return
		}
		ticket := randomID()
		s.Ticket = sha256.Sum256([]byte(ticket))
		s.TicketExpiry = time.Now().UTC().Add(45 * time.Second)
		if s.TicketExpiry.After(s.ExpiresAt) {
			s.TicketExpiry = s.ExpiresAt
		}
		respond(w, 200, map[string]any{"ticket": ticket, "expiresAt": s.TicketExpiry, "webSocketPath": Prefix + "/api/v1/tunnel", "subprotocol": "guacamole"})
		return
	}
	if r.Method != "POST" || id != "" {
		fail(w, 405, "METHOD_NOT_ALLOWED", "不支持该操作")
		return
	}
	var in sessionRequest
	if !decode(w, r, &in) {
		return
	}
	if in.RenderMode == "" {
		in.RenderMode = "balanced"
	}
	if !validRenderMode(in.RenderMode) {
		fail(w, 400, "INVALID_RENDER_MODE", "画质模式无效")
		return
	}
	var c Connection
	if a.get("connections", in.ConnectionID, u.UID, &c) != nil {
		fail(w, 404, "NOT_FOUND", "连接不存在")
		return
	}
	if in.Display.Width < 320 || in.Display.Width > 4096 || in.Display.Height < 200 || in.Display.Height > 2160 || in.Display.DPI < 72 || in.Display.DPI > 192 {
		fail(w, 400, "INVALID_DISPLAY", "显示尺寸不合法")
		return
	}
	if len(in.Timezone) > 80 {
		fail(w, 400, "INVALID_TIMEZONE", "时区无效")
		return
	}
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	if _, e := time.LoadLocation(in.Timezone); e != nil {
		fail(w, 400, "INVALID_TIMEZONE", "时区无效")
		return
	}
	total, own := 0, 0
	for _, s := range a.sessions {
		if !terminal(s.State) {
			total++
			if s.Owner == u.UID {
				own++
			}
		}
	}
	if total >= a.limits.Total || own >= a.limits.PerUser {
		fail(w, 429, "SESSION_LIMIT", "已达到会话上限，请先断开其他会话")
		return
	}
	fingerprint := ""
	if c.Protocol == "rdp" {
		var err error
		fingerprint, err = a.consumeCertificateCheck(in.CertificateCheck, c, u, r)
		if err != nil {
			fail(w, 409, "CERTIFICATE_CONFIRMATION_REQUIRED", err.Error())
			return
		}
	}
	origin, _ := browserOrigin(r)
	s := &Session{CertificateFingerprint: fingerprint, Origin: origin, ID: randomID(), Owner: u.UID, ConnectionID: c.ID, Config: c, RenderMode: in.RenderMode, Display: in.Display, Timezone: in.Timezone, State: "CREATED", CreatedAt: time.Now().UTC()}
	s.ExpiresAt = s.CreatedAt.Add(90 * time.Second)
	if in.Ref != "" {
		key := sha256.Sum256([]byte(in.Ref))
		e, ok := a.ephemerals[key]
		if !ok || e.Owner != u.UID || e.ConnectionID != c.ID {
			fail(w, 409, "EPHEMERAL_CREDENTIAL_CONSUMED", "临时凭据无效或已使用，请重新输入")
			return
		}
		delete(a.ephemerals, key)
		if time.Now().After(e.Expires) {
			fail(w, 410, "EPHEMERAL_CREDENTIAL_EXPIRED", "临时凭据已过期")
			return
		}
		s.Login = e.Login
		if e.Expires.Before(s.ExpiresAt) {
			s.ExpiresAt = e.Expires
		}
	} else if c.CredentialID != "" {
		login, e := a.login(c, u.UID)
		if e != nil {
			fail(w, 500, "SECRET_ERROR", "凭据无法解密，请检查密钥备份或重新保存凭据")
			return
		}
		s.Login = login
	} else {
		fail(w, 400, "CREDENTIAL_REQUIRED", "请先输入本次连接凭据")
		return
	}
	data, _ := json.Marshal(s)
	_, e := a.db.Exec("INSERT INTO objects VALUES('sessions',?,?,1,?)", s.ID, u.UID, data)
	if e != nil {
		s.Login = Credential{}
		fail(w, 500, "STORE_ERROR", "会话保存失败")
		return
	}
	a.sessions[s.ID] = s
	a.audit(u.UID, "SESSION_CREATED", s.ID)
	respond(w, 201, s)
}

// Dynamic resize uses the same framebuffer limits as session creation.
func validDisplayResize(parts []string) bool {
	if len(parts) != 3 {
		return false
	}
	width, widthErr := strconv.Atoi(parts[1])
	height, heightErr := strconv.Atoi(parts[2])
	return widthErr == nil && heightErr == nil && width >= 320 && width <= 4096 && height >= 200 && height <= 2160
}

func validRenderMode(mode string) bool {
	return mode == "balanced" || mode == "clear" || mode == "smooth"
}

func params(s *Session) map[string]string {
	p := map[string]string{"hostname": s.Config.Host, "port": strconv.Itoa(s.Config.Port), "username": s.Login.Username, "password": s.Login.Password, "domain": s.Login.Domain, "security": "nla", "ignore-cert": "false", "disable-audio": "true", "enable-drive": "false", "enable-printing": "false", "enable-sftp": "false", "read-only": "false", "resize-method": "none", "disable-copy": "true", "disable-paste": "true"}
	if s.Config.Protocol == "rdp" {
		p["resize-method"] = "display-update"
		// Enable GFX and adaptive encoding by default; retain a text-focused fallback.
		p["force-lossless"] = "false"
		p["disable-gfx"] = "false"
		if s.RenderMode == "clear" {
			p["force-lossless"] = "true"
			p["disable-gfx"] = "true"
		}
		p["enable-font-smoothing"] = "true"
		// Match negotiated pixels/DPI instead of letting guacd rescale size updates.
		p["width"] = strconv.Itoa(s.Display.Width)
		p["height"] = strconv.Itoa(s.Display.Height)
		p["dpi"] = strconv.Itoa(s.Display.DPI)
		p["cert-fingerprints"] = "sha256:" + s.CertificateFingerprint
	}
	if s.Config.Protocol == "vnc" {
		delete(p, "username")
		delete(p, "domain")
		p["force-lossless"] = "false"
		p["quality-level"] = "6"
		p["compress-level"] = "2"
		if s.RenderMode == "clear" {
			p["force-lossless"] = "true"
		}
		if s.RenderMode == "smooth" {
			p["quality-level"] = "4"
			p["compress-level"] = "1"
		}
	}
	if s.Config.Settings.Clipboard == "text" {
		p["disable-copy"] = "false"
		p["disable-paste"] = "false"
	}
	return p
}
func (a *App) tunnel(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != "GET" {
		fail(w, 405, "METHOD_NOT_ALLOWED", "需要 WebSocket")
		return
	}
	offered := false
	for _, protocol := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if strings.TrimSpace(protocol) == "guacamole" {
			offered = true
		}
	}
	if !offered {
		fail(w, 400, "PROTOCOL_ERROR", "需要 guacamole 子协议")
		return
	}
	ticket := r.URL.Query().Get("ticket")
	if len(ticket) != 64 {
		fail(w, 401, "TICKET_EXPIRED", "连接票据无效")
		return
	}
	hash := sha256.Sum256([]byte(ticket))
	origin, _ := browserOrigin(r)
	a.mu.Lock()
	var s *Session
	for _, candidate := range a.sessions {
		if candidate.Ticket == hash && candidate.Owner == u.UID && candidate.Origin == origin && candidate.State == "CREATED" && time.Now().Before(candidate.TicketExpiry) && time.Now().Before(candidate.ExpiresAt) {
			s = candidate
			break
		}
	}
	if s == nil || a.stopping {
		a.mu.Unlock()
		fail(w, 401, "TICKET_EXPIRED", "连接票据失效，请重新连接")
		return
	}
	s.Ticket = [32]byte{}
	s.State = "CONNECTING"
	ctx, cancel := context.WithCancel(r.Context())
	s.Cancel = cancel
	s.Timer = time.AfterFunc(30*time.Second, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if s.State == "CONNECTING" {
			a.closeLocked(s, "CONNECT_TIMEOUT", true)
		}
	})
	a.persist(s)
	p := params(s)
	s.Login = Credential{}
	a.mu.Unlock()
	defer cancel()
	ws, e := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"guacamole"}, OriginPatterns: []string{webSocketOriginHost(r)}})
	if e != nil {
		a.mu.Lock()
		a.closeLocked(s, "IO_ERROR", true)
		a.mu.Unlock()
		return
	}
	ws.SetReadLimit(guac.MaxInstruction)
	a.mu.Lock()
	if terminal(s.State) {
		a.mu.Unlock()
		ws.CloseNow()
		return
	}
	s.WS = ws
	a.mu.Unlock()
	finish := func(reason string) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.closeLocked(s, reason, reason != "USER_CLOSED")
	}
	up, e := (&net.Dialer{}).DialContext(ctx, "tcp", a.cfg.Guacd)
	if e != nil {
		finish("ENGINE_UNAVAILABLE")
		return
	}
	a.mu.Lock()
	if terminal(s.State) {
		a.mu.Unlock()
		up.Close()
		return
	}
	s.Upstream = up
	a.mu.Unlock()
	up.SetDeadline(time.Now().Add(30 * time.Second))
	decoder, e := guac.Handshake(up, s.Config.Protocol, p, s.Display.Width, s.Display.Height, s.Display.DPI, s.Timezone)
	for k := range p {
		delete(p, k)
	}
	if e != nil {
		finish("PROTOCOL_ERROR")
		return
	}
	writeWS := func(parts ...string) error {
		writeCtx, done := context.WithTimeout(ctx, 10*time.Second)
		defer done()
		return ws.Write(writeCtx, websocket.MessageText, []byte(guac.Encode(parts...)))
	}
	if e = writeWS("", s.ID); e != nil {
		finish("IO_ERROR")
		return
	}
	up.SetDeadline(time.Time{})
	errs := make(chan string, 2)
	go func() {
		seenDisplay := false
		for {
			up.SetReadDeadline(time.Now().Add(45 * time.Second))
			parts, e := decoder.Read()
			if e != nil {
				errs <- "UPSTREAM_DISCONNECTED"
				return
			}
			if parts[0] == "size" && len(parts) == 4 && parts[1] == "0" {
				width, _ := strconv.Atoi(parts[2])
				height, _ := strconv.Atoi(parts[3])
				seenDisplay = width > 0 && height > 0
			}
			if parts[0] == "sync" && seenDisplay {
				a.mu.Lock()
				if s.State == "CONNECTING" {
					s.State = "ACTIVE"
					s.LastInput = time.Now()
					s.Timer.Stop()
					a.persist(s)
				}
				a.mu.Unlock()
			}
			if parts[0] == "error" {
				reason, status := remoteFailure(parts)
				log.Printf("remote-desktop remote_failure session=%s reason=%s guac_status=%s", s.ID, reason, status)
				writeWS("error", "远端连接失败，请查看会话详细状态", status)
				errs <- reason
				return
			}
			if e = writeWS(parts...); e != nil {
				errs <- "IO_ERROR"
				return
			}
		}
	}()
	go func() {
		streams := map[string]int{}
		for {
			typ, b, e := ws.Read(ctx)
			if e != nil {
				errs <- "UPSTREAM_DISCONNECTED"
				return
			}
			if typ != websocket.MessageText {
				errs <- "PROTOCOL_ERROR"
				return
			}
			d := guac.NewDecoder(bytes.NewReader(b))
			for {
				parts, e := d.Read()
				if e == io.EOF {
					break
				}
				if e != nil {
					errs <- "PROTOCOL_ERROR"
					return
				}
				op := parts[0]
				if op == "" {
					if len(parts) == 3 && parts[1] == "ping" {
						if writeWS(parts...) != nil {
							errs <- "IO_ERROR"
							return
						}
					} else {
						errs <- "PROTOCOL_ERROR"
						return
					}
					continue
				}
				switch op {
				case "size":
					if s.Config.Protocol != "rdp" || !validDisplayResize(parts) {
						errs <- "PROTOCOL_ERROR"
						return
					}
				case "sync", "nop":
				case "key", "mouse":
					a.mu.Lock()
					s.LastInput = time.Now()
					a.mu.Unlock()
				case "clipboard":
					if s.Config.Settings.Clipboard != "text" || len(parts) != 3 || parts[2] != "text/plain" || len(streams) >= 4 {
						errs <- "PROTOCOL_ERROR"
						return
					}
					streams[parts[1]] = 0
				case "blob":
					if len(parts) != 3 {
						errs <- "PROTOCOL_ERROR"
						return
					}
					n, ok := streams[parts[1]]
					if !ok || n+len(parts[2]) > 1<<20 {
						errs <- "PROTOCOL_ERROR"
						return
					}
					streams[parts[1]] = n + len(parts[2])
				case "end":
					if len(parts) != 2 {
						errs <- "PROTOCOL_ERROR"
						return
					}
					delete(streams, parts[1])
				case "ack":
				case "disconnect":
					errs <- "USER_CLOSED"
					return
				default:
					errs <- "PROTOCOL_ERROR"
					return
				}
				up.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if _, e = io.WriteString(up, guac.Encode(parts...)); e != nil {
					errs <- "IO_ERROR"
					return
				}
			}
		}
	}()
	reason := <-errs
	finish(reason)
	<-errs
}

var _ = fmt.Sprint
