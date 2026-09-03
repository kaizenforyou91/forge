package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

type manifestValueKind uint8

const (
	manifestString manifestValueKind = iota
	manifestNullableObject
	manifestObjectArray
)

type manifestField struct {
	kind   manifestValueKind
	object map[string]manifestField
}

var dependencyManifestFields = map[string]manifestField{
	"name":    {kind: manifestString},
	"version": {kind: manifestString},
}

var moduleManifestFields = map[string]manifestField{
	"name":         {kind: manifestString},
	"version":      {kind: manifestString},
	"import_path":  {kind: manifestString},
	"dependencies": {kind: manifestObjectArray, object: dependencyManifestFields},
}

var entrypointManifestFields = map[string]manifestField{
	"module":  {kind: manifestString},
	"version": {kind: manifestString},
}

var rootManifestFields = map[string]manifestField{
	"version":    {kind: manifestString},
	"name":       {kind: manifestString},
	"entrypoint": {kind: manifestNullableObject, object: entrypointManifestFields},
	"modules":    {kind: manifestObjectArray, object: moduleManifestFields},
}

func decodeStrictManifestJSON(data []byte, destination *Manifest) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON manifest is not valid UTF-8")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("JSON manifest must not begin with a UTF-8 BOM")
	}

	if err := inspectManifestJSON(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}

	return nil
}

func inspectManifestJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return fmt.Errorf("top-level JSON value must be an object")
	}

	if err := inspectManifestJSONObject(decoder, rootManifestFields, "manifest"); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON token %v", token)
		}
		return err
	}

	return nil
}

func inspectManifestJSONObject(
	decoder *json.Decoder,
	fields map[string]manifestField,
	path string,
) error {
	seen := make(map[string]struct{}, len(fields))
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("%s object key must be a string", path)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate object key %q at %s", key, path)
		}
		seen[key] = struct{}{}

		field, exists := fields[key]
		if !exists {
			return fmt.Errorf("unknown field %q at %s", key, path)
		}
		if err := inspectManifestJSONField(decoder, field, path+"."+key); err != nil {
			return err
		}
	}

	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if closing != json.Delim('}') {
		return fmt.Errorf("object at %s is not closed", path)
	}

	return nil
}

func inspectManifestJSONField(
	decoder *json.Decoder,
	field manifestField,
	path string,
) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}

	switch field.kind {
	case manifestString:
		if _, ok := token.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
		return nil

	case manifestNullableObject:
		if token == nil {
			return nil
		}
		delimiter, ok := token.(json.Delim)
		if !ok || delimiter != '{' {
			return fmt.Errorf("%s must be an object", path)
		}
		return inspectManifestJSONObject(decoder, field.object, path)

	case manifestObjectArray:
		delimiter, ok := token.(json.Delim)
		if !ok || delimiter != '[' {
			return fmt.Errorf("%s must be an array", path)
		}
		index := 0
		for decoder.More() {
			item, err := decoder.Token()
			if err != nil {
				return err
			}
			itemDelimiter, ok := item.(json.Delim)
			if !ok || itemDelimiter != '{' {
				return fmt.Errorf("%s[%d] must be an object", path, index)
			}
			if err := inspectManifestJSONObject(
				decoder,
				field.object,
				fmt.Sprintf("%s[%d]", path, index),
			); err != nil {
				return err
			}
			index++
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array at %s is not closed", path)
		}
		return nil

	default:
		return fmt.Errorf("unsupported JSON manifest field at %s", path)
	}
}
