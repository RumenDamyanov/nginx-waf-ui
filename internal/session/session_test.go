package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndGet(t *testing.T) {
	store := NewStore("test-secret", 3600)

	// Create session
	w := httptest.NewRecorder()
	store.Create(w, "admin")

	// Extract cookie
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookie set")
	}

	// Get session
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	sess := store.Get(req)
	if sess == nil {
		t.Fatal("session not found")
	}
	if sess.Username != "admin" {
		t.Errorf("username = %q", sess.Username)
	}
}

func TestGetMissing(t *testing.T) {
	store := NewStore("test-secret", 3600)
	req := httptest.NewRequest("GET", "/", nil)
	if sess := store.Get(req); sess != nil {
		t.Error("expected nil session")
	}
}

func TestGetTampered(t *testing.T) {
	store := NewStore("test-secret", 3600)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "waf_session", Value: "bad-id.bad-sig"})
	if sess := store.Get(req); sess != nil {
		t.Error("expected nil for tampered cookie")
	}
}

func TestDestroy(t *testing.T) {
	store := NewStore("test-secret", 3600)

	w := httptest.NewRecorder()
	store.Create(w, "admin")
	resp := w.Result()
	cookies := resp.Cookies()

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])

	w2 := httptest.NewRecorder()
	store.Destroy(w2, req)

	// Session should be gone
	if sess := store.Get(req); sess != nil {
		t.Error("session should be destroyed")
	}
}
