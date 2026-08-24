package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func testArtifactBundle() ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "logger",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
			},
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
			{
				Module:     "web",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
			},
		},
	}
}

func TestMarshalArtifactBundle(t *testing.T) {
	bundle := testArtifactBundle()

	data, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	var document ArtifactBundleDocument

	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("serialized JSON is invalid: %v", err)
	}

	if document.ManifestName != "demo" {
		t.Fatalf(
			"expected manifest name %q, got %q",
			"demo",
			document.ManifestName,
		)
	}

	if document.ManifestVersion != "v1" {
		t.Fatalf(
			"expected manifest version %q, got %q",
			"v1",
			document.ManifestVersion,
		)
	}

	if len(document.Artifacts) != 3 {
		t.Fatalf(
			"expected 3 artifacts, got %d",
			len(document.Artifacts),
		)
	}

	if document.Artifacts[0].ImportPath !=
		"github.com/kaizenforyou91/forge/pkg/logger" {
		t.Fatalf(
			"expected logger import path, got %q",
			document.Artifacts[0].ImportPath,
		)
	}
}

func TestMarshalArtifactBundlePreservesArtifactOrder(t *testing.T) {
	bundle := testArtifactBundle()

	data, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	want := []ArtifactDocument{
		{
			Module:     "logger",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
		},
		{
			Module:     "http",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
		},
		{
			Module:     "web",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
		},
	}

	var document ArtifactBundleDocument

	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(document.Artifacts, want) {
		t.Fatalf(
			"expected artifact order %#v, got %#v",
			want,
			document.Artifacts,
		)
	}
}

func TestMarshalArtifactBundleIsDeterministic(t *testing.T) {
	bundle := testArtifactBundle()

	first, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	second, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf(
			"serialization is not deterministic:\nfirst:  %s\nsecond: %s",
			first,
			second,
		)
	}
}

func TestMarshalArtifactBundleRejectsInvalidBundle(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "",
		ManifestVersion: "v1",
	}

	_, err := MarshalArtifactBundle(bundle)
	if err == nil {
		t.Fatal("expected invalid bundle error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestUnmarshalArtifactBundle(t *testing.T) {
	data := []byte(`{
        "manifest_name":"demo",
        "manifest_version":"v1",
        "artifacts":[
            {
                "module":"logger",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/logger"
            },
            {
                "module":"http",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/http"
            },
            {
                "module":"web",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/router"
            }
        ]
    }`)

	got, err := UnmarshalArtifactBundle(data)
	if err != nil {
		t.Fatal(err)
	}

	want := testArtifactBundle()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"expected bundle %#v, got %#v",
			want,
			got,
		)
	}
}

