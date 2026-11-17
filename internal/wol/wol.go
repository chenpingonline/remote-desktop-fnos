// Package wol sends standard UDP magic packets on directly connected IPv4 networks.
package wol

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

type Config struct {
	Enabled   bool   `json:"enabled"`
	MAC       string `json:"mac"`
	Broadcast string `json:"broadcast"`
}
type Network struct {
	Source net.IP
	Subnet *net.IPNet
}
type Target struct {
	Source    net.IP
	Broadcast net.IP
}

func (c *Config) Validate() error {
	c.MAC = strings.TrimSpace(c.MAC)
	c.Broadcast = strings.TrimSpace(c.Broadcast)
	if c.MAC != "" || c.Enabled {
		mac, err := net.ParseMAC(c.MAC)
		if err != nil || len(mac) != 6 || mac[0]&1 != 0 {
			return errors.New("请填写有效的单播网卡 MAC 地址（6 字节）")
		}
		zero := true
		for _, b := range mac {
			zero = zero && b == 0
		}
		if zero {
			return errors.New("MAC 地址不能全为零")
		}
		c.MAC = mac.String()
	}
	if c.Broadcast != "" {
		ip := net.ParseIP(c.Broadcast)
		if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.Equal(net.IPv4bcast) {
			return errors.New("请填写 NAS 直连网段的 IPv4 子网广播地址，例如 192.168.1.255")
		}
		c.Broadcast = ip.To4().String()
	}
	return nil
}
func MagicPacket(mac string) ([]byte, error) {
	c := Config{Enabled: true, MAC: mac}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	addr, _ := net.ParseMAC(c.MAC)
	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 255
	}
	for i := 6; i < len(packet); i += 6 {
		copy(packet[i:], addr)
	}
	return packet, nil
}
func Networks() ([]Network, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var result []Network
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if n, ok := addr.(*net.IPNet); ok && n.IP.To4() != nil && !n.IP.IsLinkLocalUnicast() {
				result = append(result, Network{n.IP.To4(), n})
			}
		}
	}
	return result, nil
}
func broadcast(n Network) net.IP {
	ip := n.Source.To4()
	mask := n.Subnet.Mask
	if len(mask) != 4 {
		return nil
	}
	out := make(net.IP, 4)
	for i := range out {
		out[i] = ip[i] | ^mask[i]
	}
	return out
}

// Select chooses the most specific matching subnet. Ambiguous multi-NIC setups
// require an explicit broadcast rather than sending on unrelated networks.
func Select(networks []Network, hosts []net.IP, manual string) ([]Target, error) {
	var candidates []Target
	best := -1
	seen := map[string]bool{}
	for _, n := range networks {
		bits, total := n.Subnet.Mask.Size()
		if total != 32 || bits > 30 || n.Source.To4() == nil {
			continue
		}
		b := broadcast(n)
		if manual != "" && b.String() != manual {
			continue
		}
		score := 0
		if manual == "" {
			for _, h := range hosts {
				if h.To4() != nil && n.Subnet.Contains(h) && bits > score {
					score = bits
				}
			}
		}
		if score > best {
			best = score
			candidates = nil
			seen = map[string]bool{}
		}
		if score != best {
			continue
		}
		key := b.String()
		if !seen[key] {
			candidates = append(candidates, Target{n.Source, b})
			seen[key] = true
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("没有匹配的 NAS 直连 IPv4 广播网络，请检查广播地址和网卡状态")
	}
	if len(candidates) > 1 {
		return nil, errors.New("存在多个可用发送网络，无法自动确定，请在连接设置中手动填写广播地址")
	}
	return candidates, nil
}
func Send(ctx context.Context, host string, c Config) ([]string, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, errors.New("此连接尚未启用网络唤醒")
	}
	packet, err := MagicPacket(c.MAC)
	if err != nil {
		return nil, err
	}
	networks, err := Networks()
	if err != nil {
		return nil, errors.New("无法读取 NAS 网络接口")
	}
	var hosts []net.IP
	if c.Broadcast == "" {
		if ip := net.ParseIP(host); ip != nil {
			hosts = []net.IP{ip}
		} else {
			ips, e := net.DefaultResolver.LookupIP(ctx, "ip4", host)
			if e != nil {
				return nil, errors.New("无法解析目标地址，请手动配置广播地址")
			}
			hosts = ips
		}
	}
	targets, err := Select(networks, hosts, c.Broadcast)
	if err != nil {
		return nil, err
	}
	var sent []string
	for _, target := range targets {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err := sendPacket(ctx, target.Source, &net.UDPAddr{IP: target.Broadcast, Port: 9}, packet); err != nil {
			return nil, errors.New("唤醒包发送失败，请检查 NAS 网络和防火墙")
		}
		sent = append(sent, target.Broadcast.String())
	}
	sort.Strings(sent)
	return sent, nil
}

// A bounded UDP sender; the destination is constructed exclusively by Send.
func sendPacket(ctx context.Context, source net.IP, destination *net.UDPAddr, packet []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := net.DialUDP("udp4", &net.UDPAddr{IP: source}, destination)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err = conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		size, err := conn.Write(packet)
		if err != nil {
			return err
		}
		if size != len(packet) {
			return fmt.Errorf("short UDP write")
		}
	}
	return nil
}
