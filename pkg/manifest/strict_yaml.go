package manifest

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

func decodeStrictManifestYAML(data []byte, destination *Manifest) error {
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) ||
		bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		return fmt.Errorf("YAML manifest must be UTF-8, not UTF-16")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("YAML manifest is not valid UTF-8")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("YAML manifest must not begin with a UTF-8 BOM")
	}
	if len(data) == 0 {
		return fmt.Errorf("YAML manifest is empty")
	}
	if hasYAMLTagDirective(data) {
		return fmt.Errorf("YAML tag directives are not supported")
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if err == io.EOF {
			return fmt.Errorf("YAML manifest is empty")
		}
		return err
	}

	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("YAML manifest must contain exactly one document")
		}
		return err
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return fmt.Errorf("YAML manifest must contain one document value")
	}
	root := document.Content[0]
	if err := rejectUnsupportedYAMLFeatures(root, newYAMLSource(data)); err != nil {
		return err
	}
	if err := inspectManifestYAMLMapping(root, rootManifestFields, "manifest"); err != nil {
		return err
	}

	typedDecoder := yaml.NewDecoder(bytes.NewReader(data))
	typedDecoder.KnownFields(true)
	if err := typedDecoder.Decode(destination); err != nil {
		return err
	}

	return nil
}

// yaml.v3 exposes used tags on nodes, but it does not retain unused %TAG
// directives. Directives are only legal at column zero in the document
// preamble, so inspect that grammar-delimited region rather than searching
// arbitrary document text (which could match comments or scalar content).
func hasYAMLTagDirective(data []byte) bool {
	for len(data) > 0 {
		line := data
		if newline := bytes.IndexByte(data, '\n'); newline >= 0 {
			line = data[:newline]
			data = data[newline+1:]
		} else {
			data = nil
		}
		line = bytes.TrimSuffix(line, []byte{'\r'})
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}

		if line[0] == '%' {
			if bytes.HasPrefix(line, []byte("%TAG")) &&
				(len(line) == len("%TAG") || line[len("%TAG")] == ' ' || line[len("%TAG")] == '\t') {
				return true
			}
			continue
		}

		if yamlDocumentMarkerLine(line, "---") {
			continue
		}

		break
	}

	return false
}

func yamlDocumentMarkerLine(line []byte, marker string) bool {
	if !bytes.HasPrefix(line, []byte(marker)) {
		return false
	}
	rest := bytes.TrimSpace(line[len(marker):])
	return len(rest) == 0 || rest[0] == '#'
}

type yamlSource struct {
	data       []byte
	lineStarts []int
}

func newYAMLSource(data []byte) yamlSource {
	source := yamlSource{
		data:       data,
		lineStarts: []int{0},
	}
	for i, b := range data {
		if b == '\n' && i+1 < len(data) {
			source.lineStarts = append(source.lineStarts, i+1)
		}
	}
	return source
}

func rejectUnsupportedYAMLFeatures(node *yaml.Node, source yamlSource) error {
	if node == nil {
		return fmt.Errorf("YAML manifest contains an empty node")
	}
	if node.Kind == yaml.AliasNode || node.Alias != nil {
		return fmt.Errorf("YAML aliases are not supported")
	}
	if node.Anchor != "" {
		return fmt.Errorf("YAML anchors are not supported")
	}
	if node.Style&yaml.TaggedStyle != 0 || source.nodeStartsWithTag(node) {
		return fmt.Errorf("explicit YAML tags are not supported")
	}

	if node.Kind == yaml.MappingNode {
		if len(node.Content)%2 != 0 {
			return fmt.Errorf("YAML mapping contains an unmatched key")
		}
		seen := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yaml.ScalarNode && key.ShortTag() == "!!merge" {
				return fmt.Errorf("YAML merge keys are not supported")
			}
			if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
				return fmt.Errorf("YAML mapping keys must be strings")
			}
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("duplicate YAML mapping key %q", key.Value)
			}
			seen[key.Value] = struct{}{}
		}
	}

	for _, child := range node.Content {
		if err := rejectUnsupportedYAMLFeatures(child, source); err != nil {
			return err
		}
	}

	return nil
}

func (source yamlSource) nodeStartsWithTag(node *yaml.Node) bool {
	if node.Line < 1 || node.Column < 1 {
		return false
	}
	if node.Line > len(source.lineStarts) {
		return false
	}

	data := source.data[source.lineStarts[node.Line-1]:]
	column := 1
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if column == node.Column {
			return r == '!'
		}
		if r == '\n' {
			return false
		}
		data = data[size:]
		column++
	}

	return false
}

func inspectManifestYAMLMapping(
	node *yaml.Node,
	fields map[string]manifestField,
	path string,
) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%s must be a mapping", path)
	}

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		field, exists := fields[key.Value]
		if !exists {
			return fmt.Errorf("unknown field %q at %s", key.Value, path)
		}
		if err := inspectManifestYAMLField(value, field, path+"."+key.Value); err != nil {
			return err
		}
	}

	return nil
}

func inspectManifestYAMLField(
	node *yaml.Node,
	field manifestField,
	path string,
) error {
	switch field.kind {
	case manifestString:
		if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!str" {
			return fmt.Errorf("%s must be a YAML string", path)
		}
		return nil

	case manifestNullableObject:
		if node.Kind == yaml.ScalarNode && node.ShortTag() == "!!null" {
			return nil
		}
		return inspectManifestYAMLMapping(node, field.object, path)

	case manifestObjectArray:
		if node.Kind != yaml.SequenceNode {
			return fmt.Errorf("%s must be a sequence", path)
		}
		for i, item := range node.Content {
			if err := inspectManifestYAMLMapping(
				item,
				field.object,
				fmt.Sprintf("%s[%d]", path, i),
			); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported YAML manifest field at %s", path)
	}
}
