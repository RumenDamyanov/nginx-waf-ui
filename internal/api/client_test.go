package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func mockAPI() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/lists", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "ok",
			"data": []map[string]any{
				{"name": "blocklist", "path": "blocklist.txt", "entries": 5, "mod_time": "2025-01-01T00:00:00Z"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/v1/lists/blocklist", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "ok",
			"data": map[string]any{
				"name": "blocklist", "path": "blocklist.txt", "entries": 2, "mod_time": "2025-01-01T00:00:00Z",
				"ips": []string{"192.168.1.1", "10.0.0.0/8"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return httptest.NewServer(mux)
}

func TestGetLists(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	client := NewClient(srv.URL, "test-key", 5*time.Second)
	lists, err := client.GetLists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}
	if lists[0].Name != "blocklist" {
		t.Errorf("name = %q", lists[0].Name)
	}
}

func TestGetList(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	client := NewClient(srv.URL, "test-key", 5*time.Second)
	detail, err := client.GetList("blocklist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.IPs) != 2 {
		t.Errorf("ips = %d, want 2", len(detail.IPs))
	}
}

func TestHealth(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	client := NewClient(srv.URL, "test-key", 5*time.Second)
	if err := client.Health(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "my-secret", 5*time.Second)
	client.Health()
	if gotAuth != "Bearer my-secret" {
		t.Errorf("auth = %q", gotAuth)
	}
}
