package service

import (
	"context"
	"errors"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

type lockerUseCaseStub struct {
	reserveRequest biz.ReserveRequest
	reservation    biz.Reservation
	reserveErr     error
	releaseCellID  uint64
	releaseKey     string
	releaseErr     error
}

func (s *lockerUseCaseStub) Reserve(_ context.Context, req biz.ReserveRequest) (biz.Reservation, error) {
	s.reserveRequest = req
	return s.reservation, s.reserveErr
}

func (s *lockerUseCaseStub) Release(_ context.Context, cellID uint64, key string) error {
	s.releaseCellID, s.releaseKey = cellID, key
	return s.releaseErr
}

func TestReserveCellForwardsTTLAndIdempotency(t *testing.T) {
	expiresAt := time.Date(2026, 7, 24, 12, 5, 0, 0, time.UTC)
	uc := &lockerUseCaseStub{reservation: biz.Reservation{CellID: 7, DeviceNo: "DEV-1", CellNo: "B01", ExpiresAt: expiresAt}}
	service := NewService(uc)
	reply, err := service.ReserveCell(context.Background(), &v1.ReserveCellRequest{SiteId: "3", Size: v1.CellSize_CELL_SIZE_MEDIUM, TtlSeconds: 120, IdempotencyKey: "reserve-1"})
	if err != nil {
		t.Fatal(err)
	}
	if uc.reserveRequest.SiteID != 3 || uc.reserveRequest.Size != biz.SizeMedium || uc.reserveRequest.TTL != 2*time.Minute || uc.reserveRequest.ReservationKey != "reserve-1" {
		t.Fatalf("reserve request = %+v", uc.reserveRequest)
	}
	if reply.CellId != "7" || reply.DeviceNo != "DEV-1" || reply.CellNo != "B01" || reply.ExpiresAt != expiresAt.Format(time.RFC3339Nano) {
		t.Fatalf("reply = %+v", reply)
	}
}

func TestReleaseCellParsesIDAndMapsConflict(t *testing.T) {
	uc := &lockerUseCaseStub{releaseErr: biz.ErrReservationKeyMismatch}
	service := NewService(uc)
	_, err := service.ReleaseCell(context.Background(), &v1.ReleaseCellRequest{CellId: "9", IdempotencyKey: "reserve-9"})
	if uc.releaseCellID != 9 || uc.releaseKey != "reserve-9" {
		t.Fatalf("release input = (%d, %q)", uc.releaseCellID, uc.releaseKey)
	}
	public := kratoserrors.FromError(err)
	if public.Code != 409 || public.Reason != "CELL_RESERVATION_CONFLICT" {
		t.Fatalf("error = %+v", public)
	}
}

func TestReserveCellMapsStableErrors(t *testing.T) {
	service := NewService(&lockerUseCaseStub{reserveErr: errors.New("secret mysql dsn")})
	_, err := service.ReserveCell(context.Background(), &v1.ReserveCellRequest{SiteId: "1", Size: v1.CellSize_CELL_SIZE_SMALL, IdempotencyKey: "key"})
	public := kratoserrors.FromError(err)
	if public.Code != 503 || public.Reason != "LOCKER_DATA_UNAVAILABLE" {
		t.Fatalf("error = %+v", public)
	}
}
