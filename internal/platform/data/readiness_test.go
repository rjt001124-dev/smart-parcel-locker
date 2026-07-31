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
	const waitTimeout = 250 * time.Millisecond
	mysqlStarted := make(chan struct{})
	redisStarted := make(chan struct{})
	releaseMySQL := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseMySQL) })
	}
	mysqlError := errors.New("mysql unavailable")
	checker := newCheckerWithTimeout(
		func(context.Context) error {
			close(mysqlStarted)
			<-releaseMySQL
			return mysqlError
		},
		func(context.Context) error {
			close(redisStarted)
			return nil
		},
		time.Second,
	)

	result := make(chan Status, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		result <- checker.Check(context.Background())
	}()
	defer func() {
		release()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("Check goroutine did not exit during cleanup")
		}
	}()

	select {
	case <-mysqlStarted:
	case <-time.After(waitTimeout):
		t.Fatal("MySQL probe did not start")
	}
	select {
	case <-redisStarted:
	case <-time.After(waitTimeout):
		t.Fatal("Redis probe did not start before MySQL was released")
	}

	release()
	var got Status
	select {
	case got = <-result:
	case <-time.After(waitTimeout):
		t.Fatal("Check did not return after MySQL was released")
	}
	want := Status{Status: StatusNotReady, MySQL: DependencyStatusUnavailable, Redis: DependencyStatusOK}
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