func TestUnmarshalArtifactBundleRejectsInvalidJSON(t *testing.T) {
	_, err := UnmarshalArtifactBundle(
		[]byte(`{"manifest_name":`),
	)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestUnmarshalArtifactBundleRejectsEmptyDocument(t *testing.T) {
	_, err := UnmarshalArtifactBundle(nil)
	if err == nil {
		t.Fatal("expected empty document error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestUnmarshalArtifactBundleValidatesDecodedBundle(t *testing.T) {
	data := []byte(`{
        "manifest_name":"demo",
        "manifest_version":"v1",
        "artifacts":[
            {
                "module":"http",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/http"
            },
            {
                "module":"http",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/http"
            }
        ]
    }`)

	_, err := UnmarshalArtifactBundle(data)
	if err == nil {
		t.Fatal("expected duplicate artifact error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "duplicate artifact") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArtifactBundleJSONRoundTrip(t *testing.T) {
	original := testArtifactBundle()

	data, err := MarshalArtifactBundle(original)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := UnmarshalArtifactBundle(data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf(
			"round-trip changed bundle:\noriginal: %#v\ndecoded: %#v",
			original,
			decoded,
		)
	}
}

func TestArtifactBundleJSONRoundTripIsDeterministic(t *testing.T) {
	original := testArtifactBundle()

	first, err := MarshalArtifactBundle(original)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := UnmarshalArtifactBundle(first)
	if err != nil {
		t.Fatal(err)
	}

	second, err := MarshalArtifactBundle(decoded)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf(
			"round-trip serialization changed bytes:\nfirst:  %s\nsecond: %s",
			first,
			second,
		)
	}
}

func TestArtifactBundleSerializationDoesNotMutateInput(t *testing.T) {
	bundle := testArtifactBundle()
	original := bundle

	if _, err := MarshalArtifactBundle(bundle); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bundle, original) {
		t.Fatal("MarshalArtifactBundle mutated the input bundle")
	}
}

func TestUnmarshalArtifactBundleCreatesIndependentSlices(t *testing.T) {
	data := []byte(`{
        "manifest_name":"demo",
        "manifest_version":"v1",
        "artifacts":[
            {
                "module":"http",
                "version":"v1",
                "import_path":"github.com/kaizenforyou91/forge/pkg/http"
            }
        ]
    }`)

	bundle, err := UnmarshalArtifactBundle(data)
	if err != nil {
		t.Fatal(err)
	}

	bundle.Artifacts[0].Module = "changed"

	if bundle.Artifacts[0].Module != "changed" {
		t.Fatal("expected decoded bundle to own its artifact slice")
	}
}

func TestMarshalArtifactBundleV1CanonicalBytesUnchanged(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName: "demo", ManifestVersion: "v1",
		Artifacts: []Artifact{{Module: "demo", Version: "v1", ImportPath: "example.com/demo"}},
	}
	want := []byte(`{"manifest_name":"demo","manifest_version":"v1","artifacts":[{"module":"demo","version":"v1","import_path":"example.com/demo"}]}`)
	got, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestUnmarshalArtifactBundleV1IgnoresRuntimeField(t *testing.T) {
	data := []byte(`{"manifest_name":"demo","manifest_version":"v1","runtime":{"kind":"future"},"artifacts":[{"module":"demo","version":"v1","import_path":"example.com/demo"}]}`)
	bundle, err := UnmarshalArtifactBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Runtime != nil {
		t.Fatalf("expected v1 runtime to be ignored, got %#v", bundle.Runtime)
	}
}

func TestUnmarshalArtifactBundleV1UnknownFieldRegression(t *testing.T) {
	data := []byte(`{"manifest_name":"demo","manifest_version":"v1","future":true,"artifacts":[{"module":"demo","version":"v1","import_path":"example.com/demo","future":true}]}`)
	if _, err := UnmarshalArtifactBundle(data); err != nil {
		t.Fatal(err)
	}
}

func TestMarshalArtifactBundleV2CanonicalBytes(t *testing.T) {
	want := []byte(`{"manifest_name":"demo","manifest_version":"v1","runtime":{"kind":"application_executable","entrypoint":{"module":"demo","version":"v1"},"target_os":"windows","target_arch":"amd64"},"artifacts":[{"module":"demo","version":"v1","import_path":"example.com/demo"}]}`)
	got, err := marshalArtifactBundleForSchema(testRunnableArtifactBundle(), artifactBundleSchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestArtifactBundleV2SerializationIsDeterministic(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	first, err := marshalArtifactBundleForSchema(bundle, artifactBundleSchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		got, err := marshalArtifactBundleForSchema(bundle, artifactBundleSchemaVersionV2)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, first) {
			t.Fatalf("serialization %d differs", i)
		}
	}
}

func TestUnmarshalArtifactBundleV2RoundTrip(t *testing.T) {
	want := testRunnableArtifactBundle()
	data, err := marshalArtifactBundleForSchema(want, artifactBundleSchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := unmarshalArtifactBundleForSchema(data, artifactBundleSchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}

	got.Runtime.TargetOS = "changed"
	if want.Runtime.TargetOS != "windows" {
		t.Fatal("decoded runtime descriptor aliases source bundle")
	}
}

func TestUnmarshalArtifactBundleV2RejectsUnknownRootField(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"artifacts":`, `"future":true,"artifacts":`))
}

func TestUnmarshalArtifactBundleV2RejectsUnknownRuntimeField(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"entrypoint":`, `"future":true,"entrypoint":`))
}

func TestUnmarshalArtifactBundleV2RejectsUnknownEntrypointField(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"module":"demo"`, `"future":true,"module":"demo"`))
}

func TestUnmarshalArtifactBundleV2RejectsUnknownArtifactField(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"import_path":"example.com/demo"`, `"import_path":"example.com/demo","future":true`))
}

func TestUnmarshalArtifactBundleV2RejectsDuplicateRootKey(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"manifest_version":"v1"`, `"manifest_version":"v1","manifest_version":"v2"`))
}

func TestUnmarshalArtifactBundleV2RejectsDuplicateRuntimeKey(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"kind":"application_executable"`, `"kind":"application_executable","kind":"other"`))
}

func TestUnmarshalArtifactBundleV2RejectsDuplicateEntrypointKey(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"module":"demo"`, `"module":"demo","module":"other"`))
}

func TestUnmarshalArtifactBundleV2RejectsDuplicateArtifactKey(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, replaceV2BundleJSON(t, `"import_path":"example.com/demo"`, `"import_path":"example.com/demo","import_path":"other"`))
}

func TestUnmarshalArtifactBundleV2RejectsTrailingJSON(t *testing.T) {
	requireInvalidArtifactBundleJSON(t, append(canonicalV2BundleJSON(t), []byte(`{}`)...))
}

func canonicalV2BundleJSON(t *testing.T) []byte {
	t.Helper()
	data, err := marshalArtifactBundleForSchema(testRunnableArtifactBundle(), artifactBundleSchemaVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func replaceV2BundleJSON(t *testing.T, old, replacement string) []byte {
	t.Helper()
	data := canonicalV2BundleJSON(t)
	result := bytes.Replace(data, []byte(old), []byte(replacement), 1)
	if bytes.Equal(result, data) {
		t.Fatalf("fixture pattern %q not found", old)
	}
	return result
}

func requireInvalidArtifactBundleJSON(t *testing.T, data []byte) {
	t.Helper()
	_, err := unmarshalArtifactBundleForSchema(data, artifactBundleSchemaVersionV2)
	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
