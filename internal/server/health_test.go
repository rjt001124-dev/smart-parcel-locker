package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	platformdata "github.com/rjt001124-dev/smart-parcel-locker/internal/platform/data"
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

func TestRegisterReadiness(t *testing.T) {
	tests := []struct {
		name       string
		status     platformdata.Status
		wantStatus int
	}{
		{
			name:       "ok",
			status:     platformdata.Status{Status: "ok", MySQL: "ok", Redis: "ok"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "degraded",
			status:     platformdata.Status{Status: "degraded", MySQL: "ok", Redis: "unavailable"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not ready",
			status:     platformdata.Status{Status: "not_ready", MySQL: "unavailable", Redis: "ok"},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := khttp.NewServer()
			RegisterReadiness(srv, fakeReadiness{
				status: tt.status,
				secret: "supplied-fake-secret-raw-error",
			})

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", got)
			}

			body, err := io.ReadAll(rec.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			var got platformdata.Status
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got != tt.status {
				t.Fatalf("response = %+v, want %+v", got, tt.status)
			}
			if strings.Contains(string(body), "supplied-fake-secret-raw-error") {
				t.Fatal("response exposed internal error or secret text")
			}
		})
	}
}

type fakeReadiness struct {
	status platformdata.Status
	secret string
}

func (f fakeReadiness) Check(context.Context) platformdata.Status {
	return f.status
}
