package app

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func normalizeOrigin(raw string) (string, bool) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", false
	}
	u, e := url.Parse(raw)
	if e != nil || u.User != nil || u.Host == "" || u.Hostname() == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return "", false
		}
	}
	if port == "80" && u.Scheme == "http" || port == "443" && u.Scheme == "https" {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return u.Scheme + "://" + host, true
}

func configuredOrigins(raw string) (map[string]struct{}, bool) {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		normalized, ok := normalizeOrigin(strings.TrimSpace(part))
		if !ok {
			return nil, false
		}
		out[normalized] = struct{}{}
	}
	return out, len(out) > 0
}

// browserOrigin reads the browser-generated Origin, never a forwarding header.
func browserOrigin(r *http.Request) (string, bool) {
	values := r.Header.Values("Origin")
	if len(values) != 1 {
		return "", false
	}
	return normalizeOrigin(values[0])
}

func (a *App) validBrowserOrigin(origin string) bool {
	if !a.cfg.Gateway {
		_, ok := a.origins[origin]
		return ok
	}
	// The trusted fnOS gateway authenticates both HTTP and HTTPS desktop entries.
	// Keep accepting only parsed web origins; CSRF tokens remain bound to the
	// exact scheme, host, port and user rather than to rewritten gateway headers.
	_, ok := normalizeOrigin(origin)
	return ok
}

func (a *App) originAllowed(r *http.Request) bool {
	origin, ok := browserOrigin(r)
	return ok && a.validBrowserOrigin(origin) && r.Header.Get("Sec-Fetch-Site") != "cross-site"
}

// An authenticated same-origin /me read bootstraps an origin-bound token.
// Browsers cannot read this response cross-origin: no CORS headers are emitted,
// and the custom header causes a cross-origin fetch to require a preflight.
// This works even when the trusted fnOS gateway rewrites Host internally.
func (a *App) bootstrapOrigin(r *http.Request) (string, bool) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return "", false
	}
	origin, hasOrigin := browserOrigin(r)
	if len(r.Header.Values("Origin")) > 0 && !hasOrigin {
		return "", false
	}
	values := r.Header.Values("X-Remote-Desktop-Origin")
	if len(values) > 1 {
		return "", false
	}
	if len(values) == 1 {
		declared, ok := normalizeOrigin(values[0])
		if !ok {
			return "", false
		}
		if hasOrigin && declared != origin {
			return "", false
		}
		origin = declared
	} else if !hasOrigin {
		return "", false
	}
	return origin, a.validBrowserOrigin(origin)
}

func webSocketOriginHost(r *http.Request) string {
	u, _ := url.Parse(r.Header.Get("Origin"))
	return u.Host
}
