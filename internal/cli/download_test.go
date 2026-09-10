package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadAutoName_WhenOutputArtifact_ItShouldUseZoaPrefixAndJsonExtension(t *testing.T) {
	opts := &downloadOptions{artifact: "output", file: ""}
	expected := "zoa-exec-123-output.json"

	result := resolveDownloadPath(opts, "exec-123", "application/json")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDownloadAutoName_WhenLogsArtifact_ItShouldUseZoaPrefixAndJsonlExtension(t *testing.T) {
	opts := &downloadOptions{artifact: "logs", file: ""}
	expected := "zoa-exec-456-logs.jsonl"

	result := resolveDownloadPath(opts, "exec-456", "text/plain")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDownloadAutoName_WhenGzipContent_ItShouldUseTarGzExtension(t *testing.T) {
	opts := &downloadOptions{artifact: "output", file: ""}
	expected := "zoa-exec-789-output.tar.gz"

	result := resolveDownloadPath(opts, "exec-789", "application/gzip")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDownloadAutoName_WhenFileSpecified_ItShouldUseProvidedPath(t *testing.T) {
	opts := &downloadOptions{artifact: "output", file: "/tmp/custom.json"}
	expected := "/tmp/custom.json"

	result := resolveDownloadPath(opts, "exec-789", "application/json")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDownloadStreamToFile_WhenServerReturns200_ItShouldSaveContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"result":"test-data"}`)
	}))
	defer server.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "test-output.json")

	n, err := streamToFile(server.URL+"/artifact", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Error("expected non-zero bytes written")
	}

	content, _ := os.ReadFile(outPath)
	if string(content) != `{"result":"test-data"}` {
		t.Errorf("expected %q, got %q", `{"result":"test-data"}`, string(content))
	}
}

func TestDownloadStreamToFile_WhenServerReturns404_ItShouldReturnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer server.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "should-not-exist.json")

	_, err := streamToFile(server.URL+"/artifact", outPath)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestDownloadOutputToFile_WhenServerReturnsTarGz_ItShouldReturnByteCount(t *testing.T) {
	body := []byte("fake-tar-gz-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trusted-actions/runs/exec-123/output" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	mock := &mockClient{
		rawGetFn: func(_ context.Context, path string) (*http.Response, error) {
			return http.Get(server.URL + path)
		},
	}

	dir := t.TempDir()
	outPath, nbytes, err := downloadOutputToFile(context.Background(), mock, "exec-123", filepath.Join(dir, "bundle.tar.gz"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outPath != filepath.Join(dir, "bundle.tar.gz") {
		t.Errorf("expected custom out path, got %q", outPath)
	}
	if nbytes != int64(len(body)) {
		t.Errorf("expected %d bytes, got %d", len(body), nbytes)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, body) {
		t.Errorf("unexpected file content: %q", content)
	}
}

func TestTarballUnpackCommand_WhenRelativePath_ItShouldChainExtractAndCd(t *testing.T) {
	got := tarballUnpackCommand("zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output.tar.gz")
	want := "tar xzf zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output.tar.gz --one-top-level && cd zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestTarballUnpackCommand_WhenAbsolutePath_ItShouldCdToParentFirst(t *testing.T) {
	got := tarballUnpackCommand("/tmp/zoa-exec-output.tar.gz")
	want := "cd /tmp && tar xzf zoa-exec-output.tar.gz --one-top-level && cd zoa-exec-output"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestPrintSavedArtifact_ItShouldWriteToStderr(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	printSavedArtifact("output", 42, "zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output.tar.gz")

	_ = w.Close()
	os.Stderr = old

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	msg := string(got)
	if !strings.Contains(msg, "Saved output (42B)") {
		t.Errorf("unexpected message: %q", msg)
	}
	if !strings.Contains(msg, tarballUnpackComment) {
		t.Errorf("expected unpack comment, got: %q", msg)
	}
	want := "tar xzf zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output.tar.gz --one-top-level && cd zoa-bc2fd50c-3965-4730-a112-46eb6b15e98d-output"
	if !strings.Contains(msg, want) {
		t.Errorf("expected unpack command %q, got: %q", want, msg)
	}
}
