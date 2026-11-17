package app

import (
	"context"
	"net/http"
	"time"
)

// Called with a.mu held, like the other connection actions.
func (a *App) wakeAPI(w http.ResponseWriter, r *http.Request, u User, id string) {
	if r.Method != "POST" {
		fail(w, 405, "METHOD_NOT_ALLOWED", "不支持该操作")
		return
	}
	var c Connection
	if a.get("connections", id, u.UID, &c) != nil {
		fail(w, 404, "NOT_FOUND", "连接不存在")
		return
	}
	if !c.WOL.Enabled {
		fail(w, 400, "WOL_DISABLED", "请先在编辑连接中启用并配置网络唤醒")
		return
	}
	if time.Since(a.wakes[u.UID]) < 10*time.Second {
		fail(w, 429, "RATE_LIMIT", "请等待 10 秒后再发送唤醒包")
		return
	}
	a.wakes[u.UID] = time.Now()
	// DNS and UDP writes must not hold the application's global mutex.
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	targets, err := a.wakeSender(ctx, c.Host, c.WOL)
	cancel()
	a.mu.Lock()
	if err != nil {
		a.audit(u.UID, "WAKE_FAILED", id)
		fail(w, 422, "WOL_SEND_FAILED", err.Error())
		return
	}
	a.audit(u.UID, "WAKE_SENT", id)
	respond(w, 200, map[string]any{"message": "唤醒包已发送，尚未确认电脑已开机", "broadcasts": targets, "sentAt": time.Now().UTC()})
}

// Successful sends are already durable audit events. Recover the latest for
// each owned connection, including events older than the audit UI's first page.
func (a *App) wakeHistoryAPI(w http.ResponseWriter, r *http.Request, u User) {
	rows, err := a.db.Query(`SELECT object_id,created_at FROM (
  SELECT object_id,created_at,ROW_NUMBER() OVER (PARTITION BY object_id ORDER BY rowid DESC) AS rn
  FROM audit_events WHERE owner=? AND action='WAKE_SENT'
 ) WHERE rn=1`, u.UID)
	if err != nil {
		fail(w, 500, "STORE_ERROR", "读取唤醒记录失败")
		return
	}
	defer rows.Close()
	items := map[string]string{}
	for rows.Next() {
		var id, at string
		if err = rows.Scan(&id, &at); err != nil {
			fail(w, 500, "STORE_ERROR", "读取唤醒记录失败")
			return
		}
		items[id] = at
	}
	if rows.Err() != nil {
		fail(w, 500, "STORE_ERROR", "读取唤醒记录失败")
		return
	}
	respond(w, 200, map[string]any{"items": items})
}
