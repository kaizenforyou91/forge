package config

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {

	key := []byte("0123456789abcdef0123456789abcdef")

	data := []byte("Forge Encryption Test")

	encrypted, err := Encrypt(data, key)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, encrypted) {
		t.Fatal("data should be encrypted")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decrypted) {
		t.Fatal("decrypt failed")
	}
}
