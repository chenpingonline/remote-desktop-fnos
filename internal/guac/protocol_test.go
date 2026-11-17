package guac

import (
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
)

type singleByte struct{ r io.Reader }

func (s singleByte) Read(b []byte) (int, error) { return s.r.Read(b[:1]) }
func TestUnicodeFragmentation(t *testing.T) {
	want := []string{"clipboard", "中文é😀;,", ""}
	d := NewDecoder(singleByte{strings.NewReader(Encode(want...) + Encode("sync", "123"))})
	got, e := d.Read()
	if e != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%q %v", got, e)
	}
	got, e = d.Read()
	if e != nil || got[0] != "sync" {
		t.Fatal(got, e)
	}
}
func TestMalformedBounded(t *testing.T) {
	for _, s := range []string{"x.test;", "99999999.a;", "1.\xff;", "1.a!", "2.a;", "0.;,"} {
		d := NewDecoder(strings.NewReader(s))
		if _, e := d.Read(); e == nil {
			if s != "0.;," {
				t.Errorf("accepted %q", s)
			}
		}
	}
}
func TestHandshakeReordered(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	done := make(chan error, 1)
	go func() {
		d := NewDecoder(server)
		p, e := d.Read()
		if e != nil || p[0] != "select" {
			done <- e
			return
		}
		io.WriteString(server, Encode("args", "VERSION_1_5_0", "password", "hostname", "port", "username"))
		for i := 0; i < 6; i++ {
			p, e = d.Read()
			if e != nil {
				done <- e
				return
			}
		}
		if !reflect.DeepEqual(p, []string{"connect", "VERSION_1_5_0", "secret", "host", "5900", "user"}) {
			done <- io.ErrUnexpectedEOF
			return
		}
		io.WriteString(server, Encode("ready", "uuid"))
		done <- nil
	}()
	_, e := Handshake(client, "vnc", map[string]string{"hostname": "host", "password": "secret", "port": "5900", "username": "user"}, 800, 600, 96, "UTC")
	if e != nil {
		t.Fatal(e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}
func FuzzDecode(f *testing.F) {
	for _, s := range []string{Encode("中文", "😀"), "0.;", "100000000.a;", "1.a,0.;"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > MaxInstruction {
			return
		}
		NewDecoder(strings.NewReader(s)).Read()
	})
}

// Image updates mostly contain ASCII base64 payloads. Include metadata so the
// benchmark exercises the complete instruction decoder used by the gateway.
func BenchmarkDecodeImageBlob(b *testing.B) {
	payload := Encode("blob", "1", strings.Repeat("aGVsbG8=", 8192))
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(strings.NewReader(payload))
		if _, err := d.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestASCIIBlockDecodePreservesUnicodeBoundaries(t *testing.T) {
	for _, padding := range []int{0, 1, 4090, 4095, 4096, 8192} {
		value := strings.Repeat("a", padding) + "中文😀é" + strings.Repeat("b", 5000)
		want := []string{"blob", "1", value}
		for _, reader := range []io.Reader{strings.NewReader(Encode(want...)), singleByte{strings.NewReader(Encode(want...))}} {
			got, err := NewDecoder(reader).Read()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("padding=%d: decoded incorrectly: %v", padding, err)
			}
		}
	}
	// A field's character count may fit while its UTF-8 byte count exceeds the limit.
	data := Encode("blob", strings.Repeat("界", MaxInstruction/3+1))
	if _, err := NewDecoder(strings.NewReader(data)).Read(); err == nil {
		t.Fatal("oversized Unicode instruction accepted")
	}
}
