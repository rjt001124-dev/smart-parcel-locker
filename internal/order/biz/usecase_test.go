package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeDoor is a no-op DoorGateway for unit tests.
type fakeDoor struct{}

func (fakeDoor) OpenDepositDoor(context.Context, string) error { return nil }
func (fakeDoor) OpenPickupDoor(context.Context, string) error  { return nil }

// fakeRepo is an in-memory biz.Repository used to exercise the state machine
// without a database.
type fakeRepo struct {
	seq    uint64
	orders map[uint64]*Order
	now    func() time.Time
}

func newFakeRepo(now func() time.Time) *fakeRepo {
	return &fakeRepo{orders: make(map[uint64]*Order), now: now}
}

func (f *fakeRepo) nextID() uint64 {
	f.seq++
	return f.seq
}

func (f *fakeRepo) CreateOrder(_ context.Context, order *Order) (*Order, error) {
	id := f.nextID()
	order.ID = id
	order.CellID = 1000 + id
	order.CellNo = "A0" + itoa(int64(id))
	if order.CreatedAt.IsZero() {
		order.CreatedAt = f.now()
	}
	order.UpdatedAt = f.now()
	cp := *order
	f.orders[id] = &cp
	return order, nil
}

func (f *fakeRepo) GetOrder(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cp := *o
	return &cp, nil
}

