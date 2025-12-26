package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
)

func TestIntercept404CallsFallback(t *testing.T) {
	primary := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("missing"))
	})
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("index"))
	})

	handler := Intercept404(primary, fallback)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "index" {
		t.Fatalf("expected fallback body, got %q", rec.Body.String())
	}
}

func TestIntercept404SkipsFallbackOnSuccess(t *testing.T) {
	primary := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
	})

	handler := Intercept404(primary, fallback)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if fallbackCalled {
		t.Fatal("expected fallback not to be called")
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected primary body, got %q", rec.Body.String())
	}
}

func TestServeFileContentsHTML(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte("<html>ok</html>"), ModTime: time.Now()},
	}

	handler := ServeFileContents("index.html", http.FS(files))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "<html>ok</html>" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestServeFileContentsRejectsNonHTML(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte("<html>ok</html>"), ModTime: time.Now()},
	}

	handler := ServeFileContents("index.html", http.FS(files))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}
