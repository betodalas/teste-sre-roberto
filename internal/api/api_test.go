package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequest(t *testing.T, handler http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeResult(t *testing.T, rec *httptest.ResponseRecorder) resultResponse {
	t.Helper()
	var body resultResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

func TestSumEndpoint(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/sum?term_one=4&term_two=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := decodeResult(t, rec).Result; got != 5 {
		t.Errorf("result = %d, want 5", got)
	}
}

func TestSubEndpoint(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/sub?term_one=4&term_two=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := decodeResult(t, rec).Result; got != 3 {
		t.Errorf("result = %d, want 3", got)
	}
}

func TestMulEndpoint(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/mul?term_one=4&term_two=3")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := decodeResult(t, rec).Result; got != 12 {
		t.Errorf("result = %d, want 12", got)
	}
}

func TestDivEndpoint(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/div?term_one=10&term_two=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := decodeResult(t, rec).Result; got != 5 {
		t.Errorf("result = %d, want 5", got)
	}
}

func TestDivEndpointByZero(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/div?term_one=10&term_two=0")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestMissingParam(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/sum?term_one=4")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestInvalidParam(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/sum?term_one=abc&term_two=1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodPost, "/api/sum?term_one=4&term_two=1")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
