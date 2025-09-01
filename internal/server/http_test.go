package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cinekami-server/internal/server"
	"cinekami-server/pkg/cache"
)

func TestHealth(t *testing.T) {
	signer := server.NewCursorSigner([]byte("test-secret"))
	s := server.New(nil, cache.NewInMemory(), signer)
	r := s.Router()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}
