package secret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultOwnerAndTamper(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master.key")
	v, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	b, e := v.Encrypt([]byte("private"), "alice:id:rdp:key1")
	if e != nil {
		t.Fatal(e)
	}
	v, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := v.Decrypt(b, "alice:id:rdp:key1")
	if e != nil || string(plain) != "private" {
		t.Fatal(e)
	}
	if _, e = v.Decrypt(b, "bob:id:rdp:key1"); e == nil {
		t.Fatal("cross owner decrypt")
	}
	b[len(b)-1] ^= 1
	if _, e = v.Decrypt(b, "alice:id:rdp:key1"); e == nil {
		t.Fatal("tampered decrypt")
	}
	os.Chmod(path, 0644)
	if _, e = Open(path); e == nil {
		t.Fatal("accepted insecure key")
	}
}
