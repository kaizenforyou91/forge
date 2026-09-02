package compiler

import (
	"errors"
	"testing"
)

func TestValidateKeyID(t *testing.T) {
	valid := map[string]string{
		"ASCII":                "team-key",
		"punctuation":          "forge-dev/key+1",
		"embedded space":       "team key",
		"Unicode":              "kunci-produksi-é",
		"composed Unicode":     "é",
		"decomposed Unicode":   "e\u0301",
		"embedded nonbreaking": "team\u00a0key",
	}
	for name, keyID := range valid {
		t.Run("valid "+name, func(t *testing.T) {
			if err := ValidateKeyID(keyID); err != nil {
				t.Fatalf("ValidateKeyID(%q): %v", keyID, err)
			}
		})
	}

	invalid := map[string]string{
		"empty":                      "",
		"whitespace only":            " \t\r\n ",
		"leading space":              " team",
		"trailing space":             "team ",
		"leading nonbreaking space":  "\u00a0team",
		"trailing nonbreaking space": "team\u00a0",
		"NUL":                        "team\x00key",
		"TAB":                        "team\x09key",
		"LF":                         "team\x0akey",
		"CR":                         "team\x0dkey",
		"ESC":                        "team\x1bkey",
		"US":                         "team\x1fkey",
		"DEL":                        "team\x7fkey",
		"invalid UTF-8":              string([]byte{'t', 0xff, 'm'}),
	}
	for name, keyID := range invalid {
		t.Run("invalid "+name, func(t *testing.T) {
			err := ValidateKeyID(keyID)
			if !errors.Is(err, ErrInvalidKeyID) {
				t.Fatalf("expected ErrInvalidKeyID, got %v", err)
			}
		})
	}
}
