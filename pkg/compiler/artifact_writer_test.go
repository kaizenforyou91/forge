package compiler

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeArtifactWriter struct {
	artifacts []Artifact
	payloads  [][]byte
	err       error
}

func (w *fakeArtifactWriter) Write(
	artifact Artifact,
	data []byte,
) error {
	if w.err != nil {
		return w.err
	}

	w.artifacts = append(w.artifacts, artifact)
	w.payloads = append(
		w.payloads,
		append([]byte(nil), data...),
	)

	return nil
}

func TestArtifactWriterContract(t *testing.T) {
	var _ ArtifactWriter = (*fakeArtifactWriter)(nil)
}

func TestFileArtifactWriterCreatesDeterministicArtifact(t *testing.T) {
	root := t.TempDir()

	writer, err := NewFileArtifactWriter(root)
	if err != nil {
		t.Fatal(err)
	}

	artifact := Artifact{
		Module:  "web",
		Version: "v1",
	}

	data := []byte("compiled artifact")

	if err := writer.Write(artifact, data); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(
		root,
		"web",
		"v1",
		"artifact",
	)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, data) {
		t.Fatalf(
			"unexpected artifact data: want %q, got %q",
			string(data),
			string(got),
		)
	}
}

func TestFileArtifactWriterCreatesDirectories(t *testing.T) {
	root := filepath.Join(
		t.TempDir(),
		"nested",
		"artifacts",
	)

	writer, err := NewFileArtifactWriter(root)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Write(
		Artifact{
			Module:  "http",
			Version: "v1",
		},
		[]byte("payload"),
	)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(
		root,
		"http",
		"v1",
		"artifact",
	)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected artifact file: %v", err)
	}
}

func TestFileArtifactWriterOverwritesDeterministically(t *testing.T) {
	root := t.TempDir()

	writer, err := NewFileArtifactWriter(root)
	if err != nil {
		t.Fatal(err)
	}

	artifact := Artifact{
		Module:  "web",
		Version: "v1",
	}

	if err := writer.Write(
		artifact,
		[]byte("first"),
	); err != nil {
		t.Fatal(err)
	}

	if err := writer.Write(
		artifact,
		[]byte("second"),
	); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(
		root,
		"web",
		"v1",
		"artifact",
	)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "second" {
		t.Fatalf(
			"expected latest artifact contents, got %q",
			string(got),
		)
	}
}

func TestFileArtifactWriterRejectsInvalidArtifact(t *testing.T) {
	writer, err := NewFileArtifactWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tests := []Artifact{
		{},
		{Module: "web"},
		{Version: "v1"},
		{Module: "../web", Version: "v1"},
		{Module: "web/sub", Version: "v1"},
		{Module: "web", Version: "../v1"},
		{Module: "web", Version: "v1/sub"},
	}

	for _, artifact := range tests {
		if err := writer.Write(artifact, []byte("x")); err == nil {
			t.Fatalf(
				"expected invalid artifact error for %#v",
				artifact,
			)
		}
	}
}

func TestNewFileArtifactWriterRejectsEmptyRoot(t *testing.T) {
	_, err := NewFileArtifactWriter("")

	if err != ErrInvalidOutputRoot {
		t.Fatalf(
			"expected ErrInvalidOutputRoot, got %v",
			err,
		)
	}
}

func TestFileArtifactWriterNilReceiver(t *testing.T) {
	var writer *FileArtifactWriter

	err := writer.Write(
		Artifact{
			Module:  "web",
			Version: "v1",
		},
		[]byte("x"),
	)

	if err != ErrNilArtifactWriter {
		t.Fatalf(
			"expected ErrNilArtifactWriter, got %v",
			err,
		)
	}
}

func TestFakeArtifactWriterCopiesPayload(t *testing.T) {
	writer := &fakeArtifactWriter{}

	payload := []byte("payload")

	err := writer.Write(
		Artifact{
			Module:  "web",
			Version: "v1",
		},
		payload,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload[0] = 'X'

	if string(writer.payloads[0]) != "payload" {
		t.Fatalf("writer payload was aliased to caller data")
	}
}
