package app

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"
)

type connectionProbe struct {
	Reachable bool      `json:"reachable"`
	LatencyMS *float64  `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
	Message   string    `json:"message"`
}

// probeAPI is entered with a.mu held by route. The network operation releases it.
func (a *App) probeAPI(w http.ResponseWriter, r *http.Request, u User, id string) {
	var c Connection
	if a.get("connections", id, u.UID, &c) != nil {
		fail(w, 404, "NOT_FOUND", "连接不存在")
		return
	}
	// Each saved target has its own cooldown so opening the connection list
	// can check multiple targets without them rate-limiting each other.
	probeKey := u.UID + "\x00" + id
	if time.Since(a.probes[probeKey]) < 5*time.Second {
		fail(w, 429, "RATE_LIMIT", "请稍候再探测")
		return
	}
	a.probes[probeKey] = time.Now()
	result := func() connectionProbe {
		a.mu.Unlock()
		defer a.mu.Lock()
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		start := time.Now()
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
		elapsed := time.Since(start).Seconds() * 1000
		result := connectionProbe{CheckedAt: time.Now().UTC(), Message: "NAS 无法连接目标 TCP 端口，请检查网络及远程服务"}
		if err == nil {
			conn.Close()
			result.Reachable = true
			result.LatencyMS = &elapsed
			result.Message = "TCP 端口可达；尚未验证远端认证或桌面可用性"
		}
		return result
	}()
	var current Connection
	if a.get("connections", id, u.UID, &current) != nil || current.Version != c.Version {
		fail(w, 409, "CONNECTION_CHANGED", "连接已改变，请刷新后重新检测")
		return
	}
	if !result.Reachable {
		respond(w, 422, map[string]any{"probe": result, "error": map[string]string{"code": "TARGET_UNREACHABLE", "message": result.Message, "requestId": w.Header().Get("X-Request-ID")}})
		return
	}
	respond(w, 200, result)
}
