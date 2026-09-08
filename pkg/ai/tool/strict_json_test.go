package tool

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func argumentCatalog(t *testing.T, kind ValueType) *Catalog {
	t.Helper()
	return mustCatalog(t, Definition{Name: "tool", Parameters: []Parameter{{Name: "v", Type: kind, Required: true}}})
}

func TestStrictJSONEnvelope(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		kind ValueType
		want Reason
	}{
		{"empty object", "{}", InvalidType, Allowed},
		{"whitespace", " \t\r\n{} \t\r\n", InvalidType, Allowed},
		{"nested object", "{\"v\":{\"inner\":{\"flag\":true}}}", Object, Allowed},
		{"nested array", "{\"v\":[[null],{\"x\":[]}]}", Array, Allowed},
		{"empty input", "", InvalidType, InvalidArguments},
		{"unfinished", "{\"v\":", Object, InvalidArguments},
		{"array root", "[]", InvalidType, InvalidArguments},
		{"string root", "\"text\"", InvalidType, InvalidArguments},
		{"number root", "1", InvalidType, InvalidArguments},
		{"boolean root", "true", InvalidType, InvalidArguments},
		{"null root", "null", InvalidType, InvalidArguments},
		{"trailing JSON", "{} {}", InvalidType, InvalidArguments},
		{"trailing garbage", "{}x", InvalidType, InvalidArguments},
		{"line comment", "{//comment\n}", InvalidType, InvalidArguments},
		{"block comment", "{/*comment*/}", InvalidType, InvalidArguments},
		{"trailing object comma", "{\"v\":1,}", Number, InvalidArguments},
		{"trailing array comma", "{\"v\":[1,]}", Array, InvalidArguments},
		{"duplicate root", "{\"v\":1,\"v\":2}", Number, InvalidArguments},
		{"duplicate nested", "{\"v\":{\"x\":1,\"x\":2}}", Object, InvalidArguments},
		{"duplicate object in array", "{\"v\":[{\"x\":1,\"x\":2}]}", Array, InvalidArguments},
		{"escaped equivalent root", "{\"v\":\"a\",\"\\u0076\":\"b\"}", String, InvalidArguments},
		{"escaped equivalent nested", "{\"v\":{\"text\":1,\"te\\u0078t\":2}}", Object, InvalidArguments},
		{"distinct normalization", "{\"v\":{\"é\":1,\"e\\u0301\":2}}", Object, Allowed},
		{"leading zero", "{\"v\":01}", Number, InvalidArguments},
		{"plus number", "{\"v\":+1}", Number, InvalidArguments},
		{"NaN", "{\"v\":NaN}", Number, InvalidArguments},
		{"incomplete exponent", "{\"v\":1e}", Number, InvalidArguments},
		{"fraction exponent", "{\"v\":-0.25e+12}", Number, Allowed},
		{"no float conversion", "{\"v\":1e999999}", Number, Allowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := mustCatalog(t, Definition{Name: "tool"})
			if tc.kind != InvalidType {
				c = argumentCatalog(t, tc.kind)
			}
			assertDecision(t, c, Call{Name: "tool", Arguments: []byte(tc.raw)}, tc.want)
		})
	}
}

func TestStrictJSONStringUnicode(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		kind ValueType
		want Reason
	}{
		{"invalid UTF8", "{\"v\":\"\xff\"}", String, InvalidArguments},
		{"UTF8 BOM", "\xef\xbb\xbf{\"v\":\"x\"}", String, InvalidArguments},
		{"BOM after whitespace", " \xef\xbb\xbf{\"v\":\"x\"}", String, InvalidArguments},
		{"bad escape", "{\"v\":\"\\q\"}", String, InvalidArguments},
		{"bad hex", "{\"v\":\"\\uGGGG\"}", String, InvalidArguments},
		{"short escape", "{\"v\":\"\\u123\"}", String, InvalidArguments},
		{"raw newline", "{\"v\":\"a\nb\"}", String, InvalidArguments},
		{"high surrogate", "{\"v\":\"\\uD800\"}", String, InvalidArguments},
		{"low surrogate", "{\"v\":\"\\uDC00\"}", String, InvalidArguments},
		{"high followed by scalar", "{\"v\":\"\\uD800\\u0041\"}", String, InvalidArguments},
		{"high followed by high", "{\"v\":\"\\uD800\\uD801\"}", String, InvalidArguments},
		{"pair followed by low", "{\"v\":\"\\uD83D\\uDE00\\uDC00\"}", String, InvalidArguments},
		{"pair separated by text", "{\"v\":\"\\uD83Dx\\uDE00\"}", String, InvalidArguments},
		{"surrogate nested key", "{\"v\":{\"\\uD800\":1}}", Object, InvalidArguments},
		{"surrogate root key", "{\"\\uDC00\":1}", String, InvalidArguments},
		{"surrogate in array", "{\"v\":[\"\\uDC00\"]}", Array, InvalidArguments},
		{"valid pair", "{\"v\":\"\\uD83D\\uDE00\"}", String, Allowed},
		{"lowercase pair", "{\"v\":\"\\ud83d\\ude00\"}", String, Allowed},
		{"literal replacement", "{\"v\":\"�\"}", String, Allowed},
		{"escaped replacement", "{\"v\":\"\\uFFFD\"}", String, Allowed},
		{"literal supplementary", "{\"v\":\"😀\"}", String, Allowed},
		{"escaped backslash not surrogate", "{\"v\":\"\\\\uD800\"}", String, Allowed},
		{"escaped quote before pair", "{\"v\":\"\\\"\\uD83D\\uDE00\"}", String, Allowed},
		{"valid pair duplicate key", "{\"v\":{\"\\uD83D\\uDE00\":1,\"😀\":2}}", Object, InvalidArguments},
		{"all simple escapes", "{\"v\":\"\\\"\\\\\\/\\b\\f\\n\\r\\t\"}", String, Allowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertDecision(t, argumentCatalog(t, tc.kind), Call{Name: "tool", Arguments: []byte(tc.raw)}, tc.want)
		})
	}
}

