package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"cinekami-server/internal/server"

	pkgcache "cinekami-server/pkg/cache"
	pkgcrypto "cinekami-server/pkg/crypto"
)

func TestHealth(t *testing.T) {
	signer := pkgcrypto.NewHMAC([]byte("test-secret"))
	s := server.New(nil, pkgcache.NewInMemory(), signer)
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

func TestVoteInvalidCategory(t *testing.T) {
	s := server.New(nil, pkgcache.NewInMemory(), pkgcrypto.NewHMAC([]byte("test")))
	r := s.Router()
	body := bytes.NewBufferString(`{"movie_id": 1, "category": "not_valid", "fingerprint": "abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/movies/1/votes", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	resp := w.Body.String()
	if resp == "" || (resp != "" && !contains(resp, "invalid category")) {
		t.Fatalf("expected error message to mention invalid category, got: %s", resp)
	}
}

func contains(s, substr string) bool { return bytes.Contains([]byte(s), []byte(substr)) }
