package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestRegisterHealth(t *testing.T) {
	srv := khttp.NewServer()
	RegisterHealth(srv, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "ok" || got.Service != "smart-parcel-locker-api" || got.Version != "test-version" {
		t.Fatalf("response = %+v", got)
	}
}
