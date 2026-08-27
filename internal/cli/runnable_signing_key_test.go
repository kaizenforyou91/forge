package cli

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestRunnableSigningKeyLoadsExactEd25519PKCS8PEM(t *testing.T) {
	_, original, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := writeRunnableSigningKeyPEM(t, "PRIVATE KEY", marshalPKCS8ForTest(t, original), nil)

	loaded, err := loadRunnableSigningPrivateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded, original) {
		t.Fatal("loaded private key does not equal generated key")
	}
	if &loaded[0] == &original[0] {
		t.Fatal("loaded private key aliases generated key")
	}

	payload := []byte("forge runnable signing key test")
	if !ed25519.Verify(original.Public().(ed25519.PublicKey), payload, ed25519.Sign(loaded, payload)) {
		t.Fatal("loaded private key cannot produce a valid signature")
	}
}

func TestRunnableSigningKeyAcceptsTrailingWhitespace(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data := validRunnableSigningKeyPEM(t, privateKey)
	data = append(data, []byte("\r\n \t\n")...)
	path := writeRunnableSigningKeyFile(t, "trailing-whitespace.pem", data, 0o600)

	if _, err := loadRunnableSigningPrivateKey(path); err != nil {
		t.Fatalf("expected trailing whitespace to be accepted: %v", err)
	}
}

func TestRunnableSigningKeyRejectsInvalidPathsAndTypes(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(""))
	})

	t.Run("nonexistent file", func(t *testing.T) {
		requireRunnableSigningKeyError(
			t,
			loadRunnableSigningPrivateKeyError(filepath.Join(t.TempDir(), "missing.pem")),
		)
	})

	t.Run("directory", func(t *testing.T) {
		requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(t.TempDir()))
	})

	t.Run("symbolic link", func(t *testing.T) {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		target := writeRunnableSigningKeyFile(
			t,
			"target.pem",
			validRunnableSigningKeyPEM(t, privateKey),
			0o600,
		)
		link := filepath.Join(filepath.Dir(target), "key-link.pem")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("Windows symlink creation is unavailable: %v", err)
			}
			t.Fatal(err)
		}
		requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(link))
	})

	t.Run("oversized file", func(t *testing.T) {
		path := writeRunnableSigningKeyFile(
			t,
			"oversized.pem",
			bytes.Repeat([]byte{'x'}, int(maxRunnableSigningKeyFileBytes)+1),
			0o600,
		)
		requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(path))
	})
}

func TestRunnableSigningKeyRejectsAlternateOrMalformedFormats(t *testing.T) {
	_, ed25519Key, err := ed25519.GenerateKey(rand.Reader)
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
	ecdsaSEC1, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		t.Fatal(err)
	}
	validDER := marshalPKCS8ForTest(t, ed25519Key)
	invalidLengthDER := marshalInvalidLengthEd25519PKCS8ForTest(t)

	tests := map[string][]byte{
		"raw private key":    append([]byte(nil), ed25519Key...),
		"base64 private key": []byte(base64.StdEncoding.EncodeToString(ed25519Key)),
		"malformed PEM":      []byte("-----BEGIN PRIVATE KEY-----\nnot-base64\n-----END PRIVATE KEY-----\n"),
		"malformed DER": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: []byte("not-pkcs8"),
		}),
		"incorrect Ed25519 key length": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: invalidLengthDER,
		}),
		"wrong PEM label": pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PRIVATE KEY",
			Bytes: validDER,
		}),
		"PKCS1": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		}),
		"SEC1": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: ecdsaSEC1,
		}),
		"RSA PKCS8": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: marshalPKCS8ForTest(t, rsaKey),
		}),
		"ECDSA PKCS8": pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: marshalPKCS8ForTest(t, ecdsaKey),
		}),
		"encrypted PEM headers": pem.EncodeToMemory(&pem.Block{
			Type: "PRIVATE KEY",
			Headers: map[string]string{
				"Proc-Type": "4,ENCRYPTED",
				"DEK-Info":  "AES-256-CBC,00000000000000000000000000000000",
			},
			Bytes: validDER,
		}),
		"encrypted PKCS8 PEM": pem.EncodeToMemory(&pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: []byte("encrypted-pkcs8-placeholder"),
		}),
		"multiple PEM blocks": append(
			validRunnableSigningKeyPEM(t, ed25519Key),
			validRunnableSigningKeyPEM(t, ed25519Key)...,
		),
		"trailing non-whitespace": append(
			validRunnableSigningKeyPEM(t, ed25519Key),
			[]byte("unexpected")...,
		),
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeRunnableSigningKeyFile(t, "invalid.pem", data, 0o600)
			requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(path))
		})
	}
}

func TestRunnableSigningKeyUnixPermissionPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are outside the Unix permission-bit contract")
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data := validRunnableSigningKeyPEM(t, privateKey)

	for _, mode := range []os.FileMode{0o644, 0o660} {
		t.Run(mode.String(), func(t *testing.T) {
			path := writeRunnableSigningKeyFile(t, "insecure.pem", data, mode)
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			requireRunnableSigningKeyError(t, loadRunnableSigningPrivateKeyError(path))
		})
	}

	path := writeRunnableSigningKeyFile(t, "secure.pem", data, 0o600)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRunnableSigningPrivateKey(path); err != nil {
		t.Fatalf("expected mode 0600 to be accepted: %v", err)
	}
}

func TestRunnableSigningKeyIDValidation(t *testing.T) {
	for name, keyID := range map[string]string{
		"blank":               "",
		"whitespace only":     " \t\r\n ",
		"leading whitespace":  " forge-dev",
		"trailing whitespace": "forge-dev ",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := validateRunnableSigningKeyID(keyID)
			if got != "" {
				t.Fatalf("expected empty rejected key ID, got %q", got)
			}
			requireRunnableSigningKeyError(t, err)
		})
	}

	const keyID = "forge-dev/key+1"
	got, err := validateRunnableSigningKeyID(keyID)
	if err != nil {
		t.Fatal(err)
	}
	if got != keyID {
		t.Fatalf("expected exact key ID %q, got %q", keyID, got)
	}
}

func loadRunnableSigningPrivateKeyError(path string) error {
	_, err := loadRunnableSigningPrivateKey(path)
	return err
}

func requireRunnableSigningKeyError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidPackageSignature) {
		t.Fatalf("expected ErrInvalidPackageSignature, got %v", err)
	}
}

func writeRunnableSigningKeyPEM(
	t *testing.T,
	blockType string,
	der []byte,
	headers map[string]string,
) string {
	t.Helper()
	return writeRunnableSigningKeyFile(
		t,
		"signing-key.pem",
		pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der, Headers: headers}),
		0o600,
	)
}

func writeRunnableSigningKeyFile(
	t *testing.T,
	name string,
	data []byte,
	mode os.FileMode,
) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func validRunnableSigningKeyPEM(t *testing.T, privateKey ed25519.PrivateKey) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalPKCS8ForTest(t, privateKey),
	})
}

func marshalPKCS8ForTest(t *testing.T, key any) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func marshalInvalidLengthEd25519PKCS8ForTest(t *testing.T) []byte {
	t.Helper()
	privateKey, err := asn1.Marshal([]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	der, err := asn1.Marshal(struct {
		Version    int
		Algorithm  pkix.AlgorithmIdentifier
		PrivateKey []byte
	}{
		Version: 0,
		Algorithm: pkix.AlgorithmIdentifier{
			Algorithm: asn1.ObjectIdentifier{1, 3, 101, 112},
		},
		PrivateKey: privateKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return der
}
