package biz

import (
	"context"
	"sync"
	"testing"
	"time"
)

type reclaimUseCaseStub struct {
	mu        sync.Mutex
	calls     int
	now       time.Time
	batchSize int
	results   []int
}

func (s *reclaimUseCaseStub) ReclaimExpired(context.Context, time.Time, int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if len(s.results) == 0 {
		return 0, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

func TestReclaimerRunOnceDrainsBoundedBatches(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	uc := &reclaimUseCaseStub{results: []int{100, 4}}
	reclaimer := NewReclaimer(uc, func() time.Time { return now }, nil)
	reclaimed, err := reclaimer.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed != 104 || uc.calls != 2 {
		t.Fatalf("reclaimed=%d calls=%d, want 104/2", reclaimed, uc.calls)
	}
}

func TestReclaimerStopsOnCanceledContextWithoutAnotherCall(t *testing.T) {
	uc := &reclaimUseCaseStub{}
	reclaimer := NewReclaimer(uc, time.Now, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := reclaimer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if uc.calls != 0 {
		t.Fatalf("calls=%d, want 0", uc.calls)
	}
}
