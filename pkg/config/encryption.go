package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
)

// Encrypt encrypts plaintext using Forge encryption.
func Encrypt(data []byte, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(
		nonce,
		nonce,
		data,
		nil,
	)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext.
func Decrypt(data []byte, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	return gcm.Open(
		nil,
		nonce,
		ciphertext,
		nil,
	)
}
func EncryptString(text string) ([]byte, error) {

	key, err := LoadKey()
	if err != nil {
		return nil, err
	}

	return Encrypt([]byte(text), key)
}

func DecryptString(data []byte) (string, error) {

	key, err := LoadKey()
	if err != nil {
		return "", err
	}

	out, err := Decrypt(data, key)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
func EncryptConfig(cfg *Config) error {

	if cfg.Secrets.APIKey != "" &&
		!strings.HasPrefix(cfg.Secrets.APIKey, "ENC:") {

		enc, err := EncryptString(cfg.Secrets.APIKey)
		if err != nil {
			return err
		}

		cfg.Secrets.APIKey =
			"ENC:" + base64.StdEncoding.EncodeToString(enc)
	}

	if cfg.Secrets.Token != "" &&
		!strings.HasPrefix(cfg.Secrets.Token, "ENC:") {

		enc, err := EncryptString(cfg.Secrets.Token)
		if err != nil {
			return err
		}

		cfg.Secrets.Token =
			"ENC:" + base64.StdEncoding.EncodeToString(enc)
	}

	return nil
}
func DecryptConfig(cfg *Config) error {

	if strings.HasPrefix(cfg.Secrets.APIKey, "ENC:") {

		raw, err := base64.StdEncoding.DecodeString(
			cfg.Secrets.APIKey[4:],
		)

		if err != nil {
			return err
		}

		plain, err := DecryptString(raw)
		if err != nil {
			return err
		}

		cfg.Secrets.APIKey = plain
	}

	if strings.HasPrefix(cfg.Secrets.Token, "ENC:") {

		raw, err := base64.StdEncoding.DecodeString(
			cfg.Secrets.Token[4:],
		)

		if err != nil {
			return err
		}

		plain, err := DecryptString(raw)
		if err != nil {
			return err
		}

		cfg.Secrets.Token = plain
	}

	return nil
}
