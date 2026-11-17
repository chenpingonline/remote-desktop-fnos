package rdpcert

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func certificate(t *testing.T, expired bool) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	end := time.Now().Add(time.Hour)
	if expired {
		end = time.Now().Add(-time.Minute)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "RDP test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: end, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, parsed
}
func server(t *testing.T, cert tls.Certificate) (string, int, <-chan bool) {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { l.Close() })
	host, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(p)
	noCredentials := make(chan bool, 1)
	go func() {
		c, e := l.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(3 * time.Second))
		request := make([]byte, 19)
		_, e = io.ReadFull(c, request)
		if e != nil {
			return
		}
		expected := []byte{3, 0, 0, 19, 14, 224, 0, 0, 0, 0, 0, 1, 0, 8, 0, 3, 0, 0, 0}
		if !bytes.Equal(request, expected) {
			noCredentials <- false
			return
		}
		c.Write([]byte{3, 0, 0, 19, 14, 208, 0, 0, 0, 0, 0, 2, 0, 8, 0, 2, 0, 0, 0})
		secure := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
		if secure.Handshake() != nil {
			noCredentials <- false
			return
		}
		one := make([]byte, 1)
		n, _ := secure.Read(one)
		noCredentials <- n == 0
	}()
	return host, port, noCredentials
}
func TestProbeVerifiesRootsAndSendsNoCredentials(t *testing.T) {
	cert, root := certificate(t, false)
	for _, trusted := range []bool{true, false} {
		t.Run(strconv.FormatBool(trusted), func(t *testing.T) {
			roots := x509.NewCertPool()
			if trusted {
				roots.AddCert(root)
			}
			host, port, noCredentials := server(t, cert)
			info, e := Probe(context.Background(), host, port, roots)
			if e != nil || info.Verified != trusted || !info.CanTrust || len(info.Fingerprint) != 64 {
				t.Fatal(info, e)
			}
			select {
			case ok := <-noCredentials:
				if !ok {
					t.Fatal("probe sent credentials or invalid negotiation")
				}
			case <-time.After(4 * time.Second):
				t.Fatal("server did not finish")
			}
		})
	}
}
func TestExpiredCertificateCannotBeTrusted(t *testing.T) {
	cert, root := certificate(t, true)
	roots := x509.NewCertPool()
	roots.AddCert(root)
	host, port, _ := server(t, cert)
	info, e := Probe(context.Background(), host, port, roots)
	if e != nil || info.CanTrust || info.Verified || info.Verification != "expired" {
		t.Fatal(info, e)
	}
}
func TestProbeRejectsOversizedResponse(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	go func() {
		c, e := l.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		b := make([]byte, 19)
		io.ReadFull(c, b)
		c.Write([]byte{3, 0, 255, 255})
	}()
	host, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(p)
	_, e := Probe(context.Background(), host, port, nil)
	if e == nil || !strings.Contains(e.Error(), "无效") {
		t.Fatal("invalid response accepted", e)
	}
}
