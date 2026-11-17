// Package rdpcert inspects RDP TLS certificates without transmitting credentials.
package rdpcert

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type Info struct {
	Fingerprint  string    `json:"fingerprint"`
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	NotBefore    time.Time `json:"notBefore"`
	NotAfter     time.Time `json:"notAfter"`
	Verified     bool      `json:"verified"`
	Verification string    `json:"verification"`
	CanTrust     bool      `json:"canTrust"`
}

func Probe(ctx context.Context, host string, port int, roots *x509.CertPool) (Info, error) {
	var info Info
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprint(port)))
	if err != nil {
		return info, fmt.Errorf("目标 TCP 端口不可达")
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	// X.224 requests TLS + NLA. Contains no username, password or routing cookie.
	request := []byte{3, 0, 0, 19, 14, 224, 0, 0, 0, 0, 0, 1, 0, 8, 0, 3, 0, 0, 0}
	if _, err = conn.Write(request); err != nil {
		return info, fmt.Errorf("无法发送 RDP 协商请求")
	}
	head := make([]byte, 4)
	if _, err = io.ReadFull(conn, head); err != nil {
		return info, fmt.Errorf("目标未返回 RDP 协商响应")
	}
	size := int(binary.BigEndian.Uint16(head[2:]))
	if head[0] != 3 || head[1] != 0 || size < 11 || size > 4096 {
		return info, fmt.Errorf("目标返回了无效的 RDP 响应，请检查地址和端口")
	}
	packet := make([]byte, size)
	copy(packet, head)
	if _, err = io.ReadFull(conn, packet[4:]); err != nil {
		return info, fmt.Errorf("RDP 协商响应不完整")
	}
	if size != 19 || packet[5] != 208 || packet[11] != 2 || binary.LittleEndian.Uint16(packet[13:15]) != 8 || binary.LittleEndian.Uint32(packet[15:19]) != 2 {
		return info, fmt.Errorf("目标未选择 NLA，请检查 Windows 远程桌面安全设置")
	}
	// Inspect only. Verification is performed below with the system/package roots;
	// no CredSSP or other login data is sent on this connection.
	secure := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: true})
	if err = secure.HandshakeContext(ctx); err != nil {
		return info, fmt.Errorf("RDP TLS 握手失败，请检查目标证书或 TLS 配置")
	}
	state := secure.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return info, fmt.Errorf("目标没有提供证书")
	}
	leaf := state.PeerCertificates[0]
	hash := sha256.Sum256(leaf.Raw)
	info = Info{Fingerprint: hex.EncodeToString(hash[:]), Subject: bounded(leaf.Subject.String()), Issuer: bounded(leaf.Issuer.String()), NotBefore: leaf.NotBefore, NotAfter: leaf.NotAfter, Verification: "unknown_authority"}
	now := time.Now()
	info.CanTrust = !now.Before(leaf.NotBefore) && now.Before(leaf.NotAfter)
	intermediates := x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	_, err = leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots, Intermediates: intermediates, CurrentTime: now})
	if err == nil {
		info.Verified = true
		info.Verification = "valid"
	} else {
		var hostname x509.HostnameError
		var invalid x509.CertificateInvalidError
		if errors.As(err, &hostname) {
			info.Verification = "name_mismatch"
		}
		if errors.As(err, &invalid) && invalid.Reason == x509.Expired {
			info.Verification = "expired"
		}
	}
	if !info.CanTrust {
		info.Verification = "expired"
	}
	return info, nil
}
func bounded(s string) string {
	r := []rune(s)
	if len(r) > 1024 {
		return string(r[:1024])
	}
	return s
}
