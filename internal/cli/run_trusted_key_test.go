package cli

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestRunTrustedPublicKeyLoadsExactEd25519PKIXPEM(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := writeRunTrustedKeyFile(t, "trusted-key.pem", validRunTrustedKeyPEM(t, publicKey))

	loaded, err := loadRunTrustedPublicKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded, publicKey) {
		t.Fatal("loaded public key does not equal generated key")
	}
	if &loaded[0] == &publicKey[0] {
		t.Fatal("loaded public key aliases generated key")
	}

	loaded[0] ^= 0xff
	again, err := loadRunTrustedPublicKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, publicKey) {
		t.Fatal("mutating returned key changed subsequent loaded evidence")
	}
}

func TestRunTrustedPublicKeyAcceptsTrailingWhitespaceAndPublicPermissions(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data := append(validRunTrustedKeyPEM(t, publicKey), []byte("\r\n \t\n")...)
	path := writeRunTrustedKeyFile(t, "public-readable.pem", data)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadRunTrustedPublicKey(path); err != nil {
		t.Fatalf("public keys do not require private-key permissions: %v", err)
	}
}

func TestRunTrustedPublicKeyRejectsInvalidPathsAndTypes(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(""))
	})
	t.Run("surrounding whitespace path", func(t *testing.T) {
		requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(" trusted-key.pem "))
	})
	t.Run("nonexistent file", func(t *testing.T) {
		requireRunTrustKeyError(
			t,
			loadRunTrustedPublicKeyError(filepath.Join(t.TempDir(), "missing.pem")),
		)
	})
	t.Run("directory", func(t *testing.T) {
		requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(t.TempDir()))
	})
	t.Run("symbolic link", func(t *testing.T) {
		publicKey, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		target := writeRunTrustedKeyFile(t, "target.pem", validRunTrustedKeyPEM(t, publicKey))
		link := filepath.Join(filepath.Dir(target), "trusted-link.pem")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("Windows symlink creation is unavailable: %v", err)
			}
			t.Fatal(err)
		}
		requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(link))
	})
	t.Run("oversized file", func(t *testing.T) {
		path := writeRunTrustedKeyFile(
			t,
			"oversized.pem",
			bytes.Repeat([]byte{'x'}, int(maxRunTrustedKeyFileBytes)+1),
		)
		requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(path))
	})
}

func TestRunTrustedPublicKeyRejectsAlternateOrMalformedFormats(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	validDER := marshalRunTrustedPKIX(t, publicKey)
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)},
		&x509.Certificate{SerialNumber: big.NewInt(1)},
		publicKey,
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string][]byte{
		"raw public key":    append([]byte(nil), publicKey...),
		"base64 public key": []byte(base64.StdEncoding.EncodeToString(publicKey)),
		"malformed PEM":     []byte("-----BEGIN PUBLIC KEY-----\nnot-base64\n-----END PUBLIC KEY-----\n"),
		"wrong PEM label": pem.EncodeToMemory(&pem.Block{
			Type: "ED25519 PUBLIC KEY", Bytes: validDER,
		}),
		"malformed DER": pem.EncodeToMemory(&pem.Block{
			Type: "PUBLIC KEY", Bytes: []byte("not-pkix"),
		}),
		"RSA PKIX": pem.EncodeToMemory(&pem.Block{
			Type: "PUBLIC KEY", Bytes: marshalRunTrustedPKIX(t, &rsaKey.PublicKey),
		}),
		"ECDSA PKIX": pem.EncodeToMemory(&pem.Block{
			Type: "PUBLIC KEY", Bytes: marshalRunTrustedPKIX(t, &ecdsaKey.PublicKey),
		}),
		"certificate": pem.EncodeToMemory(&pem.Block{
			Type: "CERTIFICATE", Bytes: certificateDER,
		}),
		"private key": pem.EncodeToMemory(&pem.Block{
			Type: "PRIVATE KEY", Bytes: privateDER,
		}),
		"multiple PEM blocks": append(
			validRunTrustedKeyPEM(t, publicKey),
			validRunTrustedKeyPEM(t, publicKey)...,
		),
		"PEM headers": pem.EncodeToMemory(&pem.Block{
			Type: "PUBLIC KEY",
			Headers: map[string]string{
				"Comment": "not permitted",
			},
			Bytes: validDER,
		}),
		"trailing non-whitespace": append(
			validRunTrustedKeyPEM(t, publicKey),
			[]byte("unexpected")...,
		),
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeRunTrustedKeyFile(t, "invalid.pem", data)
			requireRunTrustKeyError(t, loadRunTrustedPublicKeyError(path))
		})
	}
}

func TestRunTrustKeyIDValidation(t *testing.T) {
	rejected := map[string]string{
		"blank":               "",
		"whitespace only":     " \t\r\n ",
		"leading whitespace":  " forge-dev",
		"trailing whitespace": "forge-dev ",
		"newline":             "trusted\nkey",
		"carriage return":     "trusted\rkey",
		"tab":                 "trusted\tkey",
		"NUL":                 "trusted\x00key",
		"ESC":                 "trusted\x1bkey",
		"DEL":                 "trusted\x7fkey",
		"invalid UTF-8":       string([]byte{'t', 0xff}),
	}
	for name, keyID := range rejected {
		t.Run(name, func(t *testing.T) {
			got, err := validateRunTrustKeyID(keyID)
			if got != "" {
				t.Fatalf("expected empty rejected key ID, got %q", got)
			}
			requireRunTrustKeyError(t, err)
			if !errors.Is(err, compiler.ErrInvalidKeyID) {
				t.Fatalf("expected ErrInvalidKeyID, got %v", err)
			}
		})
	}

	const keyID = "forge-dev"
	got, err := validateRunTrustKeyID(keyID)
	if err != nil {
		t.Fatal(err)
	}
	if got != keyID {
		t.Fatalf("expected exact key ID %q, got %q", keyID, got)
	}

	unicodeKeyID := "kunci-é-e\u0301"
	got, err = validateRunTrustKeyID(unicodeKeyID)
	if err != nil || got != unicodeKeyID {
		t.Fatalf("Unicode key ID was not preserved exactly: got %q, err %v", got, err)
	}
}

func loadRunTrustedPublicKeyError(path string) error {
	_, err := loadRunTrustedPublicKey(path)
	return err
}

func requireRunTrustKeyError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidTrustKey) {
		t.Fatalf("expected ErrInvalidTrustKey, got %v", err)
	}
}

func writeRunTrustedKeyFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func validRunTrustedKeyPEM(t *testing.T, publicKey ed25519.PublicKey) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: marshalRunTrustedPKIX(t, publicKey),
	})
}

func marshalRunTrustedPKIX(t *testing.T, key any) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}
