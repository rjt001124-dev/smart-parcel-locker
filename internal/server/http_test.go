package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
)

type simulatorServiceStub struct{}

func (simulatorServiceStub) SetSimulatorScenario(context.Context, *v1.SetSimulatorScenarioRequest) (*v1.SetSimulatorScenarioReply, error) {
	return &v1.SetSimulatorScenarioReply{Updated: true}, nil
}

func TestHTTPServerDoesNotRegisterSimulatorWhenServiceIsNil(t *testing.T) {
	srv := NewHTTPServer(conf.Config{InternalAPIToken: "token"}, "test", nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/v1/internal/simulator/devices/DEV-1/scenario", nil)
	req.Header.Set("X-Internal-Token", "token")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s, want 404", recorder.Code, recorder.Body.String())
	}
}

func TestHTTPServerRegistersSimulatorOnlyWhenProvided(t *testing.T) {
	srv := NewHTTPServer(conf.Config{InternalAPIToken: "token"}, "test", nil, nil, nil, nil, simulatorServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/v1/internal/simulator/devices/DEV-1/scenario", nil)
	req.Header.Set("X-Internal-Token", "token")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
}