func quoted(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err) // Test fixture is always a string.
	}
	return string(b)
}

func objectWithMembers(n int) string {
	members := make([]string, n)
	for i := range members {
		members[i] = fmt.Sprintf("\"k%d\":null", i)
	}
	return "{" + strings.Join(members, ",") + "}"
}

func nullArray(n int) string {
	return "[" + strings.TrimSuffix(strings.Repeat("null,", n), ",") + "]"
}

// Root + outer array + eight child arrays + scalar leaves = n total values.
// Each child has <=128 elements; this isolates the total-value bound.
func totalValueArguments(n int) string {
	remaining := n - 10
	groups := make([]string, 8)
	for i := range groups {
		count := remaining
		if count > 128 {
			count = 128
		}
		groups[i] = nullArray(count)
		remaining -= count
	}
	if remaining != 0 {
		panic("invalid total-value test fixture")
	}
	return "{\"v\":[" + strings.Join(groups, ",") + "]}"
}

func TestStrictJSONResourceBounds(t *testing.T) {
	cases := []struct {
		name string
		kind ValueType
		at   string
		over string
	}{
		{"argument bytes", InvalidType, "{}" + strings.Repeat(" ", 16382), "{}" + strings.Repeat(" ", 16383)},
		{"array depth", Array,
			"{\"v\":" + strings.Repeat("[", 7) + "null" + strings.Repeat("]", 7) + "}",
			"{\"v\":" + strings.Repeat("[", 8) + "null" + strings.Repeat("]", 8) + "}"},
		{"object depth", Object,
			"{\"v\":" + strings.Repeat("{\"x\":", 6) + "{}" + strings.Repeat("}", 6) + "}",
			"{\"v\":" + strings.Repeat("{\"x\":", 7) + "{}" + strings.Repeat("}", 7) + "}"},
		{"object members", Object, "{\"v\":" + objectWithMembers(64) + "}", "{\"v\":" + objectWithMembers(65) + "}"},
		{"array elements", Array, "{\"v\":" + nullArray(128) + "}", "{\"v\":" + nullArray(129) + "}"},
		{"total values", Array, totalValueArguments(1024), totalValueArguments(1025)},
		{"decoded key bytes", Object,
			"{\"v\":{" + quoted(strings.Repeat("é", 64)) + ":null}}",
			"{\"v\":{" + quoted(strings.Repeat("é", 64)+"a") + ":null}}"},
		{"escaped key bytes", Object,
			"{\"v\":{\"" + strings.Repeat("\\u0061", 128) + "\":null}}",
			"{\"v\":{\"" + strings.Repeat("\\u0061", 129) + "\":null}}"},
		{"decoded string bytes", String,
			"{\"v\":" + quoted(strings.Repeat("é", 2048)) + "}",
			"{\"v\":" + quoted(strings.Repeat("é", 2048)+"a") + "}"},
		{"escaped string bytes", String,
			"{\"v\":\"" + strings.Repeat("\\n", 4096) + "\"}",
			"{\"v\":\"" + strings.Repeat("\\n", 4097) + "\"}"},
		{"number token bytes", Number,
			"{\"v\":" + strings.Repeat("1", 64) + "}",
			"{\"v\":" + strings.Repeat("1", 65) + "}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := mustCatalog(t, Definition{Name: "tool"})
			if tc.kind != InvalidType {
				c = argumentCatalog(t, tc.kind)
			}
			admitted := assertDecision(t, c, Call{Name: "tool", Arguments: []byte(tc.at)}, Allowed)
			snapshot, ok := admitted.AdmittedCall()
			if !ok || string(snapshot.Arguments) != tc.at {
				t.Fatal("boundary snapshot changed")
			}
			assertDecision(t, c, Call{Name: "tool", Arguments: []byte(tc.over)}, PolicyRejected)
		})
	}
	// Keys are not values: root + 32 nulls is 33, not 65.
	params := make([]Parameter, 32)
	for i := range params {
		params[i] = Parameter{Name: fmt.Sprintf("k%d", i), Type: Null, Required: true}
	}
	assertDecision(t, mustCatalog(t, Definition{Name: "tool", Parameters: params}),
		Call{Name: "tool", Arguments: []byte(objectWithMembers(32))}, Allowed)
}
