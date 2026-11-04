package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticFile(t *testing.T) {
	dir := t.TempDir()

	content := []byte("Hello from Gorpx!")

	err := os.WriteFile(
		filepath.Join(dir, "hello.txt"),
		content,
		0600,
	)
	if err != nil {
		t.Fatal(err)
	}

	handler := New("/static/", dir)

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/hello.txt",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got := rec.Body.String(); got != string(content) {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestDirectoryListingDisabled(t *testing.T) {
	dir := t.TempDir()

	err := os.Mkdir(
		filepath.Join(dir, "images"),
		0755,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(
		filepath.Join(dir, "images", "logo.png"),
		[]byte("test image"),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	handler := New("/static/", dir)

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/images/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf(
			"status = %d, want 403",
			rec.Code,
		)
	}
}

func TestSymlinkEscape(t *testing.T) {
	dir := t.TempDir()

	public := filepath.Join(dir, "public")
	private := filepath.Join(dir, "private")

	for _, path := range []string{public, private} {
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatal(err)
		}
	}

	secret := filepath.Join(private, "secret.txt")

	if err := os.WriteFile(
		secret,
		[]byte("PRIVATE DATA"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(public, "secret.txt")

	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	handler := New("/static/", public)

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/secret.txt",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf(
			"status = %d, want 403",
			rec.Code,
		)
	}

	if strings.Contains(rec.Body.String(), "PRIVATE DATA") {
		t.Error("private file contents were exposed")
	}
}

func TestInternalSymlink(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(dir, "hello.txt"),
		[]byte("Hello"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(
		"hello.txt",
		filepath.Join(dir, "alias.txt"),
	); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/alias.txt",
		nil,
	)
	rec := httptest.NewRecorder()

	New("/static/", dir).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got := rec.Body.String(); got != "Hello" {
		t.Errorf("body = %q, want Hello", got)
	}
}
