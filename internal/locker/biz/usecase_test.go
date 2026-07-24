package biz

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRepository struct {
	existing      *Reservation
	findErr       error
	reserveResult Reservation
	reserveErr    error
	releaseErr    error
	reclaimErr    error
	reclaimCount  int
	reserveCalls  int
	releaseCalls  int
	reclaimCalls  int
	lastReserve   ReserveRequest
	lastExpires   time.Time
	lastCellID    uint64
	lastKey       string
	lastReclaimAt time.Time
	lastBatchSize int
}

func (f *fakeRepository) FindReservation(_ context.Context, key string) (Reservation, error) {
	if f.findErr != nil {
		return Reservation{}, f.findErr
	}
	if f.existing == nil || f.existing.ReservationKey != key {
		return Reservation{}, ErrReservationNotFound
	}
	return *f.existing, nil
}

func (f *fakeRepository) ReserveAvailable(_ context.Context, req ReserveRequest, expiresAt time.Time) (Reservation, error) {
	f.reserveCalls++
	f.lastReserve = req
	f.lastExpires = expiresAt
	if f.reserveErr != nil {
		return Reservation{}, f.reserveErr
	}
	if f.reserveResult.ReservationKey == "" {
		f.reserveResult = Reservation{CellID: 7, ReservationKey: req.ReservationKey, ExpiresAt: expiresAt}
	}
	return f.reserveResult, nil
}

func (f *fakeRepository) Release(_ context.Context, cellID uint64, key string) error {
	f.releaseCalls++
	f.lastCellID = cellID
	f.lastKey = key
	if f.releaseErr != nil {
		return f.releaseErr
	}
	return nil
}

func (f *fakeRepository) ReclaimExpired(_ context.Context, now time.Time, batchSize int) (int, error) {
	f.reclaimCalls++
	f.lastReclaimAt = now
	f.lastBatchSize = batchSize
	return f.reclaimCount, f.reclaimErr
}

type fakeMarker struct {
	markErr   error
	deleteErr error
	marks     []Reservation
	deletes   []Reservation
}

func (f *fakeMarker) Mark(_ context.Context, reservation Reservation) error {
	f.marks = append(f.marks, reservation)
	return f.markErr
}

func (f *fakeMarker) Delete(_ context.Context, reservation Reservation) error {
	f.deletes = append(f.deletes, reservation)
	return f.deleteErr
}

func TestReserveReturnsExistingReservationForSameKey(t *testing.T) {
	expires := time.Date(2026, 7, 18, 1, 5, 0, 0, time.UTC)
	repo := &fakeRepository{existing: &Reservation{CellID: 9, ReservationKey: "req-1", ExpiresAt: expires}}
	marker := &fakeMarker{}
	uc := NewUseCase(repo, marker, func() time.Time { return expires.Add(-time.Minute) })

	got, err := uc.Reserve(context.Background(), ReserveRequest{SiteID: 1, Size: SizeMedium, ReservationKey: "req-1", TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.CellID != 9 || repo.reserveCalls != 0 {
		t.Fatalf("unexpected replay: %+v calls=%d", got, repo.reserveCalls)
	}
	if len(marker.marks) != 0 {
		t.Fatalf("replay rewrote marker: %+v", marker.marks)
	}
}

func TestReserveUsesDefaultAndMaximumTTL(t *testing.T) {
	now := time.Date(2026, 7, 24, 2, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{name: "default", ttl: 0, want: DefaultReservationTTL},
		{name: "maximum", ttl: MaxReservationTTL, want: MaxReservationTTL},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepository{}
			uc := NewUseCase(repo, nil, func() time.Time { return now })
			if _, err := uc.Reserve(context.Background(), ReserveRequest{SiteID: 1, Size: SizeMedium, ReservationKey: "key-" + test.name, TTL: test.ttl}); err != nil {
				t.Fatal(err)
			}
			if got, want := repo.lastExpires, now.Add(test.want); !got.Equal(want) {
				t.Fatalf("expires at %s, want %s", got, want)
			}
		})
	}
}

func TestNewUseCaseDefaultClockUsesUTC(t *testing.T) {
	previousLocal := time.Local
	time.Local = time.FixedZone("test-local", 8*60*60)
	defer func() { time.Local = previousLocal }()

	repo := &fakeRepository{}
	uc := NewUseCase(repo, nil, nil)

	if _, err := uc.Reserve(context.Background(), ReserveRequest{
		SiteID:         1,
		Size:           SizeMedium,
		ReservationKey: "default-clock",
		TTL:            MinReservationTTL,
	}); err != nil {
		t.Fatal(err)
	}
	if got := repo.lastExpires.Location(); got != time.UTC {
		t.Fatalf("default clock location = %v, want UTC", got)
	}
}

