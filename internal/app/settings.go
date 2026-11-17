package app

import (
	"encoding/json"
	"net/http"
)

type Limits struct {
	PerUser     int `json:"perUserLimit"`
	Total       int `json:"totalLimit"`
	IdleMinutes int `json:"idleTimeoutMinutes"`
	MaxHours    int `json:"maxDurationHours"`
	Version     int `json:"version"`
}

func (a *App) settingsAPI(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Admin {
		fail(w, 403, "FORBIDDEN", "仅管理员可修改应用运行设置")
		return
	}
	if r.Method == "GET" {
		respond(w, 200, a.limits)
		return
	}
	if r.Method != "PATCH" {
		fail(w, 405, "METHOD_NOT_ALLOWED", "不支持该操作")
		return
	}
	var v Limits
	if !decode(w, r, &v) {
		return
	}
	if v.Version != a.limits.Version {
		fail(w, 409, "VERSION_CONFLICT", "设置已更新，请刷新")
		return
	}
	if v.PerUser < 1 || v.PerUser > 8 || v.Total < v.PerUser || v.Total > 16 || v.IdleMinutes < 1 || v.IdleMinutes > 120 || v.MaxHours < 1 || v.MaxHours > 24 {
		fail(w, 400, "INVALID_SETTINGS", "请检查会话配额与超时范围")
		return
	}
	v.Version++
	data, _ := json.Marshal(v)
	if _, e := a.db.Exec("INSERT INTO app_settings VALUES('limits',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", data); e != nil {
		fail(w, 500, "STORE_ERROR", "保存设置失败")
		return
	}
	a.limits = v
	a.audit(u.UID, "UPDATE_LIMITS", "")
	respond(w, 200, v)
}
