package biz

import (
	"context"
	"errors"
	"strings"
	"time"
)

type UseCase struct {
	repo   Repository
	marker Marker
	now    func() time.Time
}

func NewUseCase(repo Repository, marker Marker, now func() time.Time) *UseCase {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &UseCase{repo: repo, marker: marker, now: now}
}

func (uc *UseCase) Reserve(ctx context.Context, req ReserveRequest) (Reservation, error) {
	if strings.TrimSpace(req.ReservationKey) == "" {
		return Reservation{}, ErrIdempotencyKeyRequired
	}

	ttl := req.TTL
	if ttl == 0 {
		ttl = DefaultReservationTTL
	}
	if ttl < MinReservationTTL || ttl > MaxReservationTTL {
		return Reservation{}, ErrInvalidReservationTTL
	}
	req.TTL = ttl

	existing, err := uc.repo.FindReservation(ctx, req.ReservationKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrReservationNotFound) {
		return Reservation{}, err
	}

	reservation, err := uc.repo.ReserveAvailable(ctx, req, uc.now().Add(ttl))
	if err != nil {
		return Reservation{}, err
	}
	if uc.marker != nil {
		_ = uc.marker.Mark(ctx, reservation)
	}
	return reservation, nil
}

func (uc *UseCase) Release(ctx context.Context, cellID uint64, reservationKey string) error {
	if cellID == 0 {
		return ErrInvalidCellID
	}
	if strings.TrimSpace(reservationKey) == "" {
		return ErrIdempotencyKeyRequired
	}

	if err := uc.repo.Release(ctx, cellID, reservationKey); err != nil {
		return err
	}
	if uc.marker != nil {
		_ = uc.marker.Delete(ctx, Reservation{CellID: cellID, ReservationKey: reservationKey})
	}
	return nil
}

func (uc *UseCase) ReclaimExpired(ctx context.Context, now time.Time, batchSize int) (int, error) {
	return uc.repo.ReclaimExpired(ctx, now, batchSize)
}