func TestReserveRejectsInvalidTTLAndBlankIdempotencyKey(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo, nil, time.Now)
	for _, test := range []struct {
		name string
		req  ReserveRequest
		want error
	}{
		{name: "too short", req: ReserveRequest{ReservationKey: "key", TTL: MinReservationTTL - time.Nanosecond}, want: ErrInvalidReservationTTL},
		{name: "too long", req: ReserveRequest{ReservationKey: "key", TTL: MaxReservationTTL + time.Nanosecond}, want: ErrInvalidReservationTTL},
		{name: "empty key", req: ReserveRequest{TTL: MinReservationTTL}, want: ErrIdempotencyKeyRequired},
		{name: "whitespace key", req: ReserveRequest{ReservationKey: " \t\n", TTL: MinReservationTTL}, want: ErrIdempotencyKeyRequired},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := uc.Reserve(context.Background(), test.req); !errors.Is(err, test.want) {
				t.Fatalf("Reserve() error = %v, want %v", err, test.want)
			}
			if repo.reserveCalls != 0 {
				t.Fatalf("ReserveAvailable calls = %d, want 0", repo.reserveCalls)
			}
		})
	}
}

func TestReservePropagatesOfflineAndNoAvailableErrors(t *testing.T) {
	for _, want := range []error{ErrDeviceOffline, ErrCellNotAvailable} {
		t.Run(want.Error(), func(t *testing.T) {
			repo := &fakeRepository{reserveErr: want}
			uc := NewUseCase(repo, nil, time.Now)
			_, err := uc.Reserve(context.Background(), ReserveRequest{SiteID: 1, Size: SizeMedium, ReservationKey: "key", TTL: MinReservationTTL})
			if !errors.Is(err, want) {
				t.Fatalf("Reserve() error = %v, want %v", err, want)
			}
		})
	}
}

func TestReserveMarkerFailureIsNonAuthoritative(t *testing.T) {
	repo := &fakeRepository{}
	marker := &fakeMarker{markErr: errors.New("temporary marker failure")}
	uc := NewUseCase(repo, marker, time.Now)

	got, err := uc.Reserve(context.Background(), ReserveRequest{SiteID: 1, Size: SizeSmall, ReservationKey: "key", TTL: MinReservationTTL})
	if err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}
	if got.CellID == 0 || len(marker.marks) != 1 {
		t.Fatalf("reservation/marker = %+v/%+v", got, marker.marks)
	}
}

func TestReleaseRequiresMatchingReservationKeyAndMarkerFailureIsNonAuthoritative(t *testing.T) {
	repo := &fakeRepository{}
	marker := &fakeMarker{deleteErr: errors.New("temporary marker failure")}
	uc := NewUseCase(repo, marker, time.Now)

	if err := uc.Release(context.Background(), 9, "req-1"); err != nil {
		t.Fatalf("Release() error = %v, want nil", err)
	}
	if repo.releaseCalls != 1 || repo.lastCellID != 9 || repo.lastKey != "req-1" {
		t.Fatalf("Release() args/calls = %d/%d/%q", repo.releaseCalls, repo.lastCellID, repo.lastKey)
	}
	if len(marker.deletes) != 1 || marker.deletes[0].CellID != 9 || marker.deletes[0].ReservationKey != "req-1" {
		t.Fatalf("marker deletes = %+v", marker.deletes)
	}
}

func TestReleaseRejectsBlankKeyAndPropagatesRepositoryConflict(t *testing.T) {
	repo := &fakeRepository{releaseErr: ErrReservationKeyMismatch}
	uc := NewUseCase(repo, nil, time.Now)
	if err := uc.Release(context.Background(), 9, " \t"); !errors.Is(err, ErrIdempotencyKeyRequired) {
		t.Fatalf("Release() blank key error = %v", err)
	}
	if repo.releaseCalls != 0 {
		t.Fatalf("Release() calls = %d, want 0", repo.releaseCalls)
	}
	if err := uc.Release(context.Background(), 9, "req-1"); !errors.Is(err, ErrReservationKeyMismatch) {
		t.Fatalf("Release() conflict error = %v", err)
	}
}

func TestReclaimExpiredDelegatesCleanupContract(t *testing.T) {
	now := time.Date(2026, 7, 24, 2, 0, 0, 0, time.UTC)
	repo := &fakeRepository{reclaimCount: 3}
	uc := NewUseCase(repo, nil, func() time.Time { return now })

	got, err := uc.ReclaimExpired(context.Background(), now, 100)
	if err != nil {
		t.Fatalf("ReclaimExpired() error = %v", err)
	}
	if got != 3 || repo.reclaimCalls != 1 || !repo.lastReclaimAt.Equal(now) || repo.lastBatchSize != 100 {
		t.Fatalf("ReclaimExpired() = %d calls=%d at=%s batch=%d", got, repo.reclaimCalls, repo.lastReclaimAt, repo.lastBatchSize)
	}
}

func TestReservationErrorsAreStableSentinels(t *testing.T) {
	if strings.TrimSpace(ErrIdempotencyKeyRequired.Error()) == "" || strings.TrimSpace(ErrInvalidReservationTTL.Error()) == "" {
		t.Fatal("reservation domain errors must have stable messages")
	}
}
