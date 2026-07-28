package service

import (
	"context"
	"errors"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/order/biz"
)

// stubRepo is a minimal biz.Repository for service-layer tests. The dev-gate
// assertions never reach the repository, so every method just reports a generic
// not-found to make accidental calls obvious.
type stubRepo struct{}

func (stubRepo) CreateOrder(context.Context, *biz.Order) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) GetOrder(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) ListOrders(context.Context, string) ([]*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) MarkOverdue(context.Context, uint64, int64) error { return biz.ErrOrderNotFound }
func (stubRepo) CancelOrder(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) OpenDepositDoor(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) OpenPickupDoor(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) ConfirmDevelopmentPayment(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}
func (stubRepo) SettleOverdue(context.Context, uint64) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}

type stubDoor struct{}

func (stubDoor) OpenDepositDoor(context.Context, string) error { return nil }
func (stubDoor) OpenPickupDoor(context.Context, string) error  { return nil }

func newTestService(appEnv string) *Service {
	uc := biz.NewUseCase(stubRepo{}, stubDoor{}, func() time.Time { return time.Now().UTC() })
	svc, _ := NewService(uc, appEnv)
	return svc
}

func TestDevEndpointsGatedByEnvironment(t *testing.T) {
	production := newTestService("production")
	ctx := context.Background()
	req := &v1.ConfirmDevelopmentPaymentRequest{OrderId: "1"}

	_, err := production.ConfirmDevelopmentPayment(ctx, req)
	if err == nil {
		t.Fatal("ConfirmDevelopmentPayment in production must be rejected")
	}
	if se := new(kratoserrors.Error); !errors.As(err, &se) || se.Reason != "DEV_ENDPOINT_DISABLED" {
		t.Fatalf("ConfirmDevelopmentPayment error = %v, want reason DEV_ENDPOINT_DISABLED", err)
	}

	_, err = production.SettleOverdue(ctx, &v1.SettleOverdueRequest{OrderId: "1"})
	if err == nil {
		t.Fatal("SettleOverdue in production must be rejected")
	}
	if se := new(kratoserrors.Error); !errors.As(err, &se) || se.Reason != "DEV_ENDPOINT_DISABLED" {
		t.Fatalf("SettleOverdue error = %v, want reason DEV_ENDPOINT_DISABLED", err)
	}

	development := newTestService("development")
	if _, err := development.ConfirmDevelopmentPayment(ctx, req); err == nil {
		t.Fatal("ConfirmDevelopmentPayment in development should reach the use case (repo returns not-found here)")
	} else if se := new(kratoserrors.Error); errors.As(err, &se) && se.Reason == "DEV_ENDPOINT_DISABLED" {
		t.Fatalf("dev endpoint must not be gated in development, got %v", err)
	}
}

func TestInvalidInputsMapped(t *testing.T) {
	svc := newTestService("development")
	ctx := context.Background()

	if _, err := svc.CreateOrder(ctx, &v1.CreateOrderRequest{SiteId: "0", Size: v1.CellSize_CELL_SIZE_SMALL, DurationMinutes: 60}); err == nil {
		t.Fatal("zero site_id must be rejected")
	} else if se := new(kratoserrors.Error); !errors.As(err, &se) || se.Reason != "INVALID_SITE_ID" {
		t.Fatalf("CreateOrder error = %v, want INVALID_SITE_ID", err)
	}

	if _, err := svc.GetOrder(ctx, &v1.GetOrderRequest{OrderId: "abc"}); err == nil {
		t.Fatal("non-numeric order_id must be rejected")
	} else if se := new(kratoserrors.Error); !errors.As(err, &se) || se.Reason != "INVALID_ORDER_ID" {
		t.Fatalf("GetOrder error = %v, want INVALID_ORDER_ID", err)
	}

	if _, err := svc.CreateOrder(ctx, &v1.CreateOrderRequest{SiteId: "1", Size: v1.CellSize_CELL_SIZE_UNSPECIFIED, DurationMinutes: 60}); err == nil {
		t.Fatal("unspecified size must be rejected")
	} else if se := new(kratoserrors.Error); !errors.As(err, &se) || se.Reason != "INVALID_CELL_SIZE" {
		t.Fatalf("CreateOrder size error = %v, want INVALID_CELL_SIZE", err)
	}
}
