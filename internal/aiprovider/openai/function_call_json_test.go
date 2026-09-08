package openai

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func fcWithExtra(raw string) []byte {
	base := encoded(fcResponse(fcItem("{}")))
	return []byte(strings.TrimSuffix(base, "}") + `,"extra":` + raw + "}")
}

func fcObjectMembers(count int) string {
	members := make([]string, count)
	for i := range members {
		members[i] = fmt.Sprintf(`"k%d":null`, i)
	}
	return "{" + strings.Join(members, ",") + "}"
}

func fcNullArray(count int) string {
	return "[" + strings.TrimSuffix(strings.Repeat("null,", count), ",") + "]"
}

// The complete fixture has 14 values before adding extra. Extra contributes
// one array and eight child arrays; the remaining values are null leaves.
// Keys are excluded. Each child has <=1024 leaves, isolating the total cap.
func fcTotalValues(count int) []byte {
	remaining := count - 23
	groups := make([]string, 8)
	for i := range groups {
		n := remaining
		if n > 1024 {
			n = 1024
		}
		groups[i] = fcNullArray(n)
		remaining -= n
	}
	if remaining != 0 {
		panic("invalid bounded test fixture")
	}
	return fcWithExtra("[" + strings.Join(groups, ",") + "]")
}

func TestFunctionCallJSONEnvelope(t *testing.T) {
	catalog := fcCatalog(t)
	valid := fcBytes(fcResponse(fcItem("{}")))
	for _, data := range [][]byte{valid, append(append([]byte(" \r\n\t"), valid...), []byte(" \r\n\t")...)} {
		fcAssertDecision(t, data, catalog, tool.Allowed)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil}, {"whitespace", []byte(" \n\t")},
		{"root array", []byte("[]")}, {"root scalar", []byte("1")},
		{"root string", []byte(`"text"`)}, {"root boolean", []byte("true")}, {"root null", []byte("null")},
		{"unfinished", []byte(`{"status":`)}, {"bad escape", fcWithExtra(`"\q"`)},
		{"bad hex", fcWithExtra(`"\uGGGG"`)}, {"short escape", fcWithExtra(`"\u123"`)},
		{"trailing value", append(bytes.Clone(valid), []byte(" {}")...)},
		{"trailing garbage", append(bytes.Clone(valid), '!')},
		{"comment", fcWithExtra("/* comment */null")},
		{"line comment", fcWithExtra("// comment\nnull")},
		{"trailing comma", fcWithExtra("[1,]")},
		{"object trailing comma", fcWithExtra(`{"a":1,}`)},
		{"raw UTF8", fcWithExtra("\"\xff\"")},
		{"BOM", append([]byte{0xef, 0xbb, 0xbf}, valid...)},
		{"BOM after whitespace", append([]byte{' ', 0xef, 0xbb, 0xbf}, valid...)},
		{"leading zero", fcWithExtra("01")}, {"NaN", fcWithExtra("NaN")},
		{"plus", fcWithExtra("+1")}, {"bad exponent", fcWithExtra("1e")},
	} {
		t.Run(tc.name, func(t *testing.T) { fcAssertError(t, tc.data, catalog, ai.ErrMalformedResponse) })
	}
	// Metadata is structurally checked but is not a semantic source of authority.
	for _, raw := range []string{
		`{"text":"metadata","nested":[{},null,true]}`,
		"1e99999999999999999999999999999999999999999999999999999",
		encoded(strings.Repeat("x", 65537)),
		`{"é":1,"e\u0301":2}`,
	} {
		fcAssertDecision(t, fcWithExtra(raw), catalog, tool.Allowed)
	}
	// Escaped member identity is decoded exactly, without case folding.
	escaped := strings.Replace(string(valid), `"call_id":`, `"call_\u0069d":`, 1)
	fcAssertDecision(t, []byte(escaped), catalog, tool.Allowed)
}

func TestFunctionCallJSONDuplicates(t *testing.T) {
	valid := string(fcBytes(fcResponse(fcItem("{}"))))
	for _, tc := range []struct{ name, data string }{
		{"root", strings.TrimSuffix(valid, "}") + `,"status":"completed"}`},
		{"root escaped", strings.TrimSuffix(valid, "}") + `,"sta\u0074us":"completed"}`},
		{"item", strings.Replace(valid, `"call_id":`, `"call_\u0069d":"duplicate","call_id":`, 1)},
		{"nested metadata", string(fcWithExtra(`{"a":{"text":1,"te\u0078t":2}}`))},
		{"object in array", string(fcWithExtra(`[{"x":1,"x":2}]`))},
		{"surrogate pair key", string(fcWithExtra(`{"\uD83D\uDE00":1,"😀":2}`))},
		{"caller", strings.Replace(valid, `"type":"function_call"`, `"caller":{"type":"direct","type":"direct"},"type":"function_call"`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fcAssertError(t, []byte(tc.data), fcCatalog(t), ai.ErrMalformedResponse)
		})
	}
	// Duplicate JSON inside logical arguments belongs to Stage A, not this layer.
	d := fcAssertDecision(t, fcBytes(fcResponse(fcItem(`{"v":[{"x":1,"x":2}]}`))),
		fcCatalog(t, tool.Parameter{Name: "v", Type: tool.Array}), tool.InvalidArguments)
	if d.Admission().Reason() != tool.InvalidArguments {
		t.Fatal("inner duplicate reclassified")
	}
}

