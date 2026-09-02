package compiler

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidateKeyID validates an exact Forge signing-key identifier.
func ValidateKeyID(keyID string) error {
	if !utf8.ValidString(keyID) {
		return fmt.Errorf("%w: identifier is not valid UTF-8", ErrInvalidKeyID)
	}
	if keyID == "" {
		return fmt.Errorf("%w: identifier is empty", ErrInvalidKeyID)
	}
	if strings.TrimSpace(keyID) != keyID {
		return fmt.Errorf("%w: identifier has surrounding whitespace", ErrInvalidKeyID)
	}
	for _, character := range keyID {
		if character <= '\x1f' || character == '\x7f' {
			return fmt.Errorf("%w: identifier contains an ASCII control character", ErrInvalidKeyID)
		}
	}

	return nil
}