func (f *fakeRepo) ListOrders(_ context.Context, userID string) ([]*Order, error) {
	out := make([]*Order, 0)
	for _, o := range f.orders {
		if o.UserID == userID {
			cp := *o
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkOverdue(_ context.Context, id uint64, fee int64) error {
	o, ok := f.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	o.Status = StatusOverdue
	o.OverdueFeeFen = fee
	return nil
}

func (f *fakeRepo) CancelOrder(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if o.Status != StatusPendingPayment {
		return nil, ErrOrderStateConflict
	}
	o.Status = StatusCancelled
	return o, nil
}

func (f *fakeRepo) OpenDepositDoor(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if o.Status != StatusAwaitingDeposit {
		return nil, ErrOrderStateConflict
	}
	now := f.now()
	o.Status = StatusInStorage
	o.DepositedAt = &now
	exp := now.Add(time.Duration(o.DurationMinutes) * time.Minute)
	o.ExpiresAt = &exp
	return o, nil
}

func (f *fakeRepo) OpenPickupDoor(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	switch o.Status {
	case StatusAwaitingPickup:
	case StatusInStorage:
		if o.ExpiresAt != nil && f.now().After(*o.ExpiresAt) {
			return nil, ErrOrderStateConflict
		}
	default:
		return nil, ErrOrderStateConflict
	}
	now := f.now()
	o.Status = StatusCompleted
	o.CompletedAt = &now
	return o, nil
}

func (f *fakeRepo) ConfirmDevelopmentPayment(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if o.Status != StatusPendingPayment {
		return nil, ErrOrderStateConflict
	}
	now := f.now()
	o.Status = StatusAwaitingDeposit
	o.PaidAt = &now
	return o, nil
}

func (f *fakeRepo) SettleOverdue(_ context.Context, id uint64) (*Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if o.Status != StatusOverdue {
		return nil, ErrOrderStateConflict
	}
	o.Status = StatusAwaitingPickup
	o.OverdueFeeFen = 0
	return o, nil
}

func newTestUseCase(now time.Time) (*UseCase, *fakeRepo) {
	repo := newFakeRepo(func() time.Time { return now })
	return NewUseCase(repo, fakeDoor{}, func() time.Time { return now }), repo
}

func TestComputeFeeIntegerFen(t *testing.T) {
	cases := []struct {
		size     OrderSize
		duration int32
		want     int64
	}{
		{SizeSmall, 60, 300},
		{SizeSmall, 61, 600}, // ceil(61/60) = 2 hours
		{SizeSmall, 120, 600},
		{SizeSmall, 1, 300}, // minimum one hour
		{SizeMedium, 60, 500},
		{SizeLarge, 90, 1600}, // 2 * 800
		{SizeLarge, 0, 0},     // no duration, no charge
	}
	for _, c := range cases {
		got := ComputeFee(c.size, c.duration)
		if got.RentFeeFen != c.want {
			t.Fatalf("ComputeFee(%s,%d).RentFeeFen = %d, want %d", c.size, c.duration, got.RentFeeFen, c.want)
		}
		if got.TotalFen != got.RentFeeFen {
			t.Fatalf("TotalFen = %d, want %d (total must equal rent in milestone)", got.TotalFen, got.RentFeeFen)
		}
		if got.DepositFen != 0 || got.DiscountFen != 0 {
			t.Fatalf("deposit/discount must be 0, got deposit=%d discount=%d", got.DepositFen, got.DiscountFen)
		}
	}
}

func TestCreateOrderInitialState(t *testing.T) {
	uc, _ := newTestUseCase(time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC))
	order, err := uc.CreateOrder(context.Background(), 1, SizeMedium, 120)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order.Status != StatusPendingPayment {
		t.Fatalf("Status = %s, want PENDING_PAYMENT", order.Status)
	}
	if order.UserID != DefaultUserID {
		t.Fatalf("UserID = %q, want %q", order.UserID, DefaultUserID)
	}
	if order.RentFeeFen != 1000 {
		t.Fatalf("RentFeeFen = %d, want 1000", order.RentFeeFen)
	}
	if order.TotalFen != 1000 {
		t.Fatalf("TotalFen = %d, want 1000", order.TotalFen)
	}
	if order.OrderNo == "" {
		t.Fatalf("OrderNo must be generated")
	}
}

func TestCreateOrderValidation(t *testing.T) {
	uc, _ := newTestUseCase(time.Now())
	if _, err := uc.CreateOrder(context.Background(), 0, SizeSmall, 60); !errors.Is(err, ErrInvalidSiteID) {
		t.Fatalf("zero site error = %v, want ErrInvalidSiteID", err)
	}
	if _, err := uc.CreateOrder(context.Background(), 1, OrderSize("XL"), 60); !errors.Is(err, ErrInvalidCellSize) {
		t.Fatalf("bad size error = %v, want ErrInvalidCellSize", err)
	}
	if _, err := uc.CreateOrder(context.Background(), 1, SizeSmall, 0); !errors.Is(err, ErrInvalidDuration) {
		t.Fatalf("zero duration error = %v, want ErrInvalidDuration", err)
	}
}

func TestIllegalTransitionsRejected(t *testing.T) {
	uc, repo := newTestUseCase(time.Now())

	pending, _ := uc.CreateOrder(context.Background(), 1, SizeSmall, 60)

	// Picking up before payment/deposit is illegal.
	if _, err := uc.OpenPickupDoor(context.Background(), pending.ID); !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("OpenPickupDoor(pending) error = %v, want ErrOrderStateConflict", err)
	}
	// Settling a not-yet-overdue order is illegal.
	if _, err := uc.SettleOverdue(context.Background(), pending.ID); !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("SettleOverdue(pending) error = %v, want ErrOrderStateConflict", err)
	}
	// Opening deposit door before dev payment is illegal.
	if _, err := uc.OpenDepositDoor(context.Background(), pending.ID); !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("OpenDepositDoor(pending) error = %v, want ErrOrderStateConflict", err)
	}
	// Cancelling twice is illegal.
	if _, err := uc.CancelOrder(context.Background(), pending.ID); err != nil {
		t.Fatalf("first CancelOrder() error = %v", err)
	}
	if _, err := uc.CancelOrder(context.Background(), pending.ID); !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("second CancelOrder() error = %v, want ErrOrderStateConflict", err)
	}
	_ = repo
}

func TestHappyPathLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	uc, _ := newTestUseCase(now)

	order, err := uc.CreateOrder(context.Background(), 1, SizeLarge, 60)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	paid, err := uc.ConfirmDevelopmentPayment(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("ConfirmDevelopmentPayment() error = %v", err)
	}
	if paid.Status != StatusAwaitingDeposit || paid.PaidAt == nil {
		t.Fatalf("after payment: status=%s paidAt=%v, want AWAITING_DEPOSIT set", paid.Status, paid.PaidAt)
	}
	deposited, err := uc.OpenDepositDoor(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("OpenDepositDoor() error = %v", err)
	}
	if deposited.Status != StatusInStorage || deposited.DepositedAt == nil || deposited.ExpiresAt == nil {
		t.Fatalf("after deposit: status=%s depositedAt=%v expiresAt=%v", deposited.Status, deposited.DepositedAt, deposited.ExpiresAt)
	}
	if !deposited.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expiresAt = %v, want %v", deposited.ExpiresAt, now.Add(time.Hour))
	}
	pickedUp, err := uc.OpenPickupDoor(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("OpenPickupDoor() error = %v", err)
	}
	if pickedUp.Status != StatusCompleted || pickedUp.CompletedAt == nil {
		t.Fatalf("after pickup: status=%s completedAt=%v", pickedUp.Status, pickedUp.CompletedAt)
	}
}

