package data

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
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

func TestCheckerRunsProbesConcurrentlyWithinDeadline(t *testing.T) {
	const timeout = 50 * time.Millisecond
	redisRan := make(chan struct{})
	checker := newCheckerWithTimeout(
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		func(context.Context) error {
			close(redisRan)
			return nil
		},
		timeout,
	)

	started := time.Now()
	got := checker.Check(context.Background())
	elapsed := time.Since(started)

	select {
	case <-redisRan:
	default:
		t.Fatal("Redis probe did not run while MySQL was blocked")
	}
	if elapsed >= time.Second {
		t.Fatalf("Check took %v, want bounded by the short checker deadline", elapsed)
	}
	want := Status{Status: "not_ready", MySQL: "unavailable", Redis: "ok"}
	if got != want {
		t.Fatalf("Check() = %+v, want %+v", got, want)
	}
}

func TestCheckerIsSafeForConcurrentCalls(t *testing.T) {
	checker := NewChecker(
		func(context.Context) error { return nil },
		func(context.Context) error { return errors.New("redis unavailable") },
	)
	want := Status{Status: StatusDegraded, MySQL: DependencyStatusOK, Redis: DependencyStatusUnavailable}

	const callers = 20
	results := make([]Status, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := range callers {
		go func() {
			defer wg.Done()
			results[i] = checker.Check(context.Background())
		}()
	}
	wg.Wait()

	for i, got := range results {
		if got != want {
			t.Errorf("Check caller %d = %+v, want %+v", i, got, want)
		}
	}
}
