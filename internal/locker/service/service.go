package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

type UseCase interface {
	Reserve(context.Context, biz.ReserveRequest) (biz.Reservation, error)
	Release(context.Context, uint64, string) error
}

type Service struct{ uc UseCase }

var _ v1.InternalLockerServiceHTTPServer = (*Service)(nil)

func NewService(uc UseCase) *Service { return &Service{uc: uc} }

func (s *Service) ReserveCell(ctx context.Context, req *v1.ReserveCellRequest) (*v1.ReserveCellReply, error) {
	if req == nil {
		req = &v1.ReserveCellRequest{}
	}
	siteID, err := parseID(req.SiteId, "INVALID_SITE_ID", "invalid site ID")
	if err != nil {
		return nil, err
	}
	size, err := toCellSize(req.Size)
	if err != nil {
		return nil, err
	}
	if req.TtlSeconds < 0 {
		return nil, kratoserrors.BadRequest("INVALID_RESERVATION_TTL", "invalid reservation TTL")
	}
	reservation, err := s.uc.Reserve(ctx, biz.ReserveRequest{
		SiteID:         siteID,
		Size:           size,
		TTL:            time.Duration(req.TtlSeconds) * time.Second,
		ReservationKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapLockerError(err)
	}
	return &v1.ReserveCellReply{
		CellId:    strconv.FormatUint(reservation.CellID, 10),
		DeviceNo:  reservation.DeviceNo,
		CellNo:    reservation.CellNo,
		ExpiresAt: reservation.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Service) ReleaseCell(ctx context.Context, req *v1.ReleaseCellRequest) (*v1.ReleaseCellReply, error) {
	if req == nil {
		req = &v1.ReleaseCellRequest{}
	}
	cellID, err := parseID(req.CellId, "INVALID_CELL_ID", "invalid cell ID")
	if err != nil {
		return nil, err
	}
	if err := s.uc.Release(ctx, cellID, req.IdempotencyKey); err != nil {
		return nil, mapLockerError(err)
	}
	return &v1.ReleaseCellReply{Released: true}, nil
}

func parseID(raw, reason, message string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, kratoserrors.BadRequest(reason, message)
	}
	return id, nil
}

func toCellSize(size v1.CellSize) (biz.CellSize, error) {
	switch size {
	case v1.CellSize_CELL_SIZE_SMALL:
		return biz.SizeSmall, nil
	case v1.CellSize_CELL_SIZE_MEDIUM:
		return biz.SizeMedium, nil
	case v1.CellSize_CELL_SIZE_LARGE:
		return biz.SizeLarge, nil
	default:
		return "", kratoserrors.BadRequest("INVALID_CELL_SIZE", "invalid cell size")
	}
}

func mapLockerError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return kratoserrors.ClientClosed("REQUEST_CANCELED", "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return kratoserrors.GatewayTimeout("REQUEST_DEADLINE_EXCEEDED", "request deadline exceeded")
	case errors.Is(err, biz.ErrIdempotencyKeyRequired):
		return kratoserrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "idempotency key required")
	case errors.Is(err, biz.ErrInvalidReservationTTL):
		return kratoserrors.BadRequest("INVALID_RESERVATION_TTL", "invalid reservation TTL")
	case errors.Is(err, biz.ErrInvalidCellID):
		return kratoserrors.BadRequest("INVALID_CELL_ID", "invalid cell ID")
	case errors.Is(err, biz.ErrReservationNotFound):
		return kratoserrors.NotFound("RESERVATION_NOT_FOUND", "reservation not found")
	case errors.Is(err, biz.ErrDeviceOffline):
		return kratoserrors.ServiceUnavailable("DEVICE_OFFLINE", "device offline")
	case errors.Is(err, biz.ErrCellNotAvailable):
		return kratoserrors.Conflict("CELL_NOT_AVAILABLE", "cell not available")
	case errors.Is(err, biz.ErrReservationKeyMismatch):
		return kratoserrors.Conflict("CELL_RESERVATION_CONFLICT", "cell reservation conflict")
	default:
		return kratoserrors.ServiceUnavailable("LOCKER_DATA_UNAVAILABLE", "locker data unavailable")
	}
}
