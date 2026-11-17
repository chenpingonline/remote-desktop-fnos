package app

import "testing"

func TestDynamicDisplaySizeBounds(t *testing.T) {
	for _, parts := range [][]string{
		{"size", "320", "200"}, {"size", "4096", "2160"}, {"size", "1920", "1080"},
	} {
		if !validDisplayResize(parts) {
			t.Fatalf("valid size rejected: %v", parts)
		}
	}
	for _, parts := range [][]string{
		{"size"}, {"size", "1920"}, {"size", "1920", "1080", "96"},
		{"size", "0", "1080"}, {"size", "4097", "1080"}, {"size", "1920", "2161"},
		{"size", "319", "200"}, {"size", "320", "199"}, {"size", "NaN", "1080"},
		{"size", "99999999999999999999999", "1080"},
	} {
		if validDisplayResize(parts) {
			t.Fatalf("invalid size accepted: %v", parts)
		}
	}
}

func TestRDPDisplayUpdateNegotiation(t *testing.T) {
	s := &Session{Config: Connection{Protocol: "rdp"}, Display: Display{1920, 1200, 192}}
	if params(s)["disable-gfx"] != "false" || params(s)["enable-font-smoothing"] != "true" {
		t.Fatal("RDP must enable GFX and smooth fonts by default")
	}
	if params(s)["force-lossless"] != "false" {
		t.Fatal("RDP must allow adaptive image compression by default")
	}
	if params(s)["resize-method"] != "display-update" {
		t.Fatal("RDP display updates disabled")
	}
	if p := params(s); p["width"] != "1920" || p["height"] != "1200" || p["dpi"] != "192" {
		t.Fatal("RDP must negotiate exact framebuffer dimensions and DPI", p["width"], p["height"], p["dpi"])
	}
	s.Config.Protocol = "vnc"
	if params(s)["resize-method"] != "none" {
		t.Fatal("RDP resize enabled for VNC")
	}
}

func TestRenderModeNegotiation(t *testing.T) {
	for _, mode := range []string{"", "balanced", "smooth", "clear"} {
		t.Run("rdp/"+mode, func(t *testing.T) {
			p := params(&Session{Config: Connection{Protocol: "rdp"}, RenderMode: mode})
			want := "false"
			if mode == "clear" {
				want = "true"
			}
			if p["force-lossless"] != want || p["disable-gfx"] != want {
				t.Fatalf("incorrect RDP codec mode %q: %v", mode, p)
			}
			if p["disable-copy"] != "true" || p["disable-paste"] != "true" || p["security"] != "nla" {
				t.Fatal("render mode changed security or clipboard policy")
			}
		})
		t.Run("vnc/"+mode, func(t *testing.T) {
			p := params(&Session{Config: Connection{Protocol: "vnc"}, RenderMode: mode})
			if _, ok := p["disable-gfx"]; ok {
				t.Fatal("RDP graphics parameter leaked into VNC")
			}
			if mode == "smooth" && (p["quality-level"] != "4" || p["compress-level"] != "1") {
				t.Fatal("VNC smooth mode not applied")
			}
			if (p["force-lossless"] == "true") != (mode == "clear") {
				t.Fatal("incorrect VNC lossless mode")
			}
		})
	}
	for _, mode := range []string{"balanced", "smooth", "clear"} {
		if !validRenderMode(mode) {
			t.Fatalf("valid mode rejected: %q", mode)
		}
	}
	for _, mode := range []string{"", "fast", "CLEAR", "smooth;hostname=other"} {
		if validRenderMode(mode) {
			t.Fatalf("invalid mode accepted: %q", mode)
		}
	}
}

func TestSessionRenderModeAPI(t *testing.T) {
	for _, mode := range []string{"", "balanced", "smooth", "clear", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			a := testApp(t)
			id := fixture(t, a)
			body := map[string]any{"connectionId": id, "display": Display{800, 600, 96}, "certificateCheck": certificateToken(t, a, id, "alice")}
			if mode != "" {
				body["renderMode"] = mode
			}
			response := call(a, "alice", "POST", "/sessions", body)
			if mode == "invalid" {
				if response.Code != 400 || len(a.sessions) != 0 {
					t.Fatal("invalid mode created a session", response.Code)
				}
				return
			}
			result := object(t, response)
			want := mode
			if want == "" {
				want = "balanced"
			}
			if result["renderMode"] != want {
				t.Fatalf("negotiated mode %v; want %s", result["renderMode"], want)
			}
			session := a.sessions[result["id"].(string)]
			if session.RenderMode != want {
				t.Fatal("mode missing from live session")
			}
		})
	}
}
