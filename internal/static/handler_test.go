package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
