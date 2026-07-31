package biz

import (
	"errors"
	"time"
)

// DefaultUserID is the singleton owner of every order in this milestone build.
// The project has no authentication system yet, so all orders are attributed to
// a fixed development user. Replace this with the authenticated principal once
// login is introduced.
const DefaultUserID = "dev-user"

// OrderSize mirrors the proto CellSize dimension for a locker cell.
type OrderSize string

const (
	SizeSmall  OrderSize = "SMALL"
	SizeMedium OrderSize = "MEDIUM"
	SizeLarge  OrderSize = "LARGE"
)

// OrderStatus is the canonical order lifecycle status. The values match the
// proto OrderStatus enum exactly and are persisted verbatim in the database.
type OrderStatus string

const (
	StatusPendingPayment  OrderStatus = "PENDING_PAYMENT"
	StatusPaid            OrderStatus = "PAID"
	StatusAwaitingDeposit OrderStatus = "AWAITING_DEPOSIT"
	StatusInStorage       OrderStatus = "IN_STORAGE"
	StatusAwaitingPickup  OrderStatus = "AWAITING_PICKUP"
	StatusOverdue         OrderStatus = "OVERDUE"
	StatusCompleted       OrderStatus = "COMPLETED"
	StatusCancelled       OrderStatus = "CANCELLED"
)

// Per-hour rates in integer fen (分). Floating point is forbidden everywhere.
const (
	RateSmallFenPerHour  = 300
	RateMediumFenPerHour = 500
	RateLargeFenPerHour  = 800
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrSiteNotFound       = errors.New("site not found")
	ErrInvalidSiteID      = errors.New("invalid site ID")
	ErrInvalidOrderID     = errors.New("invalid order ID")
	ErrInvalidCellSize    = errors.New("invalid cell size")
	ErrInvalidDuration    = errors.New("invalid duration")
	ErrNoAvailableCell    = errors.New("no available cell")
	ErrOrderStateConflict = errors.New("order state conflict")
)

// Fee is the immutable cost snapshot captured when an order is created.
type Fee struct {
	RentFeeFen  int64
	DepositFen  int64
	DiscountFen int64
	TotalFen    int64
}

// Order is the domain representation of a parcel locker order.
type Order struct {
	ID              uint64
	OrderNo         string
	UserID          string
	SiteID          uint64
	SiteName        string
	CellID          uint64
	CellNo          string
	Size            OrderSize
	Status          OrderStatus
	DurationMinutes int32
	RentFeeFen      int64
	DepositFen      int64
	DiscountFen     int64
	TotalFen        int64
	OverdueFeeFen   int64
	PaidAt          *time.Time
	DepositedAt     *time.Time
	ExpiresAt       *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StatusLog is an immutable record of every status transition.
type StatusLog struct {
	ID             uint64
	OrderID        uint64
	FromStatus     OrderStatus
	ToStatus       OrderStatus
	Reason         string
	Actor          string
	TraceID        string
	IdempotencyKey string
	CreatedAt      time.Time
}

// allowedTransitions defines the canonical state machine. Any transition not
// present here is illegal and must be rejected with ErrOrderStateConflict.
var allowedTransitions = map[OrderStatus][]OrderStatus{
	StatusPendingPayment:  {StatusPaid, StatusCancelled},
	StatusPaid:            {StatusAwaitingDeposit},
	StatusAwaitingDeposit: {StatusInStorage},
	StatusInStorage:       {StatusOverdue, StatusCompleted},
	StatusOverdue:         {StatusAwaitingPickup},
	StatusAwaitingPickup:  {StatusCompleted},
	StatusCancelled:       {},
	StatusCompleted:       {},
}

// canTransition reports whether moving from -> to is a legal edge.
func canTransition(from, to OrderStatus) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// RateFenPerHour returns the hourly rate for a size in fen.
func RateFenPerHour(size OrderSize) int64 {
	switch size {
	case SizeSmall:
		return RateSmallFenPerHour
	case SizeMedium:
		return RateMediumFenPerHour
	case SizeLarge:
		return RateLargeFenPerHour
	default:
		return 0
	}
}

// ComputeFee calculates the cost snapshot from the size and rental duration.
// rent = hourly rate * ceil(duration_minutes / 60); deposit and discount are 0
// for this milestone, so total equals rent. All values are integer fen.
func ComputeFee(size OrderSize, durationMinutes int32) Fee {
	rate := RateFenPerHour(size)
	hours := int64(0)
	if durationMinutes > 0 {
		// ceil(duration / 60) without floating point.
		hours = int64((int64(durationMinutes) + 59) / 60)
	}
	rent := rate * hours
	return Fee{
		RentFeeFen:  rent,
		DepositFen:  0,
		DiscountFen: 0,
		TotalFen:    rent,
	}
}

// computeOverdueFee returns the penalty for an order that exceeded expires_at.
// It charges for every hour that has begun since expiry, with a one-hour
// minimum, all in integer fen.
func computeOverdueFee(size OrderSize, expiresAt, now time.Time) int64 {
	rate := RateFenPerHour(size)
	elapsed := now.Sub(expiresAt)
	if elapsed < 0 {
		elapsed = 0
	}
	hours := int64((elapsed + time.Hour - 1) / time.Hour)
	if hours < 1 {
		hours = 1
	}
	return rate * hours
}
