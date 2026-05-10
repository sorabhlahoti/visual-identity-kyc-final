package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/metrics"
)

func TestCORSPreflight(t *testing.T) {
	h := NewRouter(config.Config{ServiceName: "test-api"}, nil, &metrics.Counters{})
	req := httptest.NewRequest(http.MethodOptions, "/kyc/enroll", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Fatalf("unexpected Access-Control-Allow-Origin %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("expected Access-Control-Allow-Headers")
	}
}

func TestHealthHasCORS(t *testing.T) {
	h := NewRouter(config.Config{ServiceName: "test-api"}, nil, &metrics.Counters{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Fatalf("unexpected Access-Control-Allow-Origin %q", got)
	}
}

func TestCORSRejectsUnknownOriginWhenLockedDown(t *testing.T) {
	h := NewRouter(config.Config{ServiceName: "test-api", CORSAllowedOrigins: "https://allowed.example"}, nil, &metrics.Counters{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://blocked.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}
