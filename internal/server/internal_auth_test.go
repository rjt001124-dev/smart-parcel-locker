package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalAuthRejectsMissingToken(t *testing.T) {
	nextCalled := false
	handler := InternalAuth("expected")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }))
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/cells/reserve", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized || nextCalled {
		t.Fatalf("unexpected auth result: status=%d next=%v", recorder.Code, nextCalled)
	}
}

func TestInternalAuthRejectsWrongToken(t *testing.T) {
	nextCalled := false
	handler := InternalAuth("expected")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }))
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/cells/reserve", nil)
	req.Header.Set("X-Internal-Token", "wrong")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized || nextCalled {
		t.Fatalf("unexpected auth result: status=%d next=%v", recorder.Code, nextCalled)
	}
}

func TestInternalAuthAllowsCorrectTokenAndPublicRoutes(t *testing.T) {
	calls := 0
	handler := InternalAuth("expected")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))

	internal := httptest.NewRequest(http.MethodPost, "/v1/internal/cells/reserve", nil)
	internal.Header.Set("X-Internal-Token", "expected")
	internalRecorder := httptest.NewRecorder()
	handler.ServeHTTP(internalRecorder, internal)
	if internalRecorder.Code != http.StatusOK || calls != 1 {
		t.Fatalf("correct token result: status=%d calls=%d", internalRecorder.Code, calls)
	}

	public := httptest.NewRequest(http.MethodGet, "/v1/cities", nil)
	publicRecorder := httptest.NewRecorder()
	handler.ServeHTTP(publicRecorder, public)
	if publicRecorder.Code != http.StatusOK || calls != 2 {
		t.Fatalf("public route result: status=%d calls=%d", publicRecorder.Code, calls)
	}
}

func TestInternalAuthRejectsEmptyConfiguredToken(t *testing.T) {
	handler := InternalAuth("")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("next must not be called") }))
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/cells/reserve", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", recorder.Code)
	}
}
