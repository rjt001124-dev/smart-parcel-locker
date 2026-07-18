package data

import (
	"context"
	"errors"
	"testing"
)

func TestCheckerStatus(t *testing.T) {
	tests := []struct {
		name       string
		mysqlError error
		redisError error
		want       Status
	}{
		{
			name: "healthy",
			want: Status{Status: "ok", MySQL: "ok", Redis: "ok"},
		},
		{
			name:       "MySQL unavailable",
			mysqlError: errors.New("mysql secret raw error"),
			want:       Status{Status: "not_ready", MySQL: "unavailable", Redis: "ok"},
		},
		{
			name:       "Redis unavailable",
			redisError: errors.New("redis secret raw error"),
			want:       Status{Status: "degraded", MySQL: "ok", Redis: "unavailable"},
		},
		{
			name:       "both unavailable",
			mysqlError: errors.New("mysql secret raw error"),
			redisError: errors.New("redis secret raw error"),
			want:       Status{Status: "not_ready", MySQL: "unavailable", Redis: "unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(
				func(context.Context) error { return tt.mysqlError },
				func(context.Context) error { return tt.redisError },
			)

			if got := checker.Check(context.Background()); got != tt.want {
				t.Fatalf("Check() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