func TestLazyOverdueTransition(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	uc, repo := newTestUseCase(now)

	// Seed an IN_STORAGE order that expired 90 minutes ago.
	expiredAt := now.Add(-90 * time.Minute)
	seed := &Order{
		ID:              42,
		OrderNo:         "SPLSEED",
		UserID:          DefaultUserID,
		SiteID:          1,
		CellID:          99,
		CellNo:          "Z01",
		Size:            SizeSmall,
		Status:          StatusInStorage,
		DurationMinutes: 60,
		RentFeeFen:      300,
		TotalFen:        300,
		ExpiresAt:       &expiredAt,
		CreatedAt:       now.Add(-2 * time.Hour),
		UpdatedAt:       now.Add(-2 * time.Hour),
	}
	repo.orders[seed.ID] = seed

	got, err := uc.GetOrder(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if got.Status != StatusOverdue {
		t.Fatalf("Status = %s, want OVERDUE after lazy transition", got.Status)
	}
	// 90 minutes elapsed => ceil(90/60) = 2 hours @ 300 fen/hour = 600.
	if got.OverdueFeeFen != 600 {
		t.Fatalf("OverdueFeeFen = %d, want 600", got.OverdueFeeFen)
	}

	// A second read must keep it OVERDUE and stable (idempotent).
	again, err := uc.GetOrder(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("GetOrder() second error = %v", err)
	}
	if again.Status != StatusOverdue || again.OverdueFeeFen != 600 {
		t.Fatalf("second read status=%s fee=%d, want OVERDUE 600", again.Status, again.OverdueFeeFen)
	}
}

func TestLazyOverdueMinimumOneHour(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	uc, repo := newTestUseCase(now)

	expiredAt := now.Add(-30 * time.Minute) // only 30 minutes late => 1 hour minimum
	seed := &Order{
		ID:        7,
		OrderNo:   "SPLMIN",
		UserID:    DefaultUserID,
		SiteID:    1,
		CellID:    8,
		CellNo:    "M01",
		Size:      SizeMedium,
		Status:    StatusInStorage,
		ExpiresAt: &expiredAt,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Hour),
	}
	repo.orders[seed.ID] = seed

	got, err := uc.GetOrder(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if got.Status != StatusOverdue {
		t.Fatalf("Status = %s, want OVERDUE", got.Status)
	}
	// 30 minutes late => minimum 1 hour @ 500 fen/hour = 500.
	if got.OverdueFeeFen != 500 {
		t.Fatalf("OverdueFeeFen = %d, want 500 (1-hour minimum)", got.OverdueFeeFen)
	}
}

func TestOverdueBlocksPickupUntilSettled(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	uc, repo := newTestUseCase(now)

	expiredAt := now.Add(-time.Hour)
	seed := &Order{
		ID:        11,
		OrderNo:   "SPLBLK",
		UserID:    DefaultUserID,
		SiteID:    1,
		CellID:    12,
		CellNo:    "B01",
		Size:      SizeSmall,
		Status:    StatusInStorage,
		ExpiresAt: &expiredAt,
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	}
	repo.orders[seed.ID] = seed

	if _, err := uc.OpenPickupDoor(context.Background(), seed.ID); !errors.Is(err, ErrOrderStateConflict) {
		t.Fatalf("OpenPickupDoor(overdue) error = %v, want ErrOrderStateConflict", err)
	}
	settled, err := uc.SettleOverdue(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("SettleOverdue() error = %v", err)
	}
	if settled.Status != StatusAwaitingPickup {
		t.Fatalf("after settle status = %s, want AWAITING_PICKUP", settled.Status)
	}
	if settled.OverdueFeeFen != 0 {
		t.Fatalf("OverdueFeeFen after settle = %d, want 0", settled.OverdueFeeFen)
	}
	if _, err := uc.OpenPickupDoor(context.Background(), seed.ID); err != nil {
		t.Fatalf("OpenPickupDoor(after settle) error = %v", err)
	}
}
