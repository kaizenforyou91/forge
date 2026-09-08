package tool

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// json.Valid supplies ordinary JSON lexical/syntax validation, but intentionally
// tolerates malformed Unicode that encoding/json can repair during decoding.
// Check raw UTF-8 and string surrogate escapes before using decoded tokens.
// The caller has already bounded and copied the input.
func validJSONLexically(data []byte) bool {
	if !utf8.Valid(data) || bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || !json.Valid(data) {
		return false
	}
	return validSurrogates(data)
}

// validSurrogates operates only on already syntax-valid JSON. It distinguishes
// escaped backslashes from actual Unicode escapes and never normalizes text.
func validSurrogates(data []byte) bool {
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
			code := hexCode(data[i+1 : i+5])
			i += 4
			switch {
			case code >= 0xd800 && code <= 0xdbff:
				if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
					return false
				}
				low := hexCode(data[i+3 : i+7])
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

// JSON syntax validation guarantees exactly four hexadecimal digits here.
func hexCode(digits []byte) uint16 {
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

type argumentParser struct {
	decoder *json.Decoder
	values  int
}

// inspectArguments retains only the bounded root name -> observed type index.
// Nested containers are traversed and discarded. Reasons keep limits distinct
// from malformed data without retaining or exposing decoder errors.
func inspectArguments(data []byte) (map[string]ValueType, Reason) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, InvalidArguments
	}
	parser := argumentParser{decoder: decoder, values: 1}
	root := make(map[string]ValueType)
	if reason := parser.object(1, root); reason != Allowed {
		return nil, reason
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, InvalidArguments
	}
	return root, Allowed
}

func (p *argumentParser) object(depth int, root map[string]ValueType) Reason {
	seen := make(map[string]struct{})
	members := 0
	for p.decoder.More() {
		if members == maxObjectMembers {
			return PolicyRejected
		}
		members++
		token, err := p.decoder.Token()
		if err != nil {
			return InvalidArguments
		}
		key, ok := token.(string)
		if !ok {
			return InvalidArguments
		}
		if len(key) > maxKeyBytes {
			return PolicyRejected
		}
		if _, duplicate := seen[key]; duplicate {
			return InvalidArguments
		}
		seen[key] = struct{}{}
		kind, reason := p.value(depth)
		if reason != Allowed {
			return reason
		}
		if root != nil {
			root[key] = kind
		}
	}
	if token, err := p.decoder.Token(); err != nil || token != json.Delim('}') {
		return InvalidArguments
	}
	return Allowed
}

func (p *argumentParser) array(depth int) Reason {
	elements := 0
	for p.decoder.More() {
		if elements == maxArrayElements {
			return PolicyRejected
		}
		elements++
		if _, reason := p.value(depth); reason != Allowed {
			return reason
		}
	}
	if token, err := p.decoder.Token(); err != nil || token != json.Delim(']') {
		return InvalidArguments
	}
	return Allowed
}

func (p *argumentParser) value(parentDepth int) (ValueType, Reason) {
	if p.values == maxValues {
		return InvalidType, PolicyRejected
	}
	p.values++
	token, err := p.decoder.Token()
	if err != nil {
		return InvalidType, InvalidArguments
	}
	switch value := token.(type) {
	case string:
		if len(value) > maxStringBytes {
			return InvalidType, PolicyRejected
		}
		return String, Allowed
	case json.Number:
		if len(value) > maxNumberBytes {
			return InvalidType, PolicyRejected
		}
		return Number, Allowed
	case bool:
		return Boolean, Allowed
	case nil:
		return Null, Allowed
	case json.Delim:
		if parentDepth == maxDepth {
			return InvalidType, PolicyRejected
		}
		switch value {
		case '{':
			return Object, p.object(parentDepth+1, nil)
		case '[':
			return Array, p.array(parentDepth + 1)
		}
	}
	return InvalidType, InvalidArguments
}