func TestFunctionCallJSONResourceBounds(t *testing.T) {
	valid := fcBytes(fcResponse(fcItem("{}")))
	atBody := append(bytes.Clone(valid), bytes.Repeat([]byte{' '}, maxBody-len(valid))...)
	for _, tc := range []struct {
		name     string
		at, over []byte
	}{
		{"body", atBody, append(bytes.Clone(atBody), ' ')},
		{"array depth", fcWithExtra(strings.Repeat("[", 31) + "0" + strings.Repeat("]", 31)),
			fcWithExtra(strings.Repeat("[", 32) + "0" + strings.Repeat("]", 32))},
		{"object depth", fcWithExtra(strings.Repeat(`{"x":`, 30) + "{}" + strings.Repeat("}", 30)),
			fcWithExtra(strings.Repeat(`{"x":`, 31) + "{}" + strings.Repeat("}", 31))},
		{"object members", fcWithExtra(fcObjectMembers(128)), fcWithExtra(fcObjectMembers(129))},
		{"array elements", fcWithExtra(fcNullArray(1024)), fcWithExtra(fcNullArray(1025))},
		{"total values", fcTotalValues(8192), fcTotalValues(8193)},
		{"decoded key", fcWithExtra("{" + encoded(strings.Repeat("é", 128)) + ":null}"),
			fcWithExtra("{" + encoded(strings.Repeat("é", 128)+"x") + ":null}")},
		{"escaped key", fcWithExtra(`{"` + strings.Repeat(`\u0061`, 256) + `":null}`),
			fcWithExtra(`{"` + strings.Repeat(`\u0061`, 257) + `":null}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fcAssertDecision(t, tc.at, fcCatalog(t), tool.Allowed)
			fcAssertError(t, tc.over, fcCatalog(t), ai.ErrResponseTooLarge)
		})
	}
	// Overflow inside otherwise ignored metadata is never skipped for status.
	tooDeep := fcWithExtra(strings.Repeat("[", 32) + "0" + strings.Repeat("]", 32))
	fcAssertError(t, append(tooDeep, '!'), fcCatalog(t), ai.ErrMalformedResponse)
}

func TestFunctionCallOuterUnicodeEveryLocation(t *testing.T) {
	const marker = "OUTER_UNICODE_MARKER"
	for _, position := range []string{"key", "id", "call_id", "name", "arguments", "error", "metadata"} {
		for _, bad := range []string{
			`\uD800`, `\uDC00`, `\uD800\u0041`, `\uD800\uD801`,
			`\uD83Dtext\uDE00`, `\uD83D\uDE00\uDC00`, "\xff",
		} {
			t.Run(fmt.Sprintf("%s/%x", position, []byte(bad)), func(t *testing.T) {
				item := fcItem("{}")
				obj := fcResponse(item)
				switch position {
				case "key":
					obj[marker] = fcMetadataCanary
				case "error":
					obj["error"] = map[string]any{"message": marker + fcErrorCanary}
				case "metadata":
					obj["metadata"] = map[string]any{"nested": []any{marker + fcMetadataCanary}}
				default:
					item[position] = marker
				}
				data := []byte(strings.Replace(encoded(obj), marker, bad, 1))
				fcAssertError(t, data, fcCatalog(t), ai.ErrMalformedResponse)
			})
		}
	}
	for _, good := range []string{`"\uD83D\uDE00"`, `"\ud83d\ude00"`, `"�"`, `"\uFFFD"`,
		`"\\uD800"`, `"\\\"\\uDC00"`, `"\"\\\/\b\f\n\r\t"`} {
		fcAssertDecision(t, fcWithExtra(good), fcCatalog(t), tool.Allowed)
	}
}

func TestFunctionCallTwoLayerArguments(t *testing.T) {
	catalog := fcCatalog(t, tool.Parameter{Name: "text", Type: tool.String})
	for _, tc := range []struct {
		name, logical string
		want          tool.Reason
	}{
		{"inner high surrogate", `{"text":"\uD800"}`, tool.InvalidArguments},
		{"inner low surrogate", `{"text":"\uDC00"}`, tool.InvalidArguments},
		{"inner pair", `{"text":"\uD83D\uDE00"}`, tool.Allowed},
		{"inner escaped backslash", `{"text":"\\uD800"}`, tool.Allowed},
		{"inner replacement", `{"text":"�"}`, tool.Allowed},
		{"inner escaped key", `{"te\u0078t":"\uFFFD"}`, tool.Allowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Encoding the logical string quotes its backslashes for the outer
			// document. They must not be mistaken for outer surrogate escapes.
			body := fcBytes(fcResponse(fcItem(tc.logical)))
			d := fcAssertDecision(t, body, catalog, tc.want)
			if tc.want == tool.Allowed {
				call, ok := d.Admission().AdmittedCall()
				if !ok || string(call.Arguments) != tc.logical {
					t.Fatal("logical bytes changed")
				}
			}
		})
	}
	// Actual outer escapes decode once into the logical argument string.
	base := encoded(fcResponse(fcItem(`{"text":"OUTER_TOKEN"}`)))
	body := []byte(strings.Replace(base, "OUTER_TOKEN", `\uD83D\uDE00`, 1))
	d := fcAssertDecision(t, body, catalog, tool.Allowed)
	call, _ := d.Admission().AdmittedCall()
	if string(call.Arguments) != `{"text":"😀"}` {
		t.Fatal("outer string was not decoded exactly once")
	}
}
