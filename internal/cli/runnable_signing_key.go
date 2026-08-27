package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

const maxRunnableSigningKeyFileBytes int64 = 16 * 1024

func loadRunnableSigningPrivateKey(path string) (ed25519.PrivateKey, error) {
	if strings.TrimSpace(path) == "" {
		return nil, runnableSigningKeyError("signing key path is required", nil)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, runnableSigningKeyError("open signing key file", err)
	}
	defer file.Close()

	openInfo, err := file.Stat()
	if err != nil {
		return nil, runnableSigningKeyError("stat open signing key file", err)
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, runnableSigningKeyError("inspect signing key path", err)
	}

	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, runnableSigningKeyError("signing key path is a symbolic link", nil)
	}
	if !pathInfo.Mode().IsRegular() || !openInfo.Mode().IsRegular() {
		return nil, runnableSigningKeyError("signing key path is not a regular file", nil)
	}
	if !os.SameFile(openInfo, pathInfo) {
		return nil, runnableSigningKeyError("signing key path identity changed while opening", nil)
	}
	if openInfo.Size() > maxRunnableSigningKeyFileBytes {
		return nil, runnableSigningKeyError("signing key file exceeds 16 KiB limit", nil)
	}
	if runtime.GOOS != "windows" && pathInfo.Mode().Perm()&0o077 != 0 {
		return nil, runnableSigningKeyError("signing key file permissions allow group or other access", nil)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxRunnableSigningKeyFileBytes+1))
	if err != nil {
		return nil, runnableSigningKeyError("read signing key file", err)
	}
	defer clearBytes(data)
	if int64(len(data)) > maxRunnableSigningKeyFileBytes {
		return nil, runnableSigningKeyError("signing key file exceeds 16 KiB limit", nil)
	}

	block, trailing := pem.Decode(data)
	if block == nil {
		return nil, runnableSigningKeyError("signing key file is not PEM", nil)
	}
	defer clearBytes(block.Bytes)

	if block.Type != "PRIVATE KEY" {
		return nil, runnableSigningKeyError("signing key PEM type must be PRIVATE KEY", nil)
	}
	if len(block.Headers) != 0 {
		return nil, runnableSigningKeyError("signing key PEM headers are not allowed", nil)
	}
	if len(bytes.TrimSpace(trailing)) != 0 {
		return nil, runnableSigningKeyError("signing key PEM has unexpected trailing data", nil)
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, runnableSigningKeyError("parse Ed25519 PKCS#8 private key", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, runnableSigningKeyError("PKCS#8 private key is not Ed25519", nil)
	}
	defer clearBytes(privateKey)
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, runnableSigningKeyError("Ed25519 private key has invalid length", nil)
	}

	return append(ed25519.PrivateKey(nil), privateKey...), nil
}

func validateRunnableSigningKeyID(keyID string) (string, error) {
	trimmed := strings.TrimSpace(keyID)
	if trimmed == "" {
		return "", runnableSigningKeyError("signing key ID is required", nil)
	}
	if trimmed != keyID {
		return "", runnableSigningKeyError("signing key ID must not contain surrounding whitespace", nil)
	}

	return keyID, nil
}

func runnableSigningKeyError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidPackageSignature, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidPackageSignature, message, cause)
}

func clearBytes(data []byte) {
	clear(data)
}
