package openai

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai"
)

const (
	functionCallMaxDepth    = 32
	functionCallMaxMembers  = 128
	functionCallMaxElements = 1024
	functionCallMaxValues   = 8192
	functionCallMaxKeyBytes = 256
)

// validateFunctionCallJSON operates on an already byte-bounded owned snapshot.
// It validates all outer strings before decoding tokens, including strings in
// ignored metadata. It never parses the logical JSON inside arguments strings.
// Syntax/Unicode precede bounded traversal, so malformed lexical input cannot
// be repaired into an otherwise valid proposal.
func validateFunctionCallJSON(data []byte) error {
	if !utf8.Valid(data) || bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || !json.Valid(data) {
		return ai.ErrMalformedResponse
	}
	if !functionCallSurrogatesValid(data) {
		return ai.ErrMalformedResponse
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return ai.ErrMalformedResponse
	}
	parser := functionCallJSONParser{decoder: decoder, values: 1}
	if err := parser.object(1); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ai.ErrMalformedResponse
	}
	return nil
}

// functionCallSurrogatesValid scans only syntax-valid outer JSON. Escaped
// backslashes are consumed as escapes themselves, not mistaken for a following
// logical backslash-u sequence. No decoded string is allowed before this pass.
func functionCallSurrogatesValid(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		for i++; i < len(data) && data[i] != '"'; i++ {
			if data[i] != '\\' {
				continue
			}
			i++
			if data[i] != 'u' {
				continue
			}
			code := functionCallHexCode(data[i+1 : i+5])
			i += 4
			switch {
			case code >= 0xd800 && code <= 0xdbff:
				if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
					return false
				}
				low := functionCallHexCode(data[i+3 : i+7])
				if low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			case code >= 0xdc00 && code <= 0xdfff:
				return false
			}
		}
	}
	return true
}

// Syntax validation guarantees four hexadecimal digits.
func functionCallHexCode(digits []byte) uint16 {
	var value uint16
	for _, c := range digits {
		value <<= 4
		switch {
		case c >= '0' && c <= '9':
			value |= uint16(c - '0')
		case c >= 'a' && c <= 'f':
			value |= uint16(c - 'a' + 10)
		default:
			value |= uint16(c - 'A' + 10)
		}
	}
	return value
}

// Only a bounded per-object key set and call-local counters are retained.
// Containers/scalars count as values; object keys do not. Nested values are
// traversed and discarded; no arbitrary decoded document tree is constructed.
type functionCallJSONParser struct {
	decoder *json.Decoder
	values  int
}

func (p *functionCallJSONParser) object(depth int) error {
	seen := make(map[string]struct{})
	for p.decoder.More() {
		if len(seen) == functionCallMaxMembers {
			return ai.ErrResponseTooLarge
		}
		token, err := p.decoder.Token()
		if err != nil {
			return ai.ErrMalformedResponse
		}
		key, ok := token.(string)
		if !ok {
			return ai.ErrMalformedResponse
		}
		if len(key) > functionCallMaxKeyBytes {
			return ai.ErrResponseTooLarge
		}
		if _, duplicate := seen[key]; duplicate {
			return ai.ErrMalformedResponse
		}
		seen[key] = struct{}{}
		if err := p.value(depth); err != nil {
			return err
		}
	}
	if end, err := p.decoder.Token(); err != nil || end != json.Delim('}') {
		return ai.ErrMalformedResponse
	}
	return nil
}

func (p *functionCallJSONParser) array(depth int) error {
	elements := 0
	for p.decoder.More() {
		if elements == functionCallMaxElements {
			return ai.ErrResponseTooLarge
		}
		elements++
		if err := p.value(depth); err != nil {
			return err
		}
	}
	if end, err := p.decoder.Token(); err != nil || end != json.Delim(']') {
		return ai.ErrMalformedResponse
	}
	return nil
}

func (p *functionCallJSONParser) value(parentDepth int) error {
	if p.values == functionCallMaxValues {
		return ai.ErrResponseTooLarge
	}
	p.values++
	token, err := p.decoder.Token()
	if err != nil {
		return ai.ErrMalformedResponse
	}
	switch value := token.(type) {
	case nil, bool, string, json.Number:
		// All token memory is bounded by the original 1 MiB body.
		return nil
	case json.Delim:
		if parentDepth == functionCallMaxDepth {
			return ai.ErrResponseTooLarge
		}
		switch value {
		case '{':
			return p.object(parentDepth + 1)
		case '[':
			return p.array(parentDepth + 1)
		}
	}
	return ai.ErrMalformedResponse
}
