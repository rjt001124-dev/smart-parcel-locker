package biz

import (
	"context"
	"math/rand"
	"time"
)

// UseCase holds the order business logic. It validates transitions and computes
// fees; all persistence and locker-cell mutation happens inside Repository.
type UseCase struct {
	repo Repository
	door DoorGateway
	now  func() time.Time
}

func NewUseCase(repo Repository, door DoorGateway, now func() time.Time) *UseCase {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &UseCase{repo: repo, door: door, now: now}
}

// CreateOrder reserves a cell and persists a PENDING_PAYMENT order.
func (uc *UseCase) CreateOrder(ctx context.Context, siteID uint64, size OrderSize, durationMinutes int32) (*Order, error) {
	if siteID == 0 {
		return nil, ErrInvalidSiteID
	}
	if !validSize(size) {
		return nil, ErrInvalidCellSize
	}
	if durationMinutes <= 0 {
		return nil, ErrInvalidDuration
	}

	fee := ComputeFee(size, durationMinutes)
	order := &Order{
		OrderNo:         generateOrderNo(uc.now()),
		UserID:          DefaultUserID,
		SiteID:          siteID,
		Size:            size,
		Status:          StatusPendingPayment,
		DurationMinutes: durationMinutes,
		RentFeeFen:      fee.RentFeeFen,
		DepositFen:      fee.DepositFen,
		DiscountFen:     fee.DiscountFen,
		TotalFen:        fee.TotalFen,
	}
	return uc.repo.CreateOrder(ctx, order)
}

// GetOrder loads an order and lazily flips it to OVERDUE when it has expired.
func (uc *UseCase) GetOrder(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	uc.applyLazyOverdue(ctx, order)
	return order, nil
}

// ListOrders returns all of a user's orders, newest first, applying the lazy
// overdue transition where applicable.
func (uc *UseCase) ListOrders(ctx context.Context, userID string) ([]*Order, error) {
	orders, err := uc.repo.ListOrders(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		uc.applyLazyOverdue(ctx, order)
	}
	return orders, nil
}

// CancelOrder releases the reserved cell and moves the order to CANCELLED.
func (uc *UseCase) CancelOrder(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	return uc.repo.CancelOrder(ctx, orderID)
}

// OpenDepositDoor opens the deposit door; on success the order becomes
// IN_STORAGE and the cell is occupied.
func (uc *UseCase) OpenDepositDoor(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != StatusAwaitingDeposit {
		return nil, ErrOrderStateConflict
	}
	if err := uc.door.OpenDepositDoor(ctx, order.CellNo); err != nil {
		return nil, err
	}
	return uc.repo.OpenDepositDoor(ctx, orderID)
}

// OpenPickupDoor opens the pickup door; on success the order becomes COMPLETED
// and the cell is released. Only the non-overdue IN_STORAGE or AWAITING_PICKUP
// states are valid; an expired IN_STORAGE order must settle first.
func (uc *UseCase) OpenPickupDoor(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	switch order.Status {
	case StatusInStorage, StatusAwaitingPickup:
		// valid
	default:
		return nil, ErrOrderStateConflict
	}
	if err := uc.door.OpenPickupDoor(ctx, order.CellNo); err != nil {
		return nil, err
	}
	return uc.repo.OpenPickupDoor(ctx, orderID)
}

// ConfirmDevelopmentPayment advances PENDING_PAYMENT -> PAID -> AWAITING_DEPOSIT.
// Callers must enforce the development-only gate before invoking this.
func (uc *UseCase) ConfirmDevelopmentPayment(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	return uc.repo.ConfirmDevelopmentPayment(ctx, orderID)
}

// SettleOverdue moves an OVERDUE order to AWAITING_PICKUP. Callers must enforce
// the development-only gate before invoking this. The order is first reloaded so
// the lazy IN_STORAGE -> OVERDUE transition is applied if it has expired.
func (uc *UseCase) SettleOverdue(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, ErrInvalidOrderID
	}
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	uc.applyLazyOverdue(ctx, order)
	return uc.repo.SettleOverdue(ctx, orderID)
}

// applyLazyOverdue transitions an IN_STORAGE order that has passed its expiry
// into OVERDUE, computing the integer-fen penalty, without touching a cell.
func (uc *UseCase) applyLazyOverdue(ctx context.Context, order *Order) {
	if order == nil || order.Status != StatusInStorage || order.ExpiresAt == nil {
		return
	}
	now := uc.now()
	if !now.After(*order.ExpiresAt) {
		return
	}
	fee := computeOverdueFee(order.Size, *order.ExpiresAt, now)
	if err := uc.repo.MarkOverdue(ctx, order.ID, fee); err == nil {
		order.Status = StatusOverdue
		order.OverdueFeeFen = fee
	}
}

func validSize(size OrderSize) bool {
	switch size {
	case SizeSmall, SizeMedium, SizeLarge:
		return true
	default:
		return false
	}
}

// generateOrderNo builds a unique order number: SPL + millisecond timestamp +
// random suffix. The millisecond component plus randomness keeps collisions
// effectively impossible within a single deployment.
func generateOrderNo(now time.Time) string {
	return "SPL" + formatTimestamp(now) + randomSuffix()
}

func formatTimestamp(now time.Time) string {
	ms := now.UnixNano() / int64(time.Millisecond)
	return itoa(ms)
}

func randomSuffix() string {
	const digits = "0123456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = digits[rand.Intn(len(digits))]
	}
	return string(b)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
