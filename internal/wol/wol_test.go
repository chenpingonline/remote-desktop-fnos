package wol

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func network(ip string, cidr string) Network {
	_, n, _ := net.ParseCIDR(cidr)
	return Network{net.ParseIP(ip), n}
}
func TestMagicPacket(t *testing.T) {
	p, e := MagicPacket("02-11-22-33-44-55")
	if e != nil {
		t.Fatal(e)
	}
	if len(p) != 102 || !bytes.Equal(p[:6], bytes.Repeat([]byte{255}, 6)) {
		t.Fatal("invalid packet")
	}
	for i := 6; i < len(p); i += 6 {
		if !bytes.Equal(p[i:i+6], []byte{2, 17, 34, 51, 68, 85}) {
			t.Fatal("invalid MAC repetition")
		}
	}
	for _, mac := range []string{"", "ff:ff:ff:ff:ff:ff", "00:00:00:00:00:00", "01:00:5e:00:00:01", "02:11:22:33:44:55:66:77"} {
		if _, e := MagicPacket(mac); e == nil {
			t.Fatalf("accepted %q", mac)
		}
	}
}
func TestSelection(t *testing.T) {
	networks := []Network{network("192.168.1.2", "192.168.1.0/24"), network("10.1.2.3", "10.1.0.0/16")}
	tests := []struct {
		name, host, manual, want string
		fail                     bool
	}{
		{"matched", "10.1.4.5", "", "10.1.255.255", false},
		{"manual overrides target", "10.1.4.5", "192.168.1.255", "192.168.1.255", false},
		{"ambiguous", "172.16.1.2", "", "", true},
		{"not local", "10.1.4.5", "192.168.2.255", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, e := Select(networks, []net.IP{net.ParseIP(tt.host)}, tt.manual)
			if (e != nil) != tt.fail {
				t.Fatalf("unexpected error %v", e)
			}
			if !tt.fail && (len(out) != 1 || out[0].Broadcast.String() != tt.want) {
				t.Fatalf("targets: %v", out)
			}
		})
	}
	out, e := Select(networks[:1], nil, "")
	if e != nil || out[0].Broadcast.String() != "192.168.1.255" {
		t.Fatal(out, e)
	}
	networks = append(networks, network("10.1.4.2", "10.1.4.0/24"))
	out, e = Select(networks, []net.IP{net.ParseIP("10.1.4.5")}, "")
	if e != nil || out[0].Broadcast.String() != "10.1.4.255" {
		t.Fatal(out, e)
	}
	if _, e = Select(networks, []net.IP{net.ParseIP("10.1.4.5"), net.ParseIP("192.168.1.9")}, ""); e == nil {
		t.Fatal("ambiguous DNS targets accepted")
	}
}
func TestBroadcastValidation(t *testing.T) {
	for _, address := range []string{"255.255.255.255", "::1", "127.0.0.1", "0.0.0.0", "224.0.0.1", "invalid"} {
		c := Config{Broadcast: address}
		if c.Validate() == nil {
			t.Fatalf("accepted %s", address)
		}
	}
	c := Config{Enabled: true, MAC: " 02-11-22-33-44-55 ", Broadcast: " 10.1.255.255 "}
	if e := c.Validate(); e != nil || c.MAC != "02:11:22:33:44:55" || c.Broadcast != "10.1.255.255" {
		t.Fatal(c, e)
	}
}

func TestUDPSender(t *testing.T) {
	receiver, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()
	packet, _ := MagicPacket("02:11:22:33:44:55")
	if err = sendPacket(context.Background(), net.IPv4(127, 0, 0, 1), receiver.LocalAddr().(*net.UDPAddr), packet); err != nil {
		t.Fatal(err)
	}
	receiver.SetReadDeadline(time.Now().Add(time.Second))
	for i := 0; i < 3; i++ {
		b := make([]byte, 200)
		n, from, err := receiver.ReadFromUDP(b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b[:n], packet) || !from.IP.IsLoopback() {
			t.Fatal("incorrect datagram")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sendPacket(ctx, net.IPv4(127, 0, 0, 1), receiver.LocalAddr().(*net.UDPAddr), packet) == nil {
		t.Fatal("cancelled send accepted")
	}
}
