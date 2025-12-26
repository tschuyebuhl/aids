package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerPreservesStatusCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	logged := NewLogger(handler, WithLogger(logger))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	logged.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestLoggerPanicHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	called := false
	panicHandler := func(w http.ResponseWriter, r *http.Request, recovered any, stack []byte) {
		called = true
		http.Error(w, "handled", http.StatusTeapot)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	logged := NewLogger(handler, WithLogger(logger), WithPanicHandler(panicHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	logged.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected panic handler to be called")
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d", rec.Code)
	}
}
