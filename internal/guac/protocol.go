// Package guac implements the bounded Unicode-character Guacamole wire protocol.
package guac

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxInstruction = 4 << 20

func Encode(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d.%s", utf8.RuneCountInString(p), p)
	}
	b.WriteByte(';')
	return b.String()
}

type Decoder struct{ r *bufio.Reader }

func NewDecoder(r io.Reader) *Decoder { return &Decoder{bufio.NewReader(r)} }
func (d *Decoder) Read() ([]string, error) {
	var parts []string
	total := 0
	for {
		n := 0
		digits := 0
		for {
			c, e := d.r.ReadByte()
			if e != nil {
				if e == io.EOF && (digits > 0 || len(parts) > 0) {
					e = io.ErrUnexpectedEOF
				}
				return nil, e
			}
			if c == '.' && digits > 0 {
				break
			}
			if c < '0' || c > '9' || digits >= 8 {
				return nil, fmt.Errorf("invalid length")
			}
			n = n*10 + int(c-'0')
			digits++
			if n > MaxInstruction {
				return nil, fmt.Errorf("field too large")
			}
		}
		var b strings.Builder
		b.Grow(min(n, MaxInstruction-total))
		for i := 0; i < n; {
			// Image blobs are ASCII base64. Copy buffered ASCII in blocks instead
			// of decoding and re-encoding every byte as a Unicode rune. Non-ASCII
			// fields retain the strict character-count/UTF-8 path below.
			if available := min(n-i, d.r.Buffered()); available > 0 {
				buffer, _ := d.r.Peek(available)
				count := 0
				for count < len(buffer) && buffer[count] < utf8.RuneSelf {
					count++
				}
				if count > 0 {
					total += count
					if total > MaxInstruction {
						return nil, fmt.Errorf("instruction too large")
					}
					b.Write(buffer[:count])
					d.r.Discard(count)
					i += count
					continue
				}
			}
			r, size, e := d.r.ReadRune()
			if e != nil {
				if e == io.EOF {
					e = io.ErrUnexpectedEOF
				}
				return nil, e
			}
			if r == utf8.RuneError && size == 1 {
				return nil, fmt.Errorf("invalid UTF-8")
			}
			total += size
			if total > MaxInstruction {
				return nil, fmt.Errorf("instruction too large")
			}
			b.WriteRune(r)
			i++
		}
		parts = append(parts, b.String())
		if len(parts) > 256 {
			return nil, fmt.Errorf("too many fields")
		}
		c, e := d.r.ReadByte()
		if e != nil {
			return nil, e
		}
		if c == ';' {
			return parts, nil
		}
		if c != ',' {
			return nil, fmt.Errorf("invalid delimiter")
		}
	}
}
func Handshake(rw io.ReadWriter, protocol string, params map[string]string, width, height, dpi int, timezone string) (*Decoder, error) {
	send := func(p ...string) error { _, e := io.WriteString(rw, Encode(p...)); return e }
	if e := send("select", protocol); e != nil {
		return nil, e
	}
	d := NewDecoder(rw)
	args, e := d.Read()
	if e != nil {
		return nil, e
	}
	if len(args) < 2 || args[0] != "args" {
		return nil, fmt.Errorf("expected args")
	}
	for _, p := range [][]string{{"size", strconv.Itoa(width), strconv.Itoa(height), strconv.Itoa(dpi)}, {"audio"}, {"video"}, {"image", "image/png", "image/jpeg"}, {"timezone", timezone}} {
		if e = send(p...); e != nil {
			return nil, e
		}
	}
	values := []string{"connect"}
	for _, name := range args[1:] {
		if strings.HasPrefix(name, "VERSION_") {
			switch name {
			case "VERSION_1_0_0", "VERSION_1_1_0", "VERSION_1_3_0", "VERSION_1_5_0":
				values = append(values, name)
			default:
				return nil, fmt.Errorf("unsupported protocol version")
			}
		} else {
			values = append(values, params[name])
		}
	}
	if e = send(values...); e != nil {
		return nil, e
	}
	ready, e := d.Read()
	if e != nil {
		return nil, e
	}
	if len(ready) != 2 || ready[0] != "ready" {
		return nil, fmt.Errorf("engine handshake rejected")
	}
	return d, nil
}
