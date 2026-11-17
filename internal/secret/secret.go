package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

type Vault struct{ a cipher.AEAD }

func Open(path string) (*Vault, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	key, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		key = make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return nil, e
		}
		var f *os.File
		f, e = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return nil, e
		}
		_, e = f.Write(key)
		ce := f.Close()
		if e == nil {
			e = ce
		}
	}
	if e != nil {
		return nil, e
	}
	info, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("master key permissions must be 0600")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid master key")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	a, e := cipher.NewGCM(block)
	return &Vault{a}, e
}
func (v *Vault) Encrypt(data []byte, aad string) ([]byte, error) {
	nonce := make([]byte, v.a.NonceSize())
	if _, e := rand.Read(nonce); e != nil {
		return nil, e
	}
	return v.a.Seal(nonce, nonce, data, []byte(aad)), nil
}
func (v *Vault) Decrypt(data []byte, aad string) ([]byte, error) {
	n := v.a.NonceSize()
	if len(data) < n {
		return nil, fmt.Errorf("invalid ciphertext")
	}
	return v.a.Open(nil, data[:n], data[n:], []byte(aad))
}
