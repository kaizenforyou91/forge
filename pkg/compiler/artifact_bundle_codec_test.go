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
