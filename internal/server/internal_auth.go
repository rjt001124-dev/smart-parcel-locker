package server

import (
	"crypto/subtle"
	"net/http"
)

const internalTokenHeader = "X-Internal-Token"

// InternalAuth protects only /v1/internal routes. Public routes pass through
// unchanged so the same HTTP server can expose both contracts.
func InternalAuth(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.URL.Path) < len("/v1/internal/") || r.URL.Path[:len("/v1/internal/")] != "/v1/internal/" {
				next.ServeHTTP(w, r)
				return
			}
			provided := r.Header.Get(internalTokenHeader)
			if expected == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":401,"reason":"UNAUTHORIZED","message":"internal token required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
