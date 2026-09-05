package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

const maxRunTrustedKeyFileBytes int64 = 16 * 1024

func loadRunTrustedPublicKey(path string) (ed25519.PublicKey, error) {
	return loadTrustedPublicKey(path)
}

func loadTrustedPublicKey(path string) (ed25519.PublicKey, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, runTrustedKeyError("trusted public-key path is required", nil)
	}
	if trimmedPath != path {
		return nil, runTrustedKeyError("trusted public-key path must not contain surrounding whitespace", nil)
	}

	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, runTrustedKeyError("inspect trusted public-key path", err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, runTrustedKeyError("trusted public-key path is a symbolic link", nil)
	}
	if !pathInfo.Mode().IsRegular() {
		return nil, runTrustedKeyError("trusted public-key path is not a regular file", nil)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, runTrustedKeyError("open trusted public-key file", err)
	}

	openInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, runTrustedKeyError("stat open trusted public-key file", err)
	}
	if !openInfo.Mode().IsRegular() {
		_ = file.Close()
		return nil, runTrustedKeyError("trusted public-key path is not a regular file", nil)
	}
	if !os.SameFile(pathInfo, openInfo) {
		_ = file.Close()
		return nil, runTrustedKeyError("trusted public-key path identity changed while opening", nil)
	}
	if openInfo.Size() > maxRunTrustedKeyFileBytes {
		_ = file.Close()
		return nil, runTrustedKeyError("trusted public-key file exceeds 16 KiB limit", nil)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxRunTrustedKeyFileBytes+1))
	closeErr := file.Close()
	if err != nil {
		return nil, runTrustedKeyError("read trusted public-key file", err)
	}
	if closeErr != nil {
		return nil, runTrustedKeyError("close trusted public-key file", closeErr)
	}
	if int64(len(data)) > maxRunTrustedKeyFileBytes {
		return nil, runTrustedKeyError("trusted public-key file exceeds 16 KiB limit", nil)
	}

	block, trailing := pem.Decode(data)
	if block == nil {
		return nil, runTrustedKeyError("trusted public-key file is not PEM", nil)
	}
	if block.Type != "PUBLIC KEY" {
		return nil, runTrustedKeyError("trusted public-key PEM type must be PUBLIC KEY", nil)
	}
	if len(block.Headers) != 0 {
		return nil, runTrustedKeyError("trusted public-key PEM headers are not allowed", nil)
	}
	if len(bytes.TrimSpace(trailing)) != 0 {
		return nil, runTrustedKeyError("trusted public-key PEM has unexpected trailing data", nil)
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, runTrustedKeyError("parse Ed25519 PKIX public key", err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, runTrustedKeyError("PKIX public key is not Ed25519", nil)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, runTrustedKeyError("Ed25519 public key has invalid length", nil)
	}

	return append(ed25519.PublicKey(nil), publicKey...), nil
}

func validateRunTrustKeyID(keyID string) (string, error) {
	return validateTrustedKeyID(keyID)
}

func validateTrustedKeyID(keyID string) (string, error) {
	if err := compiler.ValidateKeyID(keyID); err != nil {
		return "", runTrustedKeyError("invalid trusted key ID", err)
	}

	return keyID, nil
}

func runTrustedKeyError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidTrustKey, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidTrustKey, message, cause)
}
