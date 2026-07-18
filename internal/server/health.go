package server

import (
	"encoding/json"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	platformdata "github.com/rjt001124-dev/smart-parcel-locker/internal/platform/data"
)

const serviceName = "smart-parcel-locker-api"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func RegisterHealth(srv *khttp.Server, version string) {
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Service: serviceName,
			Version: version,
		})
	})
}

func RegisterReadiness(srv *khttp.Server, readiness platformdata.Readiness) {
	srv.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		status := readiness.Check(r.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if status.Status == "not_ready" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(status)
	})
}
