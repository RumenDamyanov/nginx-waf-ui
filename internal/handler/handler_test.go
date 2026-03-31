package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RumenDamyanov/nginx-waf-ui/internal/api"
	"github.com/RumenDamyanov/nginx-waf-ui/internal/session"
)

func setupTest(t *testing.T) (*http.ServeMux, *session.Store, *httptest.Server) {
	// Mock API server
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("/api/v1/lists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"data":   []map[string]any{{"name": "test", "entries": 2, "mod_time": "2025-01-01T00:00:00Z"}},
		})
	})
	mockMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mockSrv := httptest.NewServer(mockMux)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := api.NewClient(mockSrv.URL, "test-key", 5*time.Second)
	store := session.NewStore("test-secret", 3600)
	h := New(client, store, logger, "admin")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, store, mockSrv
}

func TestLoginPage(t *testing.T) {
	mux, _, mockSrv := setupTest(t)
	defer mockSrv.Close()

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/login", nil))
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Sign In") {
		t.Error("expected login form")
	}
}

func TestLoginSuccess(t *testing.T) {
	mux, _, mockSrv := setupTest(t)
	defer mockSrv.Close()

	body := strings.NewReader("username=admin&password=admin")
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 303 {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Errorf("redirect = %q", loc)
	}
}

func TestLoginFail(t *testing.T) {
	mux, _, mockSrv := setupTest(t)
	defer mockSrv.Close()

	body := strings.NewReader("username=admin&password=wrong")
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid") {
		t.Error("expected error message")
	}
}

func TestDashboardRequiresAuth(t *testing.T) {
	mux, _, mockSrv := setupTest(t)
	defer mockSrv.Close()

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 303 {
		t.Fatalf("status = %d, want 303 redirect to login", rr.Code)
	}
}

func TestDashboardAuthed(t *testing.T) {
	mux, store, mockSrv := setupTest(t)
	defer mockSrv.Close()

	// Create session
	w := httptest.NewRecorder()
	store.Create(w, "admin")
	cookies := w.Result().Cookies()

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Dashboard") {
		t.Error("expected dashboard content")
	}
}

func TestHealth(t *testing.T) {
	mux, _, mockSrv := setupTest(t)
	defer mockSrv.Close()

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/health", nil))
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
}
