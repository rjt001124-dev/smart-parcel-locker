package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/order/biz"
)

// devEnvironment is the only APP_ENV value that permits the development-only
// payment and overdue settlement endpoints.
const devEnvironment = "development"

type Service struct {
	uc     *biz.UseCase
	appEnv string
}

var _ v1.OrderServiceHTTPServer = (*Service)(nil)

func NewService(uc *biz.UseCase, appEnv string) (*Service, error) {
	if uc == nil {
		return nil, errors.New("order use case is required")
	}
	return &Service{uc: uc, appEnv: appEnv}, nil
}

func (s *Service) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.OrderReply, error) {
	if req == nil {
		req = &v1.CreateOrderRequest{}
	}
	siteID, err := parseID(req.SiteId, "site ID")
	if err != nil {
		return nil, err
	}
	size, err := toBizSize(req.Size)
	if err != nil {
		return nil, err
	}
	if req.DurationMinutes <= 0 {
		return nil, kratoserrors.BadRequest("INVALID_DURATION", "duration must be positive")
	}
	order, err := s.uc.CreateOrder(ctx, siteID, size, req.DurationMinutes)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

func (s *Service) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.OrderReply, error) {
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.GetOrder(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

func (s *Service) ListOrders(ctx context.Context, _ *v1.ListOrdersRequest) (*v1.ListOrdersReply, error) {
	orders, err := s.uc.ListOrders(ctx, biz.DefaultUserID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	reply := &v1.ListOrdersReply{Orders: make([]*v1.Order, 0, len(orders))}
	for _, order := range orders {
		reply.Orders = append(reply.Orders, toProtoOrder(order))
	}
	return reply, nil
}

func (s *Service) CancelOrder(ctx context.Context, req *v1.CancelOrderRequest) (*v1.OrderReply, error) {
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.CancelOrder(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

func (s *Service) OpenDepositDoor(ctx context.Context, req *v1.OpenDepositDoorRequest) (*v1.OrderReply, error) {
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.OpenDepositDoor(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

func (s *Service) OpenPickupDoor(ctx context.Context, req *v1.OpenPickupDoorRequest) (*v1.OrderReply, error) {
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.OpenPickupDoor(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

// ConfirmDevelopmentPayment is a development-only simulator and is rejected
// unless APP_ENV=development.
func (s *Service) ConfirmDevelopmentPayment(ctx context.Context, req *v1.ConfirmDevelopmentPaymentRequest) (*v1.OrderReply, error) {
	if s.appEnv != devEnvironment {
		return nil, kratoserrors.Forbidden("DEV_ENDPOINT_DISABLED", "development-only payment endpoint is disabled in this environment")
	}
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.ConfirmDevelopmentPayment(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

// SettleOverdue is a development-only simulator and is rejected unless
// APP_ENV=development.
func (s *Service) SettleOverdue(ctx context.Context, req *v1.SettleOverdueRequest) (*v1.OrderReply, error) {
	if s.appEnv != devEnvironment {
		return nil, kratoserrors.Forbidden("DEV_ENDPOINT_DISABLED", "development-only overdue settlement endpoint is disabled in this environment")
	}
	orderID, err := parseOrderID(req)
	if err != nil {
		return nil, err
	}
	order, err := s.uc.SettleOverdue(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.OrderReply{Order: toProtoOrder(order)}, nil
}

func parseOrderID(req interface{ GetOrderId() string }) (uint64, error) {
	return parseID(req.GetOrderId(), "order ID")
}

func parseID(value, label string) (uint64, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, kratoserrors.BadRequest("INVALID_"+normalizeLabel(label), "invalid "+label)
	}
	return id, nil
}

func normalizeLabel(label string) string {
	switch label {
	case "order ID":
		return "ORDER_ID"
	case "site ID":
		return "SITE_ID"
	default:
		return "ID"
	}
}

func toBizSize(size v1.CellSize) (biz.OrderSize, error) {
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

func toProtoSize(size biz.OrderSize) v1.CellSize {
	switch size {
	case biz.SizeSmall:
		return v1.CellSize_CELL_SIZE_SMALL
	case biz.SizeMedium:
		return v1.CellSize_CELL_SIZE_MEDIUM
	case biz.SizeLarge:
		return v1.CellSize_CELL_SIZE_LARGE
	default:
		return v1.CellSize_CELL_SIZE_UNSPECIFIED
	}
}

func toProtoStatus(status biz.OrderStatus) v1.OrderStatus {
	switch status {
	case biz.StatusPendingPayment:
		return v1.OrderStatus_ORDER_STATUS_PENDING_PAYMENT
	case biz.StatusPaid:
		return v1.OrderStatus_ORDER_STATUS_PAID
	case biz.StatusAwaitingDeposit:
		return v1.OrderStatus_ORDER_STATUS_AWAITING_DEPOSIT
	case biz.StatusInStorage:
		return v1.OrderStatus_ORDER_STATUS_IN_STORAGE
	case biz.StatusAwaitingPickup:
		return v1.OrderStatus_ORDER_STATUS_AWAITING_PICKUP
	case biz.StatusOverdue:
		return v1.OrderStatus_ORDER_STATUS_OVERDUE
	case biz.StatusCompleted:
		return v1.OrderStatus_ORDER_STATUS_COMPLETED
	case biz.StatusCancelled:
		return v1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return v1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func toProtoOrder(o *biz.Order) *v1.Order {
	if o == nil {
		return nil
	}
	return &v1.Order{
		Id:              strconv.FormatUint(o.ID, 10),
		OrderNo:         o.OrderNo,
		SiteId:          strconv.FormatUint(o.SiteID, 10),
		SiteName:        o.SiteName,
		CellId:          strconv.FormatUint(o.CellID, 10),
		CellNo:          o.CellNo,
		Size:            toProtoSize(o.Size),
		Status:          toProtoStatus(o.Status),
		DurationMinutes: o.DurationMinutes,
		FeeSnapshot: &v1.FeeSnapshot{
			RentFeeFen:  o.RentFeeFen,
			DepositFen:  o.DepositFen,
			DiscountFen: o.DiscountFen,
			TotalFen:    o.TotalFen,
		},
		OverdueFeeFen: o.OverdueFeeFen,
		CreatedAt:     formatTime(o.CreatedAt),
		PaidAt:        formatTimePtr(o.PaidAt),
		DepositedAt:   formatTimePtr(o.DepositedAt),
		ExpiresAt:     formatTimePtr(o.ExpiresAt),
		CompletedAt:   formatTimePtr(o.CompletedAt),
	}
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func mapOrderError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return kratoserrors.ClientClosed("REQUEST_CANCELED", "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return kratoserrors.GatewayTimeout("REQUEST_DEADLINE_EXCEEDED", "request deadline exceeded")
	case errors.Is(err, biz.ErrOrderNotFound):
		return kratoserrors.NotFound("ORDER_NOT_FOUND", "order not found")
	case errors.Is(err, biz.ErrSiteNotFound):
		return kratoserrors.NotFound("SITE_NOT_FOUND", "site not found")
	case errors.Is(err, biz.ErrInvalidSiteID), errors.Is(err, biz.ErrInvalidOrderID), errors.Is(err, biz.ErrInvalidCellSize), errors.Is(err, biz.ErrInvalidDuration):
		return kratoserrors.BadRequest("INVALID_ORDER_INPUT", "invalid order input")
	case errors.Is(err, biz.ErrNoAvailableCell):
		return kratoserrors.Conflict("NO_AVAILABLE_CELL", "no available locker cell")
	case errors.Is(err, biz.ErrOrderStateConflict):
		return kratoserrors.Conflict("ORDER_STATE_CONFLICT", "illegal order state transition")
	default:
		return kratoserrors.ServiceUnavailable("ORDER_DATA_UNAVAILABLE", "order data unavailable")
	}
}
